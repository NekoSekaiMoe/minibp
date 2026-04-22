package glob

import (
	"minibp/lib/parser"
	"os"
	"path/filepath"
	"strings"
)

func ExpandInModule(m *parser.Module, baseDir string) error {
	if m.Map == nil {
		return nil
	}
	for _, prop := range m.Map.Properties {
		if prop.Name == "srcs" {
			if l, ok := prop.Value.(*parser.List); ok {
				var expandedSrcs []parser.Expression
				seen := make(map[string]bool)
				for _, v := range l.Values {
					if s, ok := v.(*parser.String); ok {
						pattern := s.Value
						if strings.Contains(pattern, "*") {
							matches, err := expandGlob(pattern, baseDir)
							if err != nil {
								return err
							}
							for _, match := range matches {
								if !seen[match] {
									seen[match] = true
									expandedSrcs = append(expandedSrcs, &parser.String{Value: match})
								}
							}
						} else {
							if !seen[pattern] {
								seen[pattern] = true
								expandedSrcs = append(expandedSrcs, v)
							}
						}
					}
				}
				l.Values = expandedSrcs
			}
		}
	}
	return nil
}

func expandGlob(pattern, baseDir string) ([]string, error) {
	var result []string
	if strings.Contains(pattern, "**") {
		walkDir := recursiveGlobRoot(pattern, baseDir)
		err := filepath.Walk(walkDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}
			relPath, err := filepath.Rel(baseDir, path)
			if err != nil {
				return err
			}
			relPath = filepath.ToSlash(relPath)
			if matchRecursivePattern(filepath.ToSlash(pattern), relPath) {
				result = append(result, relPath)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		fullPattern := filepath.Join(baseDir, pattern)
		matches, err := filepath.Glob(fullPattern)
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			relPath, err := filepath.Rel(baseDir, match)
			if err != nil {
				return nil, err
			}
			result = append(result, relPath)
		}
	}
	return result, nil
}

func recursiveGlobRoot(pattern, baseDir string) string {
	parts := strings.Split(filepath.ToSlash(pattern), "/")
	prefix := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "**" || strings.ContainsAny(part, "*?[") {
			break
		}
		prefix = append(prefix, part)
	}
	if len(prefix) == 0 {
		return baseDir
	}
	return filepath.Join(append([]string{baseDir}, prefix...)...)
}

func matchRecursivePattern(pattern, path string) bool {
	return matchRecursiveParts(splitGlobParts(pattern), splitGlobParts(path))
}

func splitGlobParts(path string) []string {
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}

func matchRecursiveParts(patternParts, pathParts []string) bool {
	if len(patternParts) == 0 {
		return len(pathParts) == 0
	}
	if patternParts[0] == "**" {
		if matchRecursiveParts(patternParts[1:], pathParts) {
			return true
		}
		if len(pathParts) == 0 {
			return false
		}
		return matchRecursiveParts(patternParts, pathParts[1:])
	}
	if len(pathParts) == 0 {
		return false
	}
	ok, err := filepath.Match(patternParts[0], pathParts[0])
	if err != nil || !ok {
		return false
	}
	return matchRecursiveParts(patternParts[1:], pathParts[1:])
}
