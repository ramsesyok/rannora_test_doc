package generator

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeProjectFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const projectRunbookBody = `runners:
  req: {endpoint: "${API_URL}"}
steps:
  health:
    req:
      /health:
        get: {body: null}
    test: current.res.status == 200
`

// setupProject は新形式のプロジェクトを作る。
//   - 環境 unit (hooks: env-before / env-after)、mock (hooks なし)
//   - スイート scenarios (env: unit、hooks: suite-before / suite-after)
//   - runbook: a (before/after 付き)、b (envs: [mock])、part (ブロックなし = include される部品)
func setupProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeProjectFile(t, filepath.Join(dir, "runnora.yaml"), `version: 2
defaults:
  env: unit
environments:
  unit:
    vars: {API_URL: "http://127.0.0.1:1"}
    hooks:
      before: [sql/env-before.sql]
      after: [sql/env-after.sql]
  mock:
    vars: {API_URL: "http://127.0.0.1:2"}
suites:
  scenarios:
    env: unit
    select:
      paths: [runbooks/**/*.yml]
    hooks:
      before: [sql/suite-before.sql]
      after: [sql/suite-after.sql]
  picked:
    select:
      paths: [runbooks/**/*.yml]
      ids: [B-1, A-1]
`)
	writeProjectFile(t, filepath.Join(dir, "runbooks", "a.yml"), "desc: A-1 scenario\nrunnora:\n  id: A-1\n  before: [sql/a-before.sql]\n  after: [sql/a-after.sql]\n"+projectRunbookBody)
	writeProjectFile(t, filepath.Join(dir, "runbooks", "sub", "b.yml"), "desc: B-1 mock only\nrunnora:\n  id: B-1\n  envs: [mock]\n"+projectRunbookBody)
	writeProjectFile(t, filepath.Join(dir, "runbooks", "part.yml"), "desc: part\n"+projectRunbookBody)
	for _, name := range []string{"env-before", "env-after", "suite-before", "suite-after", "a-before", "a-after", "extra-before", "extra-after"} {
		writeProjectFile(t, filepath.Join(dir, "sql", name+".sql"), "-- "+name+"\n")
	}
	return dir
}

func baseNames(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		out = append(out, strings.TrimSuffix(filepath.Base(p), ".sql"))
	}
	return out
}

func TestPlanInputsFromProject(t *testing.T) {
	dir := setupProject(t)
	project := filepath.Join(dir, "runnora.yaml")
	runbookA := filepath.Join(dir, "runbooks", "a.yml")

	tests := []struct {
		name         string
		opts         Options
		wantRunbooks []string
		wantBefore   [][]string
		wantAfter    [][]string
		wantWarning  bool
	}{
		{
			name:         "suite selects runbooks with a runnora block and skips ones for other envs",
			opts:         Options{ProjectPath: project, Suite: "scenarios"},
			wantRunbooks: []string{"a.yml"},
			wantBefore:   [][]string{{"env-before", "suite-before", "a-before"}},
			wantAfter:    [][]string{{"a-after", "suite-after", "env-after"}},
			wantWarning:  true,
		},
		{
			name:         "ids keep their order and --env picks the environment",
			opts:         Options{ProjectPath: project, Suite: "picked", Env: "mock"},
			wantRunbooks: []string{"b.yml", "a.yml"},
			wantBefore:   [][]string{nil, {"a-before"}},
			wantAfter:    [][]string{nil, {"a-after"}},
		},
		{
			name:         "runbook arguments use the default environment and extra SQL goes inside",
			opts:         Options{ProjectPath: project, RunbookPaths: []string{runbookA}, BeforeSQL: []string{"sql/extra-before.sql"}},
			wantRunbooks: []string{"a.yml"},
			wantBefore:   [][]string{{"env-before", "a-before"}},
			wantAfter:    [][]string{{"a-after", "env-after"}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			inputs, warnings, err := planInputs(tc.opts)
			if err != nil {
				t.Fatal(err)
			}
			if (len(warnings) > 0) != tc.wantWarning {
				t.Errorf("warnings = %v", warnings)
			}
			var got []string
			for i, in := range inputs {
				got = append(got, filepath.Base(in.path))
				if in.opts.BaseDir != dir || in.opts.ProjectPath != project {
					t.Errorf("base dir / project = %q / %q", in.opts.BaseDir, in.opts.ProjectPath)
				}
				if i < len(tc.wantBefore) {
					if b := baseNames(in.opts.ProjectBefore); !reflect.DeepEqual(b, nonNil(tc.wantBefore[i])) {
						t.Errorf("%s before = %v, want %v", got[i], b, tc.wantBefore[i])
					}
					if a := baseNames(in.opts.ProjectAfter); !reflect.DeepEqual(a, nonNil(tc.wantAfter[i])) {
						t.Errorf("%s after = %v, want %v", got[i], a, tc.wantAfter[i])
					}
				}
			}
			if !reflect.DeepEqual(got, tc.wantRunbooks) {
				t.Errorf("runbooks = %v, want %v", got, tc.wantRunbooks)
			}
		})
	}
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func TestPlanInputsRejectsBadCombinations(t *testing.T) {
	dir := setupProject(t)
	project := filepath.Join(dir, "runnora.yaml")
	legacy := filepath.Join(dir, "config.yaml")
	writeProjectFile(t, legacy, "hooks: {}\n")
	tests := []struct {
		name string
		opts Options
		want string
	}{
		{"config with project", Options{ConfigPath: legacy, ProjectPath: project, RunbookPaths: []string{"x.yml"}}, "同時に指定できません"},
		{"suite with runbooks", Options{ProjectPath: project, Suite: "scenarios", RunbookPaths: []string{"x.yml"}}, "同時に指定できません"},
		{"nothing to generate", Options{ProjectPath: project}, "--suite"},
		{"env conflicts with suite", Options{ProjectPath: project, Suite: "scenarios", Env: "mock"}, "違います"},
		{"unknown suite", Options{ProjectPath: project, Suite: "nope"}, "nope"},
		{"legacy file as project", Options{ProjectPath: legacy, RunbookPaths: []string{"x.yml"}}, "version: 2"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := planInputs(tc.opts)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want containing %q", err, tc.want)
			}
		})
	}
}

// スイートを指定した生成で、前後処理の原稿と runnora.yaml の出典が作られる。
func TestGenerateSuiteFromProject(t *testing.T) {
	dir := setupProject(t)
	out := filepath.Join(dir, "generated")
	result, err := Generate(context.Background(), Options{ProjectPath: filepath.Join(dir, "runnora.yaml"), Suite: "scenarios", OutputDir: out})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], "b.yml") {
		t.Errorf("warnings = %v", result.Warnings)
	}
	before, err := os.ReadFile(filepath.Join(out, "a-1", "before.qmd"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(before)
	iEnv, iSuite, iRb := strings.Index(text, "env-before"), strings.Index(text, "suite-before"), strings.Index(text, "a-before")
	if iEnv < 0 || iSuite < iEnv || iRb < iSuite {
		t.Errorf("before.qmd order is wrong:\n%s", text)
	}
	manifest, err := os.ReadFile(filepath.Join(out, "a-1", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(manifest), `"runnora.yaml"`) || !strings.Contains(string(manifest), `"project"`) {
		t.Errorf("manifest does not record runnora.yaml:\n%s", manifest)
	}
	if _, err := os.Stat(filepath.Join(out, "b-1")); !os.IsNotExist(err) {
		t.Errorf("runbook for another environment was generated")
	}
}
