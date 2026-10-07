package render

import (
	"strings"
	"testing"

	"github.com/ramsesyok/runnora-docgen/internal/model"
)

func TestFilenamesModeKeepsBodySourcesWithoutDetailTables(t *testing.T) {
	scenario := &model.Scenario{ID: "review", Name: "review", Steps: []model.Step{
		{
			ID: "call", Number: 1, Kind: model.StepHTTP, SourcePath: "runbooks/template.yml",
			HTTP:            &model.HTTPRequest{Body: map[string]any{"id": "B1"}},
			RequestJSONRefs: []model.JSONRef{{Path: "cases/one.json", FieldPath: "requestBody"}},
		},
		{
			ID: "check", Number: 2, Kind: model.StepTest, Test: "compare(current.res.body, vars.expected)",
			Status:               model.StatusExpectation{Protocol: "HTTP", Value: "201"},
			ExpectationJSONFiles: []string{"cases/one.json", "responses/body.json"},
			ResponseBodyJSONRefs: []model.JSONRef{{Path: "responses/body.json"}},
		},
	}}
	documents := ScenarioDocuments(scenario, false)
	byName := map[string]string{}
	for _, doc := range documents {
		byName[doc.Name] = string(doc.Content)
	}
	summary := byName["scenario.qmd"]
	for _, want := range []string{"one.json（requestBody）", "body.json", `期待値：\[201\]`} {
		if !strings.Contains(summary, want) {
			t.Fatalf("missing %q in scenario:\n%s", want, summary)
		}
	}
	if strings.Contains(summary, "@tbl-req-") || strings.Contains(summary, "@tbl-exp-") || strings.Contains(summary, "one.json\\") {
		t.Fatalf("detail link or unrelated case reference remains:\n%s", summary)
	}
	for _, name := range []string{"request-json.qmd", "expectations.qmd"} {
		if content := byName[name]; content == "" || strings.Contains(content, "::: {.tbl") {
			t.Fatalf("%s should be an include-compatible placeholder: %q", name, content)
		}
	}
}

func TestFormatBodyRefsDisambiguatesEqualBasenames(t *testing.T) {
	got := formatBodyRefs([]model.JSONRef{
		{Path: "cases/one/default.json", FieldPath: "requestBody"},
		{Path: "cases/two/default.json", FieldPath: "requestBody"},
	})
	for _, want := range []string{"cases/one/default.json（requestBody）", "cases/two/default.json（requestBody）"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
}

func TestFilenamesModePointsToInlineBodyCheck(t *testing.T) {
	scenario := &model.Scenario{ID: "inline", Steps: []model.Step{{
		ID: "check", Number: 1, Kind: model.StepTest,
		Test: "len(current.res.body) == 3", HasResponseBodyCheck: true,
		SourcePath: "runbooks/check.yml", SourceLine: 12,
	}}}
	got := renderScenarioWithDetail(scenario, true)
	if !strings.Contains(got, "check.yml:12（検証式）") {
		t.Fatalf("inline body check has no source location:\n%s", got)
	}
}
