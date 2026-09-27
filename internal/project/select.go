package project

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Block は runbook の runnora: ブロックのうち、手順書に使う項目。
type Block struct {
	ID     string   `yaml:"id"`
	Before []string `yaml:"before"`
	After  []string `yaml:"after"`
	Envs   []string `yaml:"envs"`
}

// AllowedIn は環境 env で実行する runbook かを返す。envs がなければすべての環境で実行する。
func (b *Block) AllowedIn(env string) bool {
	if b == nil || len(b.Envs) == 0 || env == "" {
		return true
	}
	for _, e := range b.Envs {
		if e == env {
			return true
		}
	}
	return false
}

// ReadBlock は runbook の runnora: ブロックを読む。ブロックがなければ nil。
func ReadBlock(path string) (*Block, []string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("runbook を読めません: %w", err)
	}
	var head struct {
		Labels  []string `yaml:"labels"`
		Runnora *Block   `yaml:"runnora"`
	}
	if err := yaml.Unmarshal(data, &head); err != nil {
		return nil, nil, fmt.Errorf("runbook YAML が不正です: %s: %w", path, err)
	}
	if head.Runnora != nil && head.Runnora.ID == "" {
		return nil, nil, fmt.Errorf("%s: runnora: ブロックに id がありません", path)
	}
	return head.Runnora, head.Labels, nil
}

// Selected はスイートが選んだ runbook。
type Selected struct {
	Path  string
	Block *Block
}

// Select はスイートの条件に合う runbook を返す (runnora run --suite と同じ規則)。
//
//   - paths の glob (プロジェクトルート基準、** 可) に一致し、runnora: ブロックを持つ runbook だけを選ぶ。
//   - labels を指定すると、どれか 1 つを持つものに絞る。
//   - ids を指定すると、その ID のものに絞り ids の順に並べる。見つからない ID はエラー。
//   - ids がなければパスの順に並べる。
func (p *Project) Select(sel Selection) ([]Selected, error) {
	var matchers []*regexp.Regexp
	for _, path := range sel.Paths {
		re, err := globRegexp(strings.TrimPrefix(filepath.ToSlash(filepath.Clean(path)), "./"))
		if err != nil {
			return nil, fmt.Errorf("select.paths %q: %w", path, err)
		}
		matchers = append(matchers, re)
	}
	var candidates []Selected
	byID := map[string]string{}
	err := filepath.WalkDir(p.Root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != p.Root && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(p.Root, path)
		if err != nil {
			return err
		}
		if !matchAny(matchers, filepath.ToSlash(rel)) {
			return nil
		}
		block, labels, err := ReadBlock(path)
		if err != nil {
			return err
		}
		if block == nil || !hasAnyLabel(labels, sel.Labels) {
			return nil
		}
		if prev, ok := byID[block.ID]; ok {
			return fmt.Errorf("runnora.id %q が重複しています: %s, %s", block.ID, prev, path)
		}
		byID[block.ID] = path
		candidates = append(candidates, Selected{Path: path, Block: block})
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(sel.IDs) == 0 {
		sort.Slice(candidates, func(i, j int) bool { return candidates[i].Path < candidates[j].Path })
		return candidates, nil
	}
	index := map[string]Selected{}
	for _, c := range candidates {
		index[c.Block.ID] = c
	}
	var (
		selected []Selected
		missing  []string
	)
	for _, id := range sel.IDs {
		c, ok := index[id]
		if !ok {
			missing = append(missing, id)
			continue
		}
		selected = append(selected, c)
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("select.ids: 条件に合う runbook が見つからない ID があります: %s", strings.Join(missing, ", "))
	}
	return selected, nil
}

// globRegexp は glob (/ 区切り) を正規表現に変換する。
// * は / 以外の任意の文字列、? は / 以外の 1 文字、** は任意の階層 (0 階層を含む)。
func globRegexp(glob string) (*regexp.Regexp, error) {
	var sb strings.Builder
	sb.WriteString("^")
	for i := 0; i < len(glob); i++ {
		c := glob[i]
		switch {
		case c == '*' && strings.HasPrefix(glob[i:], "**/"):
			sb.WriteString("(?:.*/)?")
			i += 2
		case c == '*' && strings.HasPrefix(glob[i:], "**"):
			sb.WriteString(".*")
			i++
		case c == '*':
			sb.WriteString("[^/]*")
		case c == '?':
			sb.WriteString("[^/]")
		case c == '[':
			end := strings.IndexByte(glob[i:], ']')
			if end < 0 {
				return nil, errors.New("[ が閉じていません")
			}
			class := glob[i+1 : i+end]
			if strings.HasPrefix(class, "!") {
				class = "^" + class[1:]
			}
			sb.WriteString("[" + class + "]")
			i += end
		default:
			sb.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	sb.WriteString("$")
	return regexp.Compile(sb.String())
}

func matchAny(res []*regexp.Regexp, s string) bool {
	for _, re := range res {
		if re.MatchString(s) {
			return true
		}
	}
	return false
}

func hasAnyLabel(labels, want []string) bool {
	if len(want) == 0 {
		return true
	}
	for _, w := range want {
		for _, l := range labels {
			if l == w {
				return true
			}
		}
	}
	return false
}
