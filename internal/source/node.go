package source

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

func mappingValue(n *yaml.Node, key string) *yaml.Node {
	if n == nil || n.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Value == key {
			return n.Content[i+1]
		}
	}
	return nil
}

func mappingEntries(n *yaml.Node) [][2]*yaml.Node {
	if n == nil || n.Kind != yaml.MappingNode {
		return nil
	}
	entries := make([][2]*yaml.Node, 0, len(n.Content)/2)
	for i := 0; i+1 < len(n.Content); i += 2 {
		entries = append(entries, [2]*yaml.Node{n.Content[i], n.Content[i+1]})
	}
	return entries
}

func scalar(n *yaml.Node) string {
	if n == nil {
		return ""
	}
	return n.Value
}

func decodeAny(n *yaml.Node) any {
	if n == nil {
		return nil
	}
	var value any
	if err := n.Decode(&value); err != nil {
		return n.Value
	}
	return normalize(value)
}

func normalize(v any) any {
	switch value := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(value))
		for key, item := range value {
			out[key] = normalize(item)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(value))
		for key, item := range value {
			out[fmt.Sprint(key)] = normalize(item)
		}
		return out
	case []any:
		out := make([]any, len(value))
		for i, item := range value {
			out[i] = normalize(item)
		}
		return out
	default:
		return v
	}
}

func stringMap(n *yaml.Node) map[string]string {
	result := map[string]string{}
	for _, entry := range mappingEntries(n) {
		result[entry[0].Value] = scalar(entry[1])
	}
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
