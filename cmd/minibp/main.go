// cmd/minibp/main.go - CLI entry point
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"minibp/ninja"
	"minibp/parser"
)

// Simple graph structure for dependency ordering
type Graph struct {
	nodes map[string]*parser.Module
	edges map[string][]string
}

func NewGraph() *Graph {
	return &Graph{
		nodes: make(map[string]*parser.Module),
		edges: make(map[string][]string),
	}
}

func (g *Graph) AddNode(name string, mod *parser.Module) {
	g.nodes[name] = mod
	if _, ok := g.edges[name]; !ok {
		g.edges[name] = []string{}
	}
}

func (g *Graph) AddEdge(from, to string) {
	if _, ok := g.edges[from]; !ok {
		g.edges[from] = []string{}
	}
	if _, ok := g.edges[to]; !ok {
		g.edges[to] = []string{}
	}
	g.edges[from] = append(g.edges[from], to)
}

func (g *Graph) TopoSort() ([][]string, error) {
	inDegree := make(map[string]int)
	for name := range g.nodes {
		inDegree[name] = 0
	}

	for from, deps := range g.edges {
		for _, to := range deps {
			if _, ok := g.nodes[to]; !ok {
				return nil, fmt.Errorf("dependency '%s' of '%s' not found", to, from)
			}
			inDegree[from]++
		}
	}

	reverseEdges := make(map[string][]string)
	for from, deps := range g.edges {
		for _, to := range deps {
			reverseEdges[to] = append(reverseEdges[to], from)
		}
	}

	var levels [][]string
	visited := make(map[string]bool)
	nodeCount := len(g.nodes)

	for len(visited) < nodeCount {
		var currentLevel []string
		for name, degree := range inDegree {
			if degree == 0 && !visited[name] {
				currentLevel = append(currentLevel, name)
			}
		}

		if len(currentLevel) == 0 {
			return nil, fmt.Errorf("circular dependency detected")
		}

		levels = append(levels, currentLevel)
		for _, name := range currentLevel {
			visited[name] = true
			for _, dependent := range reverseEdges[name] {
				inDegree[dependent]--
			}
		}
	}

	return levels, nil
}

func main() {
	var (
		outFile  = flag.String("o", "build.ninja", "output ninja file")
		all      = flag.Bool("a", false, "parse all .bp files in directory")
		ccFlag   = flag.String("cc", "", "C compiler (default: gcc)")
		cxxFlag  = flag.String("cxx", "", "C++ compiler (default: g++)")
		arFlag   = flag.String("ar", "", "archiver (default: ar)")
		archFlag = flag.String("arch", "", "target architecture (arm, arm64, x86, x86_64)")
	)
	flag.Parse()

	if len(flag.Args()) < 1 && !*all {
		fmt.Fprintln(os.Stderr, "Usage: minibp [-o output] [-a] [-cc CC] [-cxx CXX] [-ar AR] [-arch ARCH] <file.bp | directory>")
		os.Exit(1)
	}

	// Determine source directory
	srcDir := "."
	if *all && len(flag.Args()) > 0 {
		srcDir = flag.Args()[0]
	} else if len(flag.Args()) > 0 {
		// For single file, use the directory containing the .bp file
		srcDir = filepath.Dir(flag.Args()[0])
	}

	// Collect all .bp files
	var files []string
	if *all {
		bpFiles, err := filepath.Glob(filepath.Join(srcDir, "*.bp"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		files = bpFiles
	} else {
		files = flag.Args()
	}

	// Parse all files
	modules := make(map[string]*parser.Module)
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening %s: %v\n", file, err)
			os.Exit(1)
		}
		defer f.Close()

		parsedFile, err := parser.ParseFile(f, file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
			os.Exit(1)
		}

		for _, def := range parsedFile.Defs {
			if mod, ok := def.(*parser.Module); ok {
				name := getStringProp(mod, "name")
				if name != "" {
					// Expand globs in srcs before storing module
					expandGlobsInModule(mod, srcDir)
					// Merge arch-specific overrides if target arch is set
					if *archFlag != "" && mod.Arch != nil {
						mergeArchProps(mod, *archFlag)
					}
					modules[name] = mod
				}
			}
		}
	}

	// Build dependency graph
	graph := NewGraph()
	for name, mod := range modules {
		graph.AddNode(name, mod)
	}

	// Add edges for deps
	for name, mod := range modules {
		deps := getListProp(mod, "deps")
		for _, dep := range deps {
			depName := strings.TrimPrefix(dep, ":")
			graph.AddEdge(name, depName)
		}
		sharedLibs := getListProp(mod, "shared_libs")
		for _, dep := range sharedLibs {
			depName := strings.TrimPrefix(dep, ":")
			graph.AddEdge(name, depName)
		}
	}

	// Get all rules
	rules := ninja.GetAllRules()
	ruleMap := make(map[string]ninja.BuildRule)
	for _, r := range rules {
		ruleMap[r.Name()] = r
	}

	// Generate ninja file
	out, err := os.Create(*outFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output: %v\n", err)
		os.Exit(1)
	}
	defer out.Close()

	// Calculate relative path from output dir to source dir
	absOutFile, _ := filepath.Abs(*outFile)
	absBuildDir := filepath.Dir(absOutFile)
	absSourceDir, _ := filepath.Abs(srcDir)

	// Only add prefix if source and build directories are different
	prefix := ""
	if absBuildDir != absSourceDir {
		relPath, err := filepath.Rel(absBuildDir, absSourceDir)
		if err == nil && relPath != "." {
			prefix = relPath + "/"
		}
	}

	outDir := filepath.Dir(absOutFile)

	gen := ninja.NewGenerator(graph, ruleMap, modules)
	gen.SetSourceDir(srcDir)
	gen.SetOutputDir(outDir)
	gen.SetPathPrefix(prefix)

	regenCmd := os.Args[0] + " -o " + *outFile
	for _, f := range files {
		regenCmd += " " + f
	}
	gen.SetRegen(regenCmd, files, *outFile)
	gen.SetWorkDir(srcDir)

	tc := ninja.DefaultToolchain()
	if *ccFlag != "" {
		tc.CC = *ccFlag
	}
	if *cxxFlag != "" {
		tc.CXX = *cxxFlag
	}
	if *arFlag != "" {
		tc.AR = *arFlag
	}
	gen.SetToolchain(tc)
	gen.SetArch(*archFlag)

	if err := gen.Generate(out); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating ninja: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s with %d modules\n", *outFile, len(modules))
}

func getStringProp(m *parser.Module, name string) string {
	if m.Map == nil {
		return ""
	}
	for _, prop := range m.Map.Properties {
		if prop.Name == name {
			if s, ok := prop.Value.(*parser.String); ok {
				return s.Value
			}
		}
	}
	return ""
}

func getListProp(m *parser.Module, name string) []string {
	if m.Map == nil {
		return nil
	}
	for _, prop := range m.Map.Properties {
		if prop.Name == name {
			if l, ok := prop.Value.(*parser.List); ok {
				var result []string
				for _, v := range l.Values {
					if s, ok := v.(*parser.String); ok {
						result = append(result, s.Value)
					}
				}
				return result
			}
		}
	}
	return nil
}

// mergeArchProps merges arch-specific properties into the module's main Map.
// For list properties (srcs, cflags, etc.), arch values are appended.
// For scalar properties (strings), arch values override base values.
func mergeArchProps(m *parser.Module, arch string) {
	archMap, ok := m.Arch[arch]
	if !ok || archMap == nil {
		return
	}
	for _, prop := range archMap.Properties {
		switch prop.Value.(type) {
		case *parser.List:
			// Append to existing list
			merged := false
			for _, baseProp := range m.Map.Properties {
				if baseProp.Name == prop.Name {
					if baseList, ok := baseProp.Value.(*parser.List); ok {
						if archList, ok := prop.Value.(*parser.List); ok {
							baseList.Values = append(baseList.Values, archList.Values...)
						}
					}
					merged = true
					break
				}
			}
			if !merged {
				m.Map.Properties = append(m.Map.Properties, prop)
			}
		default:
			// Override scalar value
			found := false
			for i, baseProp := range m.Map.Properties {
				if baseProp.Name == prop.Name {
					m.Map.Properties[i] = prop
					found = true
					break
				}
			}
			if !found {
				m.Map.Properties = append(m.Map.Properties, prop)
			}
		}
	}
}

// expandGlobsInModule expands glob patterns in module srcs
func expandGlobsInModule(m *parser.Module, baseDir string) {
	if m.Map == nil {
		return
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
							matches := expandGlob(pattern, baseDir)
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

				if len(expandedSrcs) > 0 {
					l.Values = expandedSrcs
				}
			}
		}
	}
}

func expandGlob(pattern, baseDir string) []string {
	var result []string

	if strings.Contains(pattern, "**") {
		dir := baseDir
		suffix := ""
		if idx := strings.Index(pattern, "/**"); idx >= 0 {
			dir = filepath.Join(baseDir, pattern[:idx])
			suffix = pattern[idx+3:]
		}

		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			relPath, _ := filepath.Rel(baseDir, path)
			if suffix == "" || strings.HasSuffix(path, suffix) {
				result = append(result, relPath)
			}
			return nil
		})
	} else {
		fullPattern := filepath.Join(baseDir, pattern)
		matches, _ := filepath.Glob(fullPattern)
		for _, match := range matches {
			relPath, _ := filepath.Rel(baseDir, match)
			result = append(result, relPath)
		}
	}

	return result
}
