// Package dag provides functionality for building and analyzing
// directed acyclic graphs (DAGs) of module dependencies.
// This is used for determining build order and parallel execution of modules.
package dag

import (
	"fmt"
	"sort"

	"minibp/lib/module"
)

// Graph represents a directed acyclic graph of module dependencies.
// The graph tracks which modules exist and their dependency relationships,
// enabling topological sorting for build order determination.
//
// The graph uses two internal data structures:
//   - modules: maps module names to their Module objects
//   - edges: maps module names to lists of their direct dependencies
type Graph struct {
	// modules stores all modules in the graph, keyed by module name
	modules map[string]module.Module
	// edges stores dependency relationships: key is dependent module, value is list of dependencies
	edges map[string][]string
}

// NewGraph creates a new empty dependency graph.
// Returns a pointer to a newly initialized Graph with empty maps
// for both modules and edges. This is the starting point for building
// a dependency graph - modules and edges can be added using
// AddModule and AddEdge methods respectively.
func NewGraph() *Graph {
	return &Graph{
		modules: make(map[string]module.Module),
		edges:   make(map[string][]string),
	}
}

// AddModule adds a module to the graph.
// The module must implement the module.Module interface.
// If the provided module is nil, no action is taken.
// After adding, the module can be referenced by its Name() for
// adding edges and retrieving dependencies.
//
// This method also initializes an empty edges list for the module
// if one doesn't already exist, allowing for modules with no dependencies.
func (g *Graph) AddModule(m module.Module) {
	if m != nil {
		g.modules[m.Name()] = m
		// Initialize edges slice if not exists
		if _, exists := g.edges[m.Name()]; !exists {
			g.edges[m.Name()] = []string{}
		}
	}
}

// AddEdge adds a dependency edge from 'from' to 'to'.
// This represents that the module 'from' depends on module 'to'.
// In other words, 'to' must be built/processed before 'from'.
//
// This method ensures both modules exist in the edges map by
// initializing empty slices if they don't already exist.
// It then appends 'to' to the list of dependencies for 'from'.
//
// Parameters:
//   - from: the name of the dependent module (the module that depends on another)
//   - to: the name of the dependency module (the module that must be processed first)
//
// Note: AddEdge does not validate that the modules actually exist in the
// graph; this validation is performed during TopoSort.
func (g *Graph) AddEdge(from, to string) {
	// Ensure both modules exist in edges map
	if _, exists := g.edges[from]; !exists {
		g.edges[from] = []string{}
	}
	if _, exists := g.edges[to]; !exists {
		g.edges[to] = []string{}
	}
	g.edges[from] = append(g.edges[from], to)
}

// GetDeps returns the direct dependencies of a module by name.
// A copy of the dependency slice is returned to prevent external
// modification of the internal state.
//
// Parameters:
//   - name: the name of the module to get dependencies for
//
// Returns:
//   - A slice of strings containing the names of all direct dependencies
//   - An empty slice if the module doesn't exist or has no dependencies
func (g *Graph) GetDeps(name string) []string {
	if deps, exists := g.edges[name]; exists {
		// Return a copy to prevent external modification
		result := make([]string, len(deps))
		copy(result, deps)
		return result
	}
	return []string{}
}

// TopoSort returns modules in topological order, grouped by levels for parallel execution.
// This function performs a Kahn's algorithm-based topological sort and organizes
// the result into levels where modules at the same level can be executed in parallel.
//
// The algorithm works as follows:
// 1. Calculate in-degree (number of dependencies) for each module
// 2. Build reverse edges to track which modules depend on each module
// 3. Iteratively find all modules with in-degree 0 (modules with no remaining dependencies)
// 4. Process these modules, reduce in-degree of their dependents, and move to next level
// 5. Continue until all modules are processed or a cycle is detected
//
// Returns:
//   - [][]string: A slice of slices, where each inner slice represents a level.
//     Modules at the same level have no dependencies on each other and can run in parallel.
//     Level 0 contains modules with no dependencies (leaf nodes), level 1 contains
//     modules that depend only on level 0 modules, and so on.
//   - error: Returns an error if there's a cycle in the graph (which would make it
//     impossible to determine a valid build order), or if referenced modules don't exist.
//
// Example return value: [["D"], ["B", "C"], ["A"]] means:
//   - Level 0: D (no dependencies)
//   - Level 1: B and C (both depend only on D)
//   - Level 2: A (depends on both B and C)
func (g *Graph) TopoSort() ([][]string, error) {
	// Calculate in-degree for each node (number of dependencies)
	// In-degree represents how many modules this module depends on
	// that haven't been processed yet
	inDegree := make(map[string]int)
	for name := range g.modules {
		inDegree[name] = 0
	}

	// Validate dependencies and count in-degrees
	// For each module, count its dependencies and validate they exist
	for from, deps := range g.edges {
		if _, exists := g.modules[from]; !exists {
			return nil, fmt.Errorf("module '%s' referenced in dependency graph does not exist", from)
		}
		for _, to := range deps {
			if _, exists := g.modules[to]; !exists {
				return nil, fmt.Errorf("dependency '%s' of module '%s' does not exist", to, from)
			}
			// "from" depends on "to", so "from" has an incoming edge
			inDegree[from]++
		}
	}

	// Build reverse edges: dependency -> list of dependents that need it
	// When a dependency is processed, we notify its dependents
	// This allows us to reduce the in-degree of dependents when their dependency is completed
	reverseEdges := make(map[string][]string)
	for from, deps := range g.edges {
		for _, to := range deps {
			reverseEdges[to] = append(reverseEdges[to], from)
		}
	}

	// levels will contain the result: each element is a slice of module names
	// that can be executed in parallel at that level
	var levels [][]string
	visited := make(map[string]bool)
	nodeCount := len(g.modules)

	// Continue processing until all modules have been assigned to a level
	for len(visited) < nodeCount {
		// Find all nodes with in-degree 0 that haven't been visited
		// These are modules whose dependencies have all been processed
		var currentLevel []string
		for name, degree := range inDegree {
			if degree == 0 && !visited[name] {
				currentLevel = append(currentLevel, name)
			}
		}

		// If no nodes with in-degree 0 found but not all visited, there's a cycle
		// This means the graph is not a valid DAG and cannot be sorted
		if len(currentLevel) == 0 {
			// Identify nodes in the cycle for error message
			var remaining []string
			for name := range g.modules {
				if !visited[name] {
					remaining = append(remaining, name)
				}
			}
			// Build cycle description with dependency information
			cycleInfo := fmt.Sprintf("cycle detected in dependency graph involving modules: %v", remaining)
			// Add dependency chain information
			for _, name := range remaining {
				deps := g.GetDeps(name)
				if len(deps) > 0 {
					cycleInfo += fmt.Sprintf("; %s depends on %v", name, deps)
				}
			}
			return nil, fmt.Errorf("%s", cycleInfo)
		}

		// Sort level for deterministic output
		// This ensures consistent ordering across runs
		sort.Strings(currentLevel)

		// Mark current level as visited
		// These modules are now "processed" and ready
		for _, name := range currentLevel {
			visited[name] = true
		}

		// Reduce in-degree of neighbors
		// For each module processed at this level, inform its dependents
		// that one less dependency needs to be processed
		for _, name := range currentLevel {
			for _, dependent := range reverseEdges[name] {
				if !visited[dependent] {
					inDegree[dependent]--
				}
			}
		}

		// Add this level to the results
		// All modules in currentLevel can be executed in parallel
		levels = append(levels, currentLevel)
	}

	return levels, nil
}
