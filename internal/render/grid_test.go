package render

import (
	"strings"
	"testing"
)

func TestScenarioGridHasAlignedMergedHeader(t *testing.T) {
	table := scenarioGrid("ユーザー登録", [][]string{{
		"1", "ユーザーを登録する\\\nPOST http://example.test/users", "@tbl-request", "ステータス：HTTP 201\\\n期待値：@tbl-expect", "",
	}}, []int{8, 34, 20, 28, 6})

	lines := strings.Split(strings.TrimSpace(table), "\n")
	wantWidth := displayWidth(lines[0])
	for i, line := range lines {
		if got := displayWidth(line); got != wantWidth {
			t.Fatalf("line %d width = %d, want %d:\n%s", i+1, got, wantWidth, line)
		}
	}
	if !strings.Contains(table, "ステータス：HTTP 201") {
		t.Fatalf("status is missing:\n%s", table)
	}
}

func TestDisplayWidthTreatsJapaneseAsWide(t *testing.T) {
	if got, want := displayWidth("A試験"), 5; got != want {
		t.Fatalf("displayWidth = %d, want %d", got, want)
	}
}

func TestGridTableDoesNotSplitTableReference(t *testing.T) {
	table := gridTable(
		[]string{"入力"},
		[][]string{{"requestBody: @tbl-req-ceff0909"}},
		[]int{28},
	)

	if !strings.Contains(table, "@tbl-req-ceff0909") {
		t.Fatalf("table reference was split:\n%s", table)
	}
}

func TestSourceLocationBreaksLongTemplateName(t *testing.T) {
	got := sourceLocation(`C:\runbooks\post_updatePetWithForm.template.yml`, 19)
	want := "post_\\\nupdatePet\\\nWithForm\\\n.template.yml:19"
	if got != want {
		t.Fatalf("sourceLocation = %q, want %q", got, want)
	}
}

func TestGridTableRendersColumnAlignments(t *testing.T) {
	table := gridTable(
		[]string{"中央", "左", "右"},
		[][]string{{"1", "value", "2"}},
		[]int{6, 8, 6},
		alignCenter, alignLeft, alignRight,
	)

	if !strings.Contains(table, "+:======:+:=========+=======:+") {
		t.Fatalf("alignment border is missing:\n%s", table)
	}
}
