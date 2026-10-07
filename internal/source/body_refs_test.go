package source

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ramsesyok/runnora-docgen/internal/model"
)

func TestResponseBodyVarPaths(t *testing.T) {
	for _, tt := range []struct {
		name string
		test string
		want []string
	}{
		{"compare", `steps.call.res.status == vars.case.expect.status && compare(steps.call.res.body, vars.expected, vars.case.expect.ignorePaths)`, []string{"vars.expected"}},
		{"grpc", `steps.get.res.status == vars.expected.status && compare(steps.get.res.message, vars.expected.message)`, []string{"vars.expected.message"}},
		{"stream", `compare(steps.list.res.messages, vars.expected.messages)`, []string{"vars.expected.messages"}},
		{"diff", `!diffEps(vars.expected, steps.calculate.res.message, "tolerances.yml")`, []string{"vars.expected"}},
		{"equality", `current.res.body == vars.case.expect.body && vars.case.expect.status == 200`, []string{"vars.case.expect.body"}},
		{"status only", `current.res.status == vars.case.expect.status`, nil},
		{"unrelated input", `vars.list_request.genre == genre`, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := responseBodyVarPaths(tt.test); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("responseBodyVarPaths = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBodyRefsFromExplicitIncludeWithSeparatedFiles(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "suite.yml"), `steps:
  created:
    include:
      path: template.yml
      vars:
        case: json://case.json
        request: json://request.json
        expected: json://response.json
`)
	writeTestFile(t, filepath.Join(dir, "template.yml"), `runners:
  req: {endpoint: http://example.test}
steps:
  call:
    req:
      /books:
        post:
          body:
            application/json: "{{ vars.request }}"
    test: |
      current.res.status == vars.case.expect.status &&
      compare(current.res.body, vars.expected, vars.case.expect.ignorePaths)
`)
	writeTestFile(t, filepath.Join(dir, "case.json"), `{"expect":{"status":201,"ignorePaths":[]},"headers":{}}`)
	writeTestFile(t, filepath.Join(dir, "request.json"), `{"title":"Example"}`)
	writeTestFile(t, filepath.Join(dir, "response.json"), `{"id":"B1"}`)

	scenario, _, err := Load(filepath.Join(dir, "suite.yml"), Options{})
	if err != nil {
		t.Fatal(err)
	}
	step := scenario.Steps[0]
	if want := []model.JSONRef{{Path: filepath.Join(dir, "request.json")}}; !reflect.DeepEqual(step.RequestJSONRefs, want) {
		t.Fatalf("request refs = %v, want %v", step.RequestJSONRefs, want)
	}
	if want := []model.JSONRef{{Path: filepath.Join(dir, "response.json")}}; !reflect.DeepEqual(step.ResponseBodyJSONRefs, want) {
		t.Fatalf("response refs = %v, want %v", step.ResponseBodyJSONRefs, want)
	}
}

func TestBodyRefsFromLoopCases(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "suite.yml"), `vars:
  cases: [json://first.json, json://second.json]
steps:
  run:
    loop: {count: len(vars.cases)}
    include: {path: template.yml}
`)
	writeTestFile(t, filepath.Join(dir, "template.yml"), `runners:
  req: {endpoint: http://example.test}
steps:
  call:
    req:
      /books:
        post:
          body:
            application/json: "{{ vars.case.requestBody }}"
    test: |
      current.res.status == vars.case.expect.status &&
      compare(current.res.body, vars.case.expect.body)
`)
	writeTestFile(t, filepath.Join(dir, "first.json"), `{"requestBody":{"id":"A"},"expect":{"status":201,"body":{"id":"A"}}}`)
	writeTestFile(t, filepath.Join(dir, "second.json"), `{"requestBody":{"id":"B"},"expect":{"status":400,"body":{"error":"invalid"}}}`)

	scenario, _, err := Load(filepath.Join(dir, "suite.yml"), Options{})
	if err != nil {
		t.Fatal(err)
	}
	step := scenario.Steps[0]
	if !step.Loop || len(scenario.Cases) != 2 {
		t.Fatalf("loop/cases = %v/%d", step.Loop, len(scenario.Cases))
	}
	wantRequest := []model.JSONRef{
		{Path: filepath.Join(dir, "first.json"), FieldPath: "requestBody"},
		{Path: filepath.Join(dir, "second.json"), FieldPath: "requestBody"},
	}
	wantResponse := []model.JSONRef{
		{Path: filepath.Join(dir, "first.json"), FieldPath: "expect.body"},
		{Path: filepath.Join(dir, "second.json"), FieldPath: "expect.body"},
	}
	if !reflect.DeepEqual(step.RequestJSONRefs, wantRequest) || !reflect.DeepEqual(step.ResponseBodyJSONRefs, wantResponse) {
		t.Fatalf("loop refs = %v / %v", step.RequestJSONRefs, step.ResponseBodyJSONRefs)
	}
}
