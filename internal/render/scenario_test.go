package render

import (
	"strings"
	"testing"

	"github.com/ramsesyok/runnora-test-instructions/internal/model"
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
