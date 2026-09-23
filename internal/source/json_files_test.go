package source

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadJSONFilesFollowEachStep(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "fixtures"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(dir, "fixtures", "request.json"), `{"name":"Alice"}`)
	writeTestFile(t, filepath.Join(dir, "fixtures", "response.json"), `{"id":"example"}`)
	writeTestFile(t, filepath.Join(dir, "runbook.yml"), `
runners:
  req: {endpoint: http://example.test}
vars:
  input: json://fixtures/request.json
  expected: json://fixtures/response.json
  unused: json://fixtures/unused.json
steps:
  from_files:
    req:
      /users:
        post:
          body:
            application/json: {name: "{{ vars.input.name }}"}
    test: current.res.status == 201 && current.res.body == vars.expected
  inline:
    req:
      /users:
        post:
          body:
            application/json: {name: "vars.input.name"}
    test: current.res.status == 201
  different_variable:
    req:
      /users:
        post:
          body:
            application/json: "{{ vars.inputOther }}"
    test: current.res.body == vars.expectedOther
`)
	scenario, _, err := Load(filepath.Join(dir, "runbook.yml"), Options{})
	if err != nil {
		t.Fatal(err)
	}
	first := scenario.Steps[0]
	if len(first.RequestJSONData) != 1 || first.RequestJSONData[0].Value.(map[string]any)["name"] != "Alice" {
		t.Fatalf("request JSON content = %#v", first.RequestJSONData)
	}
	if len(first.ExpectationJSONData) != 1 || first.ExpectationJSONData[0].Value.(map[string]any)["id"] != "example" {
		t.Fatalf("expectation JSON content = %#v", first.ExpectationJSONData)
	}
	jsonSources := 0
	for _, ref := range scenario.Sources {
		if ref.Kind == "json" && len(ref.SHA256) == 64 {
			jsonSources++
		}
	}
	if jsonSources != 2 {
		t.Fatalf("manifest JSON source count = %d", jsonSources)
	}
	if !reflect.DeepEqual(first.RequestJSONFiles, []string{"fixtures/request.json"}) ||
		!reflect.DeepEqual(first.ExpectationJSONFiles, []string{"fixtures/response.json"}) {
		t.Fatalf("JSON sources = %v / %v", first.RequestJSONFiles, first.ExpectationJSONFiles)
	}
	for _, step := range scenario.Steps[1:] {
		if len(step.RequestJSONFiles)+len(step.ExpectationJSONFiles) != 0 {
			t.Fatalf("unrelated JSON sources for %s: %v / %v", step.ID, step.RequestJSONFiles, step.ExpectationJSONFiles)
		}
	}
}

func TestLoadRejectsMissingOrInvalidReferencedJSON(t *testing.T) {
	for _, tt := range []struct{ name, content, message string }{
		{"missing", "", "参照JSONを読めません"},
		{"invalid", "{broken", "参照JSONが不正です"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeTestFile(t, filepath.Join(dir, "runbook.yml"), `
vars:
  expected: json://expected.json
steps:
  check:
    test: compare(current.res.body, vars.expected)
`)
			if tt.content != "" {
				writeTestFile(t, filepath.Join(dir, "expected.json"), tt.content)
			}
			_, _, err := Load(filepath.Join(dir, "runbook.yml"), Options{})
			if err == nil || !strings.Contains(err.Error(), tt.message) {
				t.Fatalf("error = %v, want %s", err, tt.message)
			}
		})
	}
}

func TestLoadSuiteJSONFilesOnlyForReferencedFields(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "suite.yml"), `
vars:
  cases: [json://valid.json, json://invalid.json]
steps:
  run:
    include: {path: template.yml}
`)
	writeTestFile(t, filepath.Join(dir, "template.yml"), `
runners:
  req: {endpoint: http://example.test}
steps:
  create:
    req:
      /users:
        post:
          body:
            application/json: "{{ vars.case.requestBody }}"
    test: current.res.status == vars.case.expect.status
  health:
    req:
      /health:
        get: {body: null}
    test: current.res.status == 200
`)
	writeTestFile(t, filepath.Join(dir, "valid.json"), `{"requestBody":{"name":"Alice"},"expect":{"status":201}}`)
	writeTestFile(t, filepath.Join(dir, "invalid.json"), `{"requestBody":{},"expect":{"status":400}}`)
	scenario, _, err := Load(filepath.Join(dir, "suite.yml"), Options{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(dir, "invalid.json"), filepath.Join(dir, "valid.json")}
	first := scenario.Steps[0]
	if !reflect.DeepEqual(first.RequestJSONFiles, want) || !reflect.DeepEqual(first.ExpectationJSONFiles, want) {
		t.Fatalf("case sources = %v / %v; want %v", first.RequestJSONFiles, first.ExpectationJSONFiles, want)
	}
	if first.Status.Value != "ケース別" {
		t.Fatalf("mixed case status = %q", first.Status.Value)
	}
	second := scenario.Steps[1]
	if len(second.RequestJSONFiles)+len(second.ExpectationJSONFiles) != 0 {
		t.Fatalf("literal health check has case sources: %#v", second)
	}
}
