package generator

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateKeepsCountOutOfScenarioTable(t *testing.T) {
	dir := t.TempDir()
	runbook := filepath.Join(dir, "scenario.yml")
	content := `desc: 一覧取得
runners:
  req:
    endpoint: http://example.test
steps:
  list:
    desc: ユーザー一覧を取得する
    req:
      /users:
        get:
          body: null
    test: current.res.status == 200 && len(current.res.body) == 3
`
	if err := os.WriteFile(runbook, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "generated")
	result, err := Generate(context.Background(), Options{RunbookPaths: []string{runbook}, OutputDir: output})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) == 0 {
		t.Fatal("no files generated")
	}
	manifestData, err := os.ReadFile(filepath.Join(output, "scenario", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var recorded map[string]any
	if err := json.Unmarshal(manifestData, &recorded); err != nil {
		t.Fatal(err)
	}
	inputs, ok := recorded["inputs"].([]any)
	if !ok || len(inputs) == 0 {
		t.Fatalf("manifest inputs = %#v", recorded["inputs"])
	}
	if filepath.IsAbs(recorded["source"].(string)) {
		t.Fatalf("manifest source is absolute: %q", recorded["source"])
	}
	for _, input := range inputs {
		path := input.(map[string]any)["path"].(string)
		if filepath.IsAbs(path) {
			t.Fatalf("manifest input path is absolute: %q", path)
		}
		if strings.Contains(path, `\`) {
			t.Fatalf("manifest input path is not slash-normalized: %q", path)
		}
	}
	scenarioData, err := os.ReadFile(filepath.Join(output, "scenario", "scenario.qmd"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(scenarioData), `期待値：\[200\]`) {
		t.Fatalf("scenario does not contain status:\n%s", scenarioData)
	}
	// 出典は相対パスで書く (一時ディレクトリの相対パスは OS によって "../tmp/..." になるので、行の先頭で判定する)
	for _, line := range strings.Split(string(scenarioData), "\n") {
		if source, ok := strings.CutPrefix(line, "source: "); ok && (filepath.IsAbs(source) || strings.HasPrefix(source, "/")) {
			t.Fatalf("scenario header contains an absolute source path:\n%s", scenarioData)
		}
	}
	if !strings.Contains(string(scenarioData), "::: {.landscape}\n::: {.tbl") {
		t.Fatalf("scenario table is not landscape:\n%s", scenarioData)
	}
	if !strings.Contains(string(scenarioData), `widths="8,46,20,20,6"`) {
		t.Fatalf("scenario widths are incorrect:\n%s", scenarioData)
	}
	if strings.Contains(string(scenarioData), "len(current.res.body)") {
		t.Fatalf("scenario contains non-status assertion:\n%s", scenarioData)
	}
	if strings.Contains(string(scenarioData), "GET http://example.test/users") {
		t.Fatalf("scenario duplicates HTTP method and URL:\n%s", scenarioData)
	}
	if !strings.Contains(string(scenarioData), "URL：[@tbl-http-") || !strings.Contains(string(scenarioData), `]-\[1\]`) {
		t.Fatalf("scenario does not reference HTTP call details:\n%s", scenarioData)
	}
	httpData, err := os.ReadFile(filepath.Join(output, "scenario", "http.qmd"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(httpData), `widths="6,8,40,23,23"`) {
		t.Fatalf("HTTP widths are incorrect:\n%s", httpData)
	}
	if strings.Contains(string(httpData), "Runner") {
		t.Fatalf("HTTP table still contains Runner column:\n%s", httpData)
	}
	expectationData, err := os.ReadFile(filepath.Join(output, "scenario", "expectations.qmd"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(expectationData), "len(current.res.body) == 3") {
		t.Fatalf("expectation details lost assertion:\n%s", expectationData)
	}
	if !strings.Contains(string(expectationData), "::: {.landscape}\n::: {.tbl") {
		t.Fatalf("expectation table is not landscape:\n%s", expectationData)
	}
}

func TestGenerateRequiresForceForExistingOutput(t *testing.T) {
	dir := t.TempDir()
	runbook := filepath.Join(dir, "scenario.yml")
	content := `desc: health
runnora:
  id: HEALTH-001
runners:
  req: {endpoint: http://example.test}
steps:
  health:
    req:
      /health:
        get: {body: null}
    test: current.res.status == 200
`
	if err := os.WriteFile(runbook, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	opts := Options{RunbookPaths: []string{runbook}, OutputDir: filepath.Join(dir, "generated")}
	if _, err := Generate(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(opts.OutputDir, "health-001", "scenario.qmd")); err != nil {
		t.Fatalf("ID-based output is missing: %v", err)
	}
	if _, err := Generate(context.Background(), opts); err == nil || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("second generation error = %v, want --force guidance", err)
	}
	opts.Force = true
	if _, err := Generate(context.Background(), opts); err != nil {
		t.Fatalf("force generation failed: %v", err)
	}
}

func TestGenerateRejectsCollidingNormalizedRunnoraIDs(t *testing.T) {
	dir := t.TempDir()
	for name, id := range map[string]string{"first.yml": "A_B", "second.yml": "A-B"} {
		content := "runnora:\n  id: " + id + "\nsteps:\n  check:\n    test: true\n"
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	out := filepath.Join(dir, "generated")
	_, err := Generate(context.Background(), Options{
		RunbookPaths: []string{filepath.Join(dir, "first.yml"), filepath.Join(dir, "second.yml")},
		OutputDir:    out,
	})
	if err == nil || !strings.Contains(err.Error(), `同じ生成コマンド内で出力フォルダ名 "a-b" が重複します`) {
		t.Fatalf("expected normalized ID collision, got %v", err)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("output was written before collision check: %v", err)
	}
}
