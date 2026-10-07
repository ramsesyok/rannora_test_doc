package generator

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// manifest.json の scenarioId と steps は、runnora の report.json (results[].id、steps[].key) と
// 同じ値にする。loop のステップは手順書では 1 行なので、キーに [n] を付けず loop: true にする。
func TestManifestStepsMatchRunnoraKeys(t *testing.T) {
	const http = "runners:\n  req:\n    endpoint: http://example.test\n"
	tests := []struct {
		name   string
		files  map[string]string
		id     string
		folder string
		steps  []manifestStep
	}{
		{
			name: "top level, loop and include",
			files: map[string]string{
				"main.yml": "desc: 貸出\nrunnora:\n  id: LIB-001\n" + http + `steps:
  member_before:
    req:
      /members/1:
        get: {body: null}
  member_loans:
    loop:
      count: 3
    req:
      /members/1/loans:
        get: {body: null}
  inspect:
    loop:
      count: 2
    include:
      path: part.yml
  check:
    test: true
`,
				"part.yml": "desc: 部品\n" + http + `steps:
  call:
    req:
      /books/1:
        get: {body: null}
  verify:
    test: true
`,
			},
			id:     "LIB-001",
			folder: "lib-001",
			steps: []manifestStep{
				{Number: 1, Key: "member_before"},
				{Number: 2, Key: "member_loans", Loop: true},
				{Number: 3, Key: "inspect.call", Loop: true},
				{Number: 4, Key: "inspect.verify", Loop: true},
				{Number: 5, Key: "check"},
			},
		},
		{
			name: "suite calling a template for each case",
			files: map[string]string{
				"main.yml": `desc: 生成 suite
runnora:
  id: GEN-getBook
vars:
  cases:
    - "json://case.json"
steps:
  run_case:
    loop:
      count: len(vars.cases)
    include:
      path: ./tmpl.yml
      vars:
        case: "{{ vars.cases[i] }}"
`,
				"tmpl.yml": "desc: template\n" + http + `vars:
  case: {}
steps:
  call_api:
    req:
      /books/1:
        get: {body: null}
    test: current.res.status == 200
`,
				"case.json": `{"name": "default", "expect": {"status": 200}}`,
			},
			id:     "GEN-getBook",
			folder: "gen-getbook",
			steps:  []manifestStep{{Number: 1, Key: "run_case.call_api", Loop: true}},
		},
		{
			name: "list steps without runnora block",
			files: map[string]string{
				"main.yml": "desc: 配列\n" + http + `steps:
  - req:
      /health:
        get: {body: null}
  - test: true
`,
			},
			folder: "main",
			steps:  []manifestStep{{Number: 1, Key: "0"}, {Number: 2, Key: "1"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			out := filepath.Join(dir, "generated")
			if _, err := Generate(context.Background(), Options{RunbookPaths: []string{filepath.Join(dir, "main.yml")}, OutputDir: out}); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(filepath.Join(out, tt.folder, "manifest.json"))
			if err != nil {
				t.Fatal(err)
			}
			var got manifest
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatal(err)
			}
			if got.ScenarioID != tt.id {
				t.Errorf("scenarioId = %q, want %q", got.ScenarioID, tt.id)
			}
			if !reflect.DeepEqual(got.Steps, tt.steps) {
				t.Errorf("steps = %+v, want %+v\n%s", got.Steps, tt.steps, data)
			}
		})
	}
}
