package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ramsesyok/runnora-test-instructions/internal/model"
	"github.com/ramsesyok/runnora-test-instructions/internal/render"
	"github.com/ramsesyok/runnora-test-instructions/internal/source"
)

type Options struct {
	RunbookPaths []string
	OutputDir    string
	ConfigPath   string
	BeforeSQL    []string
	AfterSQL     []string
	ProtoPaths   []string
	BaseDir      string
	Force        bool
}

type Result struct {
	Files    []string
	Warnings []string
}

type plannedFile struct {
	path    string
	content []byte
}

type manifest struct {
	Generator string   `json:"generator"`
	Scenario  string   `json:"scenario"`
	Source    string   `json:"source"`
	SHA256    string   `json:"sha256"`
	Files     []string `json:"files"`
	Inputs    any      `json:"inputs"`
}

func Generate(ctx context.Context, opts Options) (*Result, error) {
	if len(opts.RunbookPaths) == 0 {
		return nil, fmt.Errorf("runbook を1つ以上指定してください")
	}
	if opts.OutputDir == "" {
		opts.OutputDir = "generated"
	}
	outputDir, err := filepath.Abs(opts.OutputDir)
	if err != nil {
		return nil, fmt.Errorf("output path: %w", err)
	}

	var plan []plannedFile
	result := &Result{}
	usedIDs := map[string]string{}
	for _, runbook := range opts.RunbookPaths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		scenario, warnings, err := source.Load(runbook, source.Options{
			ConfigPath: opts.ConfigPath,
			BeforeSQL:  opts.BeforeSQL,
			AfterSQL:   opts.AfterSQL,
			ProtoPaths: opts.ProtoPaths,
			BaseDir:    opts.BaseDir,
		})
		if err != nil {
			return nil, err
		}
		makePathsPortable(scenario, opts.BaseDir)
		if previous, exists := usedIDs[scenario.ID]; exists {
			return nil, fmt.Errorf("出力 ID %q が重複します: %s, %s", scenario.ID, previous, scenario.SourcePath)
		}
		usedIDs[scenario.ID] = scenario.SourcePath
		result.Warnings = append(result.Warnings, warnings...)

		documents := render.ScenarioDocuments(scenario)
		scenarioDir := filepath.Join(outputDir, scenario.ID)
		manifestFiles := make([]string, 0, len(documents))
		for _, document := range documents {
			path := filepath.Join(scenarioDir, document.Name)
			plan = append(plan, plannedFile{path: path, content: document.Content})
			manifestFiles = append(manifestFiles, document.Name)
		}
		sort.Strings(manifestFiles)
		manifestData, err := json.MarshalIndent(manifest{
			Generator: "runnora-instructions",
			Scenario:  scenario.Name,
			Source:    scenario.SourcePath,
			SHA256:    scenario.SourceHash,
			Files:     manifestFiles,
			Inputs:    scenario.Sources,
		}, "", "  ")
		if err != nil {
			return nil, err
		}
		manifestData = append(manifestData, '\n')
		plan = append(plan, plannedFile{path: filepath.Join(scenarioDir, "manifest.json"), content: manifestData})
	}

	for _, file := range plan {
		if _, err := os.Stat(file.path); err == nil && !opts.Force {
			return nil, fmt.Errorf("生成先が既に存在します（上書きするには --force）: %s", file.path)
		} else if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("生成先を確認できません: %s: %w", file.path, err)
		}
	}
	for _, file := range plan {
		if err := os.MkdirAll(filepath.Dir(file.path), 0o755); err != nil {
			return nil, fmt.Errorf("出力ディレクトリを作れません: %w", err)
		}
		if err := os.WriteFile(file.path, file.content, 0o644); err != nil {
			return nil, fmt.Errorf("原稿を書き込めません: %s: %w", file.path, err)
		}
		result.Files = append(result.Files, file.path)
	}
	return result, nil
}

func makePathsPortable(scenario *model.Scenario, baseDir string) {
	base := baseDir
	if strings.TrimSpace(base) == "" {
		base, _ = os.Getwd()
	}
	if absolute, err := filepath.Abs(base); err == nil {
		base = absolute
	}

	scenario.SourcePath = portablePath(base, scenario.SourcePath)
	for index := range scenario.Steps {
		scenario.Steps[index].SourcePath = portablePath(base, scenario.Steps[index].SourcePath)
	}
	for index := range scenario.Cases {
		scenario.Cases[index].SourcePath = portablePath(base, scenario.Cases[index].SourcePath)
	}
	for index := range scenario.BeforeHooks {
		scenario.BeforeHooks[index].Path = portablePath(base, scenario.BeforeHooks[index].Path)
	}
	for index := range scenario.AfterHooks {
		scenario.AfterHooks[index].Path = portablePath(base, scenario.AfterHooks[index].Path)
	}
	for index := range scenario.Sources {
		scenario.Sources[index].Path = portablePath(base, scenario.Sources[index].Path)
	}
}

func portablePath(base, path string) string {
	if strings.TrimSpace(path) == "" {
		return path
	}
	relative, err := filepath.Rel(base, path)
	if err != nil {
		return filepath.ToSlash(filepath.Base(path))
	}
	return filepath.ToSlash(relative)
}
