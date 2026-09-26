package render

import (
	"strings"
	"testing"

	"github.com/ramsesyok/runnora-docgen/internal/model"
)

func TestScenarioReviewerReferences(t *testing.T) {
	scenario := &model.Scenario{ID: "review", Steps: []model.Step{{
		ID: "create", Number: 3, Kind: model.StepHTTP,
		HTTP:                 &model.HTTPRequest{Body: map[string]any{}},
		Test:                 "current.res.status == 201",
		Status:               model.StatusExpectation{Protocol: "HTTP", Value: "201"},
		RequestJSONFiles:     []string{"fixtures/create.json", "fixtures/create.json"},
		ExpectationJSONFiles: []string{`fixtures\response.json`},
	}}}
	got := renderScenario(scenario)
	for _, want := range []string{
		"URL：[@" + label("review", "http") + `]-\[3\]`,
		"@" + label("review", "request", "create") + "\\",
		`期待値：\[201\]`, "@" + label("review", "expect", "create") + "\\",
		"create.json", "response.json",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "ステータス：") || strings.Contains(got, "fixtures") || strings.Count(got, "create.json") != 1 {
		t.Fatalf("unexpected legacy status, directory, or duplicate filename:\n%s", got)
	}
}

func TestScenarioExpectationWithoutLiteralStatus(t *testing.T) {
	for _, tt := range []struct{ name, protocol, value, variable, want string }{
		{"no status", "", "", "", "期待値：@"},
		{"variable", "HTTP", "", "vars.status", `期待値：\[vars.status\]`},
		{"cases", "HTTP", "ケース別", "", `期待値：\[ケース別\]`},
		{"grpc", "gRPC", "0", "", `期待値：\[gRPC 0\]`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := renderScenario(&model.Scenario{ID: "review", Steps: []model.Step{{
				ID: "check", Number: 1, Kind: model.StepTest, Test: "check",
				Status: model.StatusExpectation{Protocol: tt.protocol, Value: tt.value, Variable: tt.variable},
			}}})
			if !strings.Contains(got, tt.want) || strings.Contains(got, ".json") || strings.Contains(got, "[200]") {
				t.Fatalf("unexpected expectation:\n%s", got)
			}
		})
	}
}

func TestHooksListFileNamesWithoutSQLBody(t *testing.T) {
	scenario := &model.Scenario{
		ID: "hooks", Name: "前後処理",
		BeforeHooks: []model.Asset{
			{Path: "sql/common/00_reset.sql", Content: "BEGIN\n  DELETE FROM loans;\nEND;"},
			{Path: "sql/cases/setup.sql", Content: "BEGIN NULL; END;"},
		},
		AfterHooks: []model.Asset{{Path: "sql/common/90_verify.sql", Content: "BEGIN NULL; END;"}},
	}
	before := renderHooks(scenario, true)
	for _, want := range []string{"順序", "ファイル", "出典", "00_reset.sql", "sql/common/00_reset.sql", "setup.sql", "sql/cases/setup.sql"} {
		if !strings.Contains(before, want) {
			t.Fatalf("missing %q:\n%s", want, before)
		}
	}
	if strings.Contains(before, "SQL/PLSQL") || strings.Contains(before, "DELETE FROM") || strings.Contains(before, "BEGIN") {
		t.Fatalf("SQL body must not be rendered:\n%s", before)
	}
	if strings.Index(before, "00_reset.sql") > strings.Index(before, "setup.sql") {
		t.Fatalf("hooks must keep execution order:\n%s", before)
	}
	after := renderHooks(scenario, false)
	if !strings.Contains(after, "90_verify.sql") || strings.Contains(after, "BEGIN") {
		t.Fatalf("unexpected after hooks table:\n%s", after)
	}
	if renderHooks(&model.Scenario{ID: "empty"}, true) != "" {
		t.Fatal("no hooks should render nothing")
	}
}

func TestHTTPTableShowsQueryPerLine(t *testing.T) {
	scenario := &model.Scenario{ID: "query", Steps: []model.Step{{
		ID: "search", Number: 1, Kind: model.StepHTTP,
		HTTP: &model.HTTPRequest{Endpoint: "http://example.test", Method: "GET", Path: "/books",
			Query: "genre=NOVEL&availableOnly=", Headers: map[string]any{"X-Test-Case": "search"}},
	}}}
	got := renderHTTP(scenario)
	for _, want := range []string{"http://example.test/books", `genre=NOVEL\`, "availableOnly=", `{"X-Test-Case":"search"}`} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "/books?") {
		t.Fatalf("query must not stay in the URL column:\n%s", got)
	}
}
