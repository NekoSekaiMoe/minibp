// Package dependency provides advanced dependency management features including
// transitive dependency resolution, conflict detection, and dependency graph
// visualization.
package dependency

import (
	"fmt"
	"sort"
	"strings"
)

// Dependency represents a module dependency
type Dependency struct {
	Name     string
	Version  string
	Optional bool
}

// DependencyGraph represents the complete dependency graph
type DependencyGraph struct {
	modules     map[string]*ModuleNode
	edges       map[string][]string // module -> dependencies
	reverseEdges map[string][]string // module -> dependents
}

// ModuleNode represents a node in the dependency graph
type ModuleNode struct {
	Name       string
	Type       string
	DirectDeps []Dependency
	AllDeps    []Dependency // Transitive dependencies
	Dependents []string     // Modules that depend on this module
	IsRoot     bool         // True if this is a root module (not a dependency)
}

// Conflict represents a dependency conflict
type Conflict struct {
	Module   string
	DepName  string
	Version1 string
	Version2 string
	Path1    []string
	Path2    []string
}

// Resolution represents a dependency resolution result
type Resolution struct {
	Success   bool
	Conflicts []Conflict
	Order     []string // Topological order
}

// NewDependencyGraph creates a new dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		modules:     make(map[string]*ModuleNode),
		edges:       make(map[string][]string),
		reverseEdges: make(map[string][]string),
	}
}

// AddModule adds a module to the dependency graph
func (g *DependencyGraph) AddModule(name, moduleType string, deps []Dependency) {
	if _, exists := g.modules[name]; exists {
		return
	}
	node := &ModuleNode{
		Name:     name,
		Type:     moduleType,
		DirectDeps: deps,
		IsRoot:   true,
	}
	g.modules[name] = node
	g.edges[name] = make([]string, 0)
	g.reverseEdges[name] = make([]string, 0)

	// Add edges for dependencies
	for _, dep := range deps {
		g.edges[name] = append(g.edges[name], dep.Name)
		g.reverseEdges[dep.Name] = append(g.reverseEdges[dep.Name], name)
	}
}

// ResolveDependencies resolves all dependencies and detects conflicts
func (g *DependencyGraph) ResolveDependencies() *Resolution {
	resolution := &Resolution{
		Success:   true,
		Conflicts: make([]Conflict, 0),
	}

	// Calculate transitive dependencies for all modules
	for moduleName := range g.modules {
		g.calculateTransitiveDeps(moduleName)
	}

	// Detect conflicts
	conflicts := g.detectConflicts()
	if len(conflicts) > 0 {
		resolution.Success = false
		resolution.Conflicts = conflicts
	}

	// Calculate topological order
	order, err := g.topologicalSort()
	if err != nil {
		resolution.Success = false
		return resolution
	}
	resolution.Order = order

	return resolution
}

// calculateTransitiveDeps calculates all transitive dependencies for a module
func (g *DependencyGraph) calculateTransitiveDeps(moduleName string) {
	visited := make(map[string]bool)
	var deps []Dependency

	var visit func(name string)
	visit = func(name string) {
		if visited[name] {
			return
		}
		visited[name] = true
		if node, exists := g.modules[name]; exists {
			for _, dep := range node.DirectDeps {
				deps = append(deps, dep)
				visit(dep.Name)
			}
		}
	}
	visit(moduleName)

	// Remove duplicates
	seen := make(map[string]bool)
	var uniqueDeps []Dependency
	for _, dep := range deps {
		if !seen[dep.Name] {
			seen[dep.Name] = true
			uniqueDeps = append(uniqueDeps, dep)
		}
	}

	if node, exists := g.modules[moduleName]; exists {
		node.AllDeps = uniqueDeps
	}
}

// detectConflicts detects version conflicts in dependencies
func (g *DependencyGraph) detectConflicts() []Conflict {
	var conflicts []Conflict

	// Track which versions of each dependency are required
	requiredVersions := make(map[string]map[string][]string) // dep -> version -> [modules]
	for moduleName, node := range g.modules {
		for _, dep := range node.DirectDeps {
			if _, exists := requiredVersions[dep.Name]; !exists {
				requiredVersions[dep.Name] = make(map[string][]string)
			}
			requiredVersions[dep.Name][dep.Version] = append(requiredVersions[dep.Name][dep.Version], moduleName)
		}
	}

	// Check for conflicts (multiple versions of same dependency)
	for depName, versions := range requiredVersions {
		if len(versions) > 1 {
			// Conflict detected
			conflict := Conflict{
				DepName:  depName,
				Version1: "",
				Version2: "",
			}
			for version, modules := range versions {
				if conflict.Version1 == "" {
					conflict.Version1 = version
					conflict.Path1 = modules
				} else {
					conflict.Version2 = version
					conflict.Path2 = modules
					break
				}
			}
			conflicts = append(conflicts, conflict)
		}
	}
	return conflicts
}

// topologicalSort performs topological sort on the dependency graph

func (g *DependencyGraph) topologicalSort() ([]string, error) {

	inDegree := make(map[string]int)

	for name := range g.modules {

		inDegree[name] = 0

	}



	// Calculate in-degrees (count how many modules depend on each module)

	for moduleName, deps := range g.edges {

		for _, dep := range deps {

			// moduleName depends on dep, so dep should come before moduleName

			// We need to track how many modules each module depends on

			if _, exists := inDegree[moduleName]; exists {

				// Count dependencies, not dependents

			}

			_ = moduleName

			_ = dep

		}

	}



	// Recalculate: in-degree = number of dependencies

	for name := range g.modules {

		inDegree[name] = len(g.edges[name])

	}



	// Initialize queue with nodes having in-degree 0 (no dependencies)

	queue := []string{}

	for name, degree := range inDegree {

		if degree == 0 {

			queue = append(queue, name)

		}

	}



	// Sort queue for deterministic order

	sort.Strings(queue)



	result := []string{}

	processed := make(map[string]bool)



	for len(queue) > 0 {

		// Take first element

		node := queue[0]

		queue = queue[1:]

		result = append(result, node)

		processed[node] = true



		// Find modules that depend on this node and reduce their effective in-degree

		for moduleName, deps := range g.edges {

			if processed[moduleName] {

				continue

			}

			// Check if this module depends on the current node

			hasDep := false

			for _, dep := range deps {

				if dep == node {

					hasDep = true

					break

				}

			}

			if hasDep {

				// Reduce in-degree by checking if all dependencies are processed

				allProcessed := true

				for _, dep := range g.edges[moduleName] {

					if !processed[dep] && dep != node {

						allProcessed = false

						break

					}

				}

								if allProcessed {

									// Check if already in queue

									found := false

									for _, q := range queue {

										if q == moduleName {

											found = true

											break

										}

									}

									if !found {

										queue = append(queue, moduleName)

										sort.Strings(queue)

									}

								}

			}

		}

	}



	// Check for cycle

	if len(result) != len(g.modules) {

		return nil, fmt.Errorf("circular dependency detected")

	}



	return result, nil

}

// GetDependents returns all modules that depend on the given module
func (g *DependencyGraph) GetDependents(moduleName string) []string {
	return g.reverseEdges[moduleName]
}

// GetDependencies returns all dependencies of a module
func (g *DependencyGraph) GetDependencies(moduleName string) []string {
	return g.edges[moduleName]
}

// Visualize generates a text representation of the dependency graph
func (g *DependencyGraph) Visualize() string {
	var sb strings.Builder
	sb.WriteString("Dependency Graph:\n")
	sb.WriteString(strings.Repeat("-", 40) + "\n")

	for name, node := range g.modules {
		sb.WriteString(fmt.Sprintf("%s (%s)\n", name, node.Type))
		if len(node.DirectDeps) > 0 {
			for _, dep := range node.DirectDeps {
				sb.WriteString(fmt.Sprintf(" -> %s [%s]\n", dep.Name, dep.Version))
			}
		} else {
			sb.WriteString(" (no dependencies)\n")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// GetModule returns a module node by name
func (g *DependencyGraph) GetModule(name string) (*ModuleNode, bool) {
	node, exists := g.modules[name]
	return node, exists
}

// GetAllModules returns all module nodes
func (g *DependencyGraph) GetAllModules() []*ModuleNode {
	modules := make([]*ModuleNode, 0, len(g.modules))
	for _, node := range g.modules {
		modules = append(modules, node)
	}
	return modules
}

// String returns a string representation of the graph
func (g *DependencyGraph) String() string {
	return g.Visualize()
}