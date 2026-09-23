package render

import (
	"reflect"
	"testing"
)

func TestFlattenPreservesEmptyJSONContainers(t *testing.T) {
	for _, tt := range []struct {
		name string
		body any
		want [][]string
	}{
		{"object", map[string]any{}, [][]string{{"$", "{}"}}},
		{"array", []any{}, [][]string{{"$", "[]"}}},
		{"nested", map[string]any{"items": []any{}, "options": map[string]any{}}, [][]string{{"$.items", "[]"}, {"$.options", "{}"}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := flattenRows(tt.body); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("empty JSON values must remain distinguishable: got %v, want %v", got, tt.want)
			}
		})
	}
}
