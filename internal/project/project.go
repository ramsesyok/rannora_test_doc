// Package project は runnora のプロジェクトファイル (runnora.yaml, version: 2) のうち、
// 手順書に必要な部分 (環境とスイートの前後処理、スイートの runbook の選び方) を読む。
//
// runnora 本体の internal/project と同じ規則で読むが、runnora の internal パッケージは
// 取り込めないため、必要な範囲だけをここに持つ。
package project

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Hooks は前後処理の SQL (プロジェクトルート基準のパス)。
type Hooks struct {
	Before []string `yaml:"before"`
	After  []string `yaml:"after"`
}

// Selection はスイートが runbook を選ぶ条件。
type Selection struct {
	Paths  []string `yaml:"paths"`
	Labels []string `yaml:"labels"`
	IDs    []string `yaml:"ids"`
}

type environment struct {
	Vars  map[string]string `yaml:"vars"`
	Hooks Hooks             `yaml:"hooks"`
}

type suite struct {
	Env    string            `yaml:"env"`
	Select Selection         `yaml:"select"`
	Vars   map[string]string `yaml:"vars"`
	Hooks  Hooks             `yaml:"hooks"`
}

type file struct {
	Version  int `yaml:"version"`
	Defaults struct {
		Env string `yaml:"env"`
	} `yaml:"defaults"`
	Environments map[string]*environment `yaml:"environments"`
	Suites       map[string]*suite       `yaml:"suites"`
}

// Project は読み込んだ runnora.yaml。
type Project struct {
	// Path は runnora.yaml の絶対パス、Root はそのディレクトリ (相対パスの基準)。
	Path string
	Root string
	f    file
}

// FileName は新形式のプロジェクトファイルの名前。
const FileName = "runnora.yaml"

// Find は dir から親へ向かって runnora.yaml を探す。見つからなければ "" を返す。
func Find(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(abs, FileName)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return "", nil
		}
		abs = parent
	}
}

// Load は runnora.yaml を読む。version: 2 でなければエラーにする。
// runnora 本体が検査する項目 (未知のキーなど) はここでは検査しない (runnora validate を使う)。
func Load(path string) (*Project, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("runnora.yaml を読めません: %w", err)
	}
	var f file
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("runnora.yaml が不正です: %s: %w", abs, err)
	}
	if f.Version != 2 {
		return nil, fmt.Errorf("%s: version: 2 の runnora.yaml ではありません (旧形式の config.yaml は --config で指定してください)", abs)
	}
	return &Project{Path: abs, Root: filepath.Dir(abs), f: f}, nil
}

// SelectEnv は使う環境を決める。--env、スイートの env、defaults.env の順。
// --env とスイートの env が食い違う場合はエラー (runnora run と同じ)。
func (p *Project) SelectEnv(envFlag, suiteName string) (string, error) {
	var suiteEnv string
	if suiteName != "" {
		s, ok := p.f.Suites[suiteName]
		if !ok {
			return "", fmt.Errorf("スイート %q が runnora.yaml にありません (定義済み: %v)", suiteName, names(p.f.Suites))
		}
		suiteEnv = s.Env
	}
	env := envFlag
	switch {
	case env != "" && suiteEnv != "" && env != suiteEnv:
		return "", fmt.Errorf("--env %s はスイート %s の環境 %s と違います", env, suiteName, suiteEnv)
	case env == "":
		env = suiteEnv
	}
	if env == "" {
		env = p.f.Defaults.Env
	}
	if env != "" {
		if _, ok := p.f.Environments[env]; !ok {
			return "", fmt.Errorf("環境 %q が runnora.yaml にありません (定義済み: %v)", env, names(p.f.Environments))
		}
	}
	return env, nil
}

// Hooks は環境 envName とスイート suiteName の前後処理を絶対パスで返す。
// before は 環境 → スイート、after は スイート → 環境 の順 (runnora run と同じ)。
// パスの ${NAME} / ${NAME:-既定値} は OS の環境変数、スイートの vars、環境の vars の順で展開する。
func (p *Project) Hooks(envName, suiteName string) (Hooks, error) {
	env := p.f.Environments[envName]
	if env == nil {
		env = &environment{}
	}
	s := p.f.Suites[suiteName]
	if s == nil {
		s = &suite{}
	}
	lookup := func(name string) (string, bool) {
		if v, ok := os.LookupEnv(name); ok {
			return v, true
		}
		if v, ok := s.Vars[name]; ok {
			return v, true
		}
		v, ok := env.Vars[name]
		return v, ok
	}
	var errs []string
	resolve := func(paths []string) []string {
		var out []string
		for _, path := range paths {
			expanded, undefined := expand(path, lookup)
			if len(undefined) > 0 {
				errs = append(errs, fmt.Sprintf("%s: 未定義の変数 %s", path, strings.Join(undefined, ", ")))
				continue
			}
			out = append(out, p.Abs(expanded))
		}
		return out
	}
	var h Hooks
	h.Before = append(resolve(env.Hooks.Before), resolve(s.Hooks.Before)...)
	h.After = append(resolve(s.Hooks.After), resolve(env.Hooks.After)...)
	if len(errs) > 0 {
		return Hooks{}, fmt.Errorf("runnora.yaml の hooks: %s", strings.Join(errs, "; "))
	}
	return h, nil
}

// Abs はプロジェクトルート基準のパスを絶対パスにする。
func (p *Project) Abs(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(p.Root, filepath.FromSlash(path))
}

// Suite はスイートの選択条件を返す。
func (p *Project) Suite(name string) (Selection, error) {
	s, ok := p.f.Suites[name]
	if !ok {
		return Selection{}, fmt.Errorf("スイート %q が runnora.yaml にありません (定義済み: %v)", name, names(p.f.Suites))
	}
	return s.Select, nil
}

func names[T any](m map[string]T) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

var varPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)([^}]*)\}`)

// expand は ${NAME} と ${NAME:-既定値} を展開する (runnora と同じ規則)。
func expand(s string, lookup func(string) (string, bool)) (string, []string) {
	var undefined []string
	out := varPattern.ReplaceAllStringFunc(s, func(m string) string {
		sub := varPattern.FindStringSubmatch(m)
		name, op := sub[1], sub[2]
		if v, ok := lookup(name); ok && (v != "" || !strings.HasPrefix(op, ":-")) {
			return v
		}
		switch {
		case strings.HasPrefix(op, ":-"):
			return op[2:]
		case strings.HasPrefix(op, "-"):
			return op[1:]
		}
		undefined = append(undefined, name)
		return ""
	})
	return out, undefined
}
