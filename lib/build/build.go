package build

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"minibp/lib/glob"
	"minibp/lib/namespace"
	"minibp/lib/ninja"
	"minibp/lib/parser"
	"minibp/lib/props"
	"minibp/lib/variant"
)

type Options struct {
	Arch     string
	Host     bool
	SrcDir   string
	OutFile  string
	Inputs   []string
	Multilib []string
	CC       string
	CXX      string
	AR       string
	LTO      string
	Sysroot  string
	Ccache   string
}

// Graph is the dependency graph used by ninja generation.
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
		if _, ok := g.nodes[from]; !ok {
			return nil, fmt.Errorf("module '%s' referenced in dependency graph does not exist", from)
		}
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
	for len(visited) < len(g.nodes) {
		var currentLevel []string
		for name, degree := range inDegree {
			if degree == 0 && !visited[name] {
				currentLevel = append(currentLevel, name)
			}
		}
		if len(currentLevel) == 0 {
			return nil, fmt.Errorf("circular dependency detected")
		}
		sort.Strings(currentLevel)
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

func CollectModules(allDefs []parser.Definition, eval *parser.Evaluator, opts Options) (map[string]*parser.Module, error) {
	modules := make(map[string]*parser.Module)
	for _, def := range allDefs {
		mod, ok := def.(*parser.Module)
		if !ok {
			continue
		}
		name := props.GetStringPropEval(mod, "name", eval)
		if name == "" {
			continue
		}
		eval.EvalModule(mod)
		variant.MergeVariantProps(mod, opts.Arch, opts.Host, eval)
		if err := glob.ExpandInModule(mod, opts.SrcDir); err != nil {
			return nil, fmt.Errorf("error expanding globs for module %s: %w", name, err)
		}
		if !variant.IsModuleEnabledForTarget(mod, opts.Host) {
			continue
		}
		modules[name] = mod
	}
	return modules, nil
}

func CollectModulesWithNames(
	allDefs []parser.Definition,
	eval *parser.Evaluator,
	opts Options,
	nameFunc func(*parser.Module, string) string,
) (map[string]*parser.Module, error) {
	if nameFunc == nil {
		nameFunc = props.GetStringProp
	}

	modules := make(map[string]*parser.Module)
	for _, def := range allDefs {
		mod, ok := def.(*parser.Module)
		if !ok {
			continue
		}
		name := nameFunc(mod, "name")
		if name == "" {
			continue
		}
		eval.EvalModule(mod)
		variant.MergeVariantProps(mod, opts.Arch, opts.Host, eval)
		if err := glob.ExpandInModule(mod, opts.SrcDir); err != nil {
			return nil, fmt.Errorf("error expanding globs for module %s: %w", name, err)
		}
		if !variant.IsModuleEnabledForTarget(mod, opts.Host) {
			continue
		}
		modules[name] = mod
	}
	return modules, nil
}

func BuildGraph(modules map[string]*parser.Module, namespaces map[string]*namespace.Info, eval *parser.Evaluator) *Graph {
	graph := NewGraph()
	for name, mod := range modules {
		graph.AddNode(name, mod)
	}
	for name, mod := range modules {
		addResolvedDeps(graph, name, props.GetListPropEval(mod, "deps", eval), namespaces, false)
		addResolvedDeps(graph, name, props.GetListPropEval(mod, "shared_libs", eval), namespaces, false)
		addResolvedDeps(graph, name, props.GetListPropEval(mod, "header_libs", eval), namespaces, false)
		addResolvedDeps(graph, name, props.GetListPropEval(mod, "data", eval), namespaces, true)
	}
	return graph
}

func addResolvedDeps(graph *Graph, from string, deps []string, namespaces map[string]*namespace.Info, moduleRefsOnly bool) {
	for _, dep := range deps {
		if moduleRefsOnly && !strings.HasPrefix(dep, ":") && !strings.HasPrefix(dep, "//") {
			continue
		}
		depName := namespace.ResolveModuleRef(dep, namespaces)
		graph.AddEdge(from, depName)
	}
}

func NewGenerator(graph *Graph, modules map[string]*parser.Module, opts Options) *ninja.Generator {
	ruleMap := make(map[string]ninja.BuildRule)
	for _, r := range ninja.GetAllRules() {
		ruleMap[r.Name()] = r
	}

	absOutFile, _ := filepath.Abs(opts.OutFile)
	outDir := filepath.Dir(absOutFile)
	prefix := pathPrefixForOutput(opts.SrcDir, absOutFile)

	gen := ninja.NewGenerator(graph, ruleMap, modules)
	gen.SetSourceDir(opts.SrcDir)
	gen.SetOutputDir(outDir)
	gen.SetPathPrefix(prefix)
	gen.SetRegen(buildRegenCmd(opts), opts.Inputs, opts.OutFile)
	gen.SetWorkDir(opts.SrcDir)
	gen.SetToolchain(toolchainFromOptions(opts))
	gen.SetArch(opts.Arch)
	if len(opts.Multilib) > 0 {
		gen.SetMultilib(opts.Multilib)
	}
	return gen
}

func pathPrefixForOutput(srcDir, outFile string) string {
	absBuildDir := filepath.Dir(outFile)
	absSourceDir, _ := filepath.Abs(srcDir)
	if absBuildDir == absSourceDir {
		return ""
	}
	relPath, err := filepath.Rel(absBuildDir, absSourceDir)
	if err != nil || relPath == "." {
		return ""
	}
	return filepath.ToSlash(relPath) + "/"
}

func buildRegenCmd(opts Options) string {
	regenCmd := os.Args[0] + " -o " + opts.OutFile
	for _, f := range opts.Inputs {
		regenCmd += " " + f
	}
	return regenCmd
}

func toolchainFromOptions(opts Options) ninja.Toolchain {
	tc := ninja.DefaultToolchain()
	if opts.CC != "" {
		tc.CC = opts.CC
	}
	if opts.CXX != "" {
		tc.CXX = opts.CXX
	}
	if opts.AR != "" {
		tc.AR = opts.AR
	}
	if opts.Sysroot != "" {
		tc.Sysroot = opts.Sysroot
	}
	if opts.LTO != "" {
		tc.Lto = opts.LTO
	}
	if opts.Ccache == "no" {
		tc.Ccache = ""
	} else if opts.Ccache != "" {
		tc.Ccache = opts.Ccache
	}
	return tc
}
