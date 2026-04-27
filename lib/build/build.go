// Package build provides the core build system functionality for minibp.
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

// Options holds the command-line configuration options for the build system.
type Options struct {
	Arch     string
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
	TargetOS string
}

// BuildOptions is an alias for Options to avoid import cycles in utils.
type BuildOptions = Options

// Graph represents a dependency graph of modules.
type Graph struct {
	nodes map[string]*parser.Module
	edges map[string][]string
}

// NewGraph creates a new, empty dependency graph.
func NewGraph() *Graph {
	return &Graph{
		nodes: make(map[string]*parser.Module),
		edges: make(map[string][]string),
	}
}

// AddNode adds a module node to the dependency graph.
func (g *Graph) AddNode(name string, mod *parser.Module) {
	g.nodes[name] = mod
	if _, ok := g.edges[name]; !ok {
		g.edges[name] = []string{}
	}
}

// AddEdge adds a directed edge from one module to another.
func (g *Graph) AddEdge(from, to string) {
	if _, ok := g.edges[from]; !ok {
		g.edges[from] = []string{}
	}
	if _, ok := g.edges[to]; !ok {
		g.edges[to] = []string{}
	}
	g.edges[from] = append(g.edges[from], to)
}

// TopoSort performs a topological sort on the dependency graph using Kahn's algorithm.
func (g *Graph) TopoSort() ([][]string, error) {
	// depCount[node] = number of dependencies the node has
	depCount := make(map[string]int)
	for name := range g.nodes {
		depCount[name] = 0
	}

	// dependentOf[node] = list of nodes that depend on node
	dependentOf := make(map[string][]string)

	for from, deps := range g.edges {
		if _, ok := g.nodes[from]; !ok {
			return nil, fmt.Errorf("module '%s' referenced in dependency graph does not exist", from)
		}
		depCount[from] = len(deps)
		for _, to := range deps {
			if _, ok := g.nodes[to]; !ok {
				if !strings.HasPrefix(to, ":") && !strings.HasPrefix(to, "//") {
					continue
				}
				return nil, fmt.Errorf("dependency '%s' of '%s' not found", to, from)
			}
			dependentOf[to] = append(dependentOf[to], from)
		}
	}

	// Initialize the queue with all nodes having zero dependencies.
	var queue []string
	for name, count := range depCount {
		if count == 0 {
			queue = append(queue, name)
		}
	}
	sort.Strings(queue)

	var levels [][]string
	visitedCount := 0
	for len(queue) > 0 {
		currentLevel := make([]string, len(queue))
		copy(currentLevel, queue)
		levels = append(levels, currentLevel)
		visitedCount += len(queue)

		nextQueue := []string{}
		for _, u := range queue {
			for _, v := range dependentOf[u] {
				depCount[v]--
				if depCount[v] == 0 {
					nextQueue = append(nextQueue, v)
				}
			}
		}
		sort.Strings(nextQueue)
		queue = nextQueue
	}

	if visitedCount != len(g.nodes) {
		return nil, fmt.Errorf("circular dependency detected")
	}

	return levels, nil
}

// CollectModules collects all enabled modules from Blueprint definitions.
func CollectModules(allDefs []parser.Definition, eval *parser.Evaluator, opts Options) (map[string]*parser.Module, error) {
	modules, err := CollectModulesWithNames(allDefs, eval, opts, nil)
	if err != nil {
		return nil, err
	}
	if err := glob.ExpandGlobs(modules, opts.SrcDir); err != nil {
		return nil, fmt.Errorf("error during global glob expansion: %w", err)
	}
	for name, mod := range modules {
		if err := glob.ExpandInModule(mod, opts.SrcDir); err != nil {
			return nil, fmt.Errorf("error expanding globs for module %s: %w", name, err)
		}
	}
	return modules, nil
}

// CollectModulesWithNames collects modules using a custom name extraction function.
func CollectModulesWithNames(
	allDefs []parser.Definition,
	eval *parser.Evaluator,
	opts Options,
	nameFunc func(*parser.Module, string) string,
) (map[string]*parser.Module, error) {
	if nameFunc == nil {
		nameFunc = func(m *parser.Module, key string) string {
			return props.GetStringPropEval(m, key, eval)
		}
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
		variant.MergeVariantProps(mod, opts.Arch, true, eval)
		if !variant.IsModuleEnabledForTarget(mod, true) {
			continue
		}
		modules[name] = mod
	}

	if err := glob.ExpandGlobs(modules, opts.SrcDir); err != nil {
		return nil, fmt.Errorf("error during global glob expansion: %w", err)
	}

	for _, mod := range modules {
		if err := glob.ExpandInModule(mod, opts.SrcDir); err != nil {
			name := nameFunc(mod, "name")
			return nil, fmt.Errorf("error expanding globs for module %s: %w", name, err)
		}
	}

	return modules, nil
}

// BuildGraph constructs a dependency graph from a collection of modules.
func BuildGraph(modules map[string]*parser.Module, namespaces map[string]*namespace.Info, eval *parser.Evaluator) *Graph {
	graph := NewGraph()

	for name, mod := range modules {
		graph.AddNode(name, mod)
	}

	for name, mod := range modules {
		addResolvedDeps(graph, name, props.GetListPropEval(mod, "deps", eval), namespaces, true)
		addResolvedDeps(graph, name, props.GetListPropEval(mod, "shared_libs", eval), namespaces, true)
		addResolvedDeps(graph, name, props.GetListPropEval(mod, "header_libs", eval), namespaces, true)
		addResolvedDeps(graph, name, props.GetListPropEval(mod, "data", eval), namespaces, true)
	}
	return graph
}

// addResolvedDeps resolves dependency references and adds edges to the graph.
func addResolvedDeps(graph *Graph, from string, deps []string, namespaces map[string]*namespace.Info, moduleRefsOnly bool) {
	for _, dep := range deps {
		if moduleRefsOnly && !strings.HasPrefix(dep, ":") && !strings.HasPrefix(dep, "//") {
			continue
		}
		depName := namespace.ResolveModuleRef(dep, namespaces)
		graph.AddEdge(from, depName)
	}
}

// NewGenerator creates a ninja generator configured with the build options.
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
	gen.SetTargetOS(opts.TargetOS)
	if len(opts.Multilib) > 0 {
		gen.SetMultilib(opts.Multilib)
	}
	return gen
}

// pathPrefixForOutput calculates the relative path prefix from build directory to source directory.
func pathPrefixForOutput(srcDir, outFile string) string {
	absBuildDir, err := filepath.Abs(filepath.Dir(outFile))
	if err != nil {
		return ""
	}
	absSourceDir, err := filepath.Abs(srcDir)
	if err != nil {
		return ""
	}

	if absBuildDir == absSourceDir {
		return ""
	}

	relPath, err := filepath.Rel(absBuildDir, absSourceDir)
	if err != nil {
		return ""
	}

	if relPath == "." {
		return ""
	}

	return filepath.ToSlash(relPath) + "/"
}

// buildRegenCmd constructs the regeneration command for ninja build rules.
func buildRegenCmd(opts Options) string {
	exe := filepath.Base(os.Args[0])

	regenCmd := filepath.ToSlash(exe)
	if opts.Arch != "" {
		regenCmd += " -arch " + opts.Arch
	}
	if len(opts.Inputs) == 1 {
		fi, err := os.Stat(opts.Inputs[0])
		if err == nil && fi.IsDir() {
			regenCmd += " -a"
		}
	}
	regenCmd += " -o " + opts.OutFile
	if len(opts.Inputs) > 0 {
		regenCmd += " " + strings.Join(opts.Inputs, " ")
	}
	return regenCmd
}

// toolchainFromOptions creates a Toolchain configuration from build options.
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
