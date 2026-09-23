package source

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/ramsesyok/runnora-docgen/internal/model"
)

// Read static JSON inputs so the review tables and manifest describe the same
// fixtures that runn uses. Included runbooks keep their own relative base path.
func loadJSONDetails(scenario *model.Scenario, vars any) error {
	cache := map[string]model.JSONData{}
	load := func(paths []string) ([]model.JSONData, error) {
		var result []model.JSONData
		for _, path := range paths {
			abs := resolveReference(filepath.Dir(scenario.SourcePath), path)
			asset, exists := cache[abs]
			if !exists {
				data, err := os.ReadFile(abs)
				if err != nil {
					return nil, fmt.Errorf("参照JSONを読めません: %s: %w", abs, err)
				}
				asset.Path = abs
				if err := json.Unmarshal(data, &asset.Value); err != nil {
					return nil, fmt.Errorf("参照JSONが不正です: %s: %w", abs, err)
				}
				cache[abs] = asset
				addSource(scenario, model.SourceRef{Kind: "json", Path: abs, SHA256: fileHash(data)})
			}
			result = append(result, asset)
		}
		return result, nil
	}
	for i := range scenario.Steps {
		step := &scenario.Steps[i]
		if step.SourcePath != scenario.SourcePath {
			continue
		}
		var request any
		if step.HTTP != nil {
			request = step.HTTP.Body
		} else if step.GRPC != nil {
			request = step.GRPC.Message
		}
		var err error
		step.RequestJSONData, err = load(referencedJSONFiles(request, vars, nil))
		if err != nil {
			return err
		}
		step.ExpectationJSONData, err = load(referencedJSONFiles("{{"+step.Test+"}}", vars, nil))
		if err != nil {
			return err
		}
	}
	return nil
}

var variablePathPattern = regexp.MustCompile(`\bvars(?:\.[A-Za-z_][A-Za-z0-9_]*)+`)
var templateExpressionPattern = regexp.MustCompile(`(?s)\{\{(.*?)\}\}`)

// Record only statically identifiable JSON sources. Runtime expressions and
// responses are never evaluated to invent a filename.
func resolveJSONFiles(scenario *model.Scenario, vars any) {
	for i := range scenario.Steps {
		step := &scenario.Steps[i]
		var request any
		if step.HTTP != nil {
			request = step.HTTP.Body
		} else if step.GRPC != nil {
			request = step.GRPC.Message
		}
		// Included steps have already been processed in their own variable scope.
		localVars := vars
		if step.SourcePath != scenario.SourcePath {
			localVars = nil
		}
		step.RequestJSONFiles = append(step.RequestJSONFiles, referencedJSONFiles(request, localVars, scenario.Cases)...)
		step.ExpectationJSONFiles = append(step.ExpectationJSONFiles, referencedJSONFiles("{{"+step.Test+"}}", localVars, scenario.Cases)...)
	}
}

func referencedJSONFiles(value, vars any, cases []model.Case) []string {
	files := map[string]bool{}
	var collectSource func(any)
	collectSource = func(value any) {
		switch v := value.(type) {
		case string:
			if strings.HasPrefix(v, "json://") && !strings.ContainsAny(v, "{}$") {
				path := strings.TrimPrefix(v, "json://")
				if strings.EqualFold(filepath.Ext(path), ".json") {
					files[path] = true
				}
			}
		case map[string]any:
			for _, child := range v {
				collectSource(child)
			}
		case []any:
			for _, child := range v {
				collectSource(child)
			}
		}
	}
	var visit func(any)
	visit = func(value any) {
		switch v := value.(type) {
		case string:
			collectSource(v)
			for _, expression := range templateExpressionPattern.FindAllString(v, -1) {
				for _, variable := range variablePathPattern.FindAllString(expression, -1) {
					parts := strings.Split(strings.TrimPrefix(variable, "vars."), ".")
					if parts[0] == "case" {
						for _, c := range cases {
							data := make(map[string]any, len(c.Data)+1)
							for key, item := range c.Data {
								data[key] = item
							}
							if len(c.Expectation) > 0 {
								data["expect"] = c.Expectation
							}
							if _, exists := resolveJSONVariable(data, parts[1:]); exists && strings.EqualFold(filepath.Ext(c.SourcePath), ".json") {
								files[c.SourcePath] = true
							}
						}
					}
					resolved, _ := resolveJSONVariable(vars, parts)
					collectSource(resolved)
				}
			}
		case map[string]any:
			for _, child := range v {
				visit(child)
			}
		case []any:
			for _, child := range v {
				visit(child)
			}
		}
	}
	visit(value)
	result := make([]string, 0, len(files))
	for path := range files {
		result = append(result, path)
	}
	sort.Strings(result)
	return result
}

func resolveJSONVariable(value any, parts []string) (any, bool) {
	for _, part := range parts {
		if ref, ok := value.(string); ok && strings.HasPrefix(ref, "json://") {
			return ref, true // The remaining fields come from this file.
		}
		mapping, ok := value.(map[string]any)
		if !ok {
			return nil, false
		}
		var exists bool
		value, exists = mapping[part]
		if !exists {
			return nil, false
		}
	}
	return value, true
}
