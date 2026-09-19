package source

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ramsesyok/runnora-test-instructions/internal/model"
)

var (
	packagePattern = regexp.MustCompile(`(?m)^\s*package\s+([\w.]+)\s*;`)
	servicePattern = regexp.MustCompile(`(?s)service\s+(\w+)\s*\{(.*?)\}`)
	rpcPattern     = regexp.MustCompile(`rpc\s+(\w+)\s*\(\s*(stream\s+)?[\w.]+\s*\)\s*returns\s*\(\s*(stream\s+)?[\w.]+\s*\)`)
)

type protoIndex map[string]model.RPCType

func loadProtoIndex(paths []string) (protoIndex, []string) {
	index := protoIndex{}
	var warnings []string
	seen := map[string]bool{}
	for _, path := range paths {
		abs, err := filepath.Abs(path)
		if err != nil || seen[abs] {
			continue
		}
		seen[abs] = true
		data, err := os.ReadFile(abs)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("proto を読めません: %s: %v", path, err))
			continue
		}
		pkg := ""
		if match := packagePattern.FindSubmatch(data); len(match) > 1 {
			pkg = string(match[1])
		}
		for _, service := range servicePattern.FindAllSubmatch(data, -1) {
			serviceName := string(service[1])
			for _, rpc := range rpcPattern.FindAllSubmatch(service[2], -1) {
				method := string(rpc[1])
				clientStream := strings.TrimSpace(string(rpc[2])) != ""
				serverStream := strings.TrimSpace(string(rpc[3])) != ""
				typeValue := model.RPCUnary
				switch {
				case clientStream:
					typeValue = model.RPCUnsupported
				case serverStream:
					typeValue = model.RPCServerStreaming
				}
				short := serviceName + "/" + method
				index[short] = typeValue
				if pkg != "" {
					index[pkg+"."+short] = typeValue
				}
			}
		}
	}
	return index, warnings
}
