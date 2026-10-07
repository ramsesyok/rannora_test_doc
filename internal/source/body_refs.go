package source

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/ramsesyok/runnora-docgen/internal/model"
)

var (
	responseResultPattern = regexp.MustCompile(`\b(?:current|steps\.[A-Za-z_][A-Za-z0-9_]*)\.res\.(?:body|message|messages)(?:\.[A-Za-z_][A-Za-z0-9_]*|\[[^]]+\])*`)
	bodyFunctionPattern   = regexp.MustCompile(`\b(?:compare|diffEps)\s*\(`)
	bodyEqualityRight     = regexp.MustCompile(responseResultPattern.String() + `\s*(?:==|!=)\s*(` + variablePathPattern.String() + `)`)
	bodyEqualityLeft      = regexp.MustCompile(`(` + variablePathPattern.String() + `)\s*(?:==|!=)\s*` + responseResultPattern.String())
)

// responseBodyVarPaths selects only variables compared with a response body.
// Status and comparison options such as ignorePaths are intentionally excluded.
// Unknown expression forms yield no inferred body reference.
func responseBodyVarPaths(test string) []string {
	var paths []string
	for _, match := range bodyFunctionPattern.FindAllStringIndex(test, -1) {
		args := callArguments(test, match[1])
		if len(args) < 2 {
			continue
		}
		left, right := responseResultPattern.MatchString(args[0]), responseResultPattern.MatchString(args[1])
		if left == right {
			continue
		}
		other := args[0]
		if left {
			other = args[1]
		}
		paths = append(paths, variablePathPattern.FindAllString(other, -1)...)
	}
	for _, match := range bodyEqualityRight.FindAllStringSubmatch(test, -1) {
		paths = append(paths, match[1])
	}
	for _, match := range bodyEqualityLeft.FindAllStringSubmatch(test, -1) {
		paths = append(paths, match[1])
	}
	return uniqueStrings(paths)
}

func callArguments(test string, start int) []string {
	depth := 1
	quote := byte(0)
	escaped := false
	argStart := start
	var args []string
	for i := start; i < len(test); i++ {
		ch := test[i]
		if quote != 0 {
			if escaped {
				escaped = false
			} else if ch == '\\' {
				escaped = true
			} else if ch == quote {
				quote = 0
			}
			continue
		}
		switch ch {
		case '\'', '"':
			quote = ch
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return append(args, strings.TrimSpace(test[argStart:i]))
			}
		case ',':
			if depth == 1 {
				args = append(args, strings.TrimSpace(test[argStart:i]))
				argStart = i + 1
			}
		}
	}
	return nil
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	var result []string
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}

func appendRefs(existing, added []model.JSONRef) []model.JSONRef {
	seen := map[model.JSONRef]bool{}
	for _, ref := range existing {
		seen[ref] = true
	}
	for _, ref := range added {
		if !seen[ref] {
			seen[ref] = true
			existing = append(existing, ref)
		}
	}
	return existing
}

func (v includeVars) refsForVars(paths []string) []model.JSONRef {
	var refs []model.JSONRef
	for _, path := range paths {
		parts := strings.Split(strings.TrimPrefix(path, "vars."), ".")
		if file, ok := v.files[parts[0]]; ok {
			refs = append(refs, model.JSONRef{Path: file, FieldPath: strings.Join(parts[1:], ".")})
		}
	}
	return appendRefs(nil, refs)
}

func refsFromValue(value, vars any, cases []model.Case, baseDir string) []model.JSONRef {
	var refs []model.JSONRef
	var collect func(any)
	collect = func(value any) {
		switch typed := value.(type) {
		case string:
			if isStaticJSONRef(typed) {
				refs = append(refs, model.JSONRef{Path: resolveReference(baseDir, typed)})
			}
			for _, path := range varPaths(typed) {
				refs = append(refs, refsForVariable(path, vars, cases, baseDir)...)
			}
		case map[string]any:
			keys := make([]string, 0, len(typed))
			for key := range typed {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				collect(typed[key])
			}
		case []any:
			for _, child := range typed {
				collect(child)
			}
		}
	}
	collect(value)
	return appendRefs(nil, refs)
}

func refsForVariable(path string, vars any, cases []model.Case, baseDir string) []model.JSONRef {
	parts := strings.Split(strings.TrimPrefix(path, "vars."), ".")
	var refs []model.JSONRef
	if len(parts) > 1 && parts[0] == "case" {
		for _, c := range cases {
			data := make(map[string]any, len(c.Data)+1)
			for key, value := range c.Data {
				data[key] = value
			}
			data["expect"] = c.Expectation
			if _, ok := resolveJSONVariable(data, parts[1:]); ok {
				refs = append(refs, model.JSONRef{Path: c.SourcePath, FieldPath: strings.Join(parts[1:], ".")})
			}
		}
	}
	if ref, field, ok := variableFileRef(vars, parts); ok {
		refs = append(refs, model.JSONRef{Path: resolveReference(baseDir, ref), FieldPath: field})
	}
	return refs
}

func variableFileRef(value any, parts []string) (string, string, bool) {
	for i, part := range parts {
		if ref, ok := value.(string); ok && isStaticJSONRef(ref) {
			return ref, strings.Join(parts[i:], "."), true
		}
		mapping, ok := value.(map[string]any)
		if !ok {
			return "", "", false
		}
		value, ok = mapping[part]
		if !ok {
			return "", "", false
		}
	}
	if ref, ok := value.(string); ok && isStaticJSONRef(ref) {
		return ref, "", true
	}
	return "", "", false
}

func isStaticJSONRef(value string) bool {
	return strings.HasPrefix(value, "json://") && !strings.ContainsAny(value, "{}$") && strings.EqualFold(filepath.Ext(value), ".json")
}
