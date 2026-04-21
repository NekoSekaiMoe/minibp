package dependency

import (
	"strings"
	"testing"
)

func TestNewDependencyGraph(t *testing.T) {
	g := NewDependencyGraph()
	if g == nil {
		t.Error("Expected graph to be created")
	}
	if g.modules == nil {
		t.Error("Expected modules map to be initialized")
	}
}

func TestAddModule(t *testing.T) {
	g := NewDependencyGraph()
	deps := []Dependency{
		{Name: "libB", Version: "1.0"},
		{Name: "libC", Version: "2.0"},
	}

	g.AddModule("libA", "cc_library", deps)

	node, exists := g.modules["libA"]
	if !exists {
		t.Fatal("Expected module to be added")
	}
	if node.Type != "cc_library" {
		t.Errorf("Expected type cc_library, got %s", node.Type)
	}
	if len(node.DirectDeps) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(node.DirectDeps))
	}
}

func TestTransitiveDependencies(t *testing.T) {
	g := NewDependencyGraph()

	// A -> B -> C
	// A -> C
	g.AddModule("libC", "cc_library", []Dependency{})
	g.AddModule("libB", "cc_library", []Dependency{
		{Name: "libC", Version: "1.0"},
	})
	g.AddModule("libA", "cc_library", []Dependency{
		{Name: "libB", Version: "1.0"},
		{Name: "libC", Version: "1.0"},
	})

	g.calculateTransitiveDeps("libA")

	nodeA := g.modules["libA"]
	if len(nodeA.AllDeps) < 2 {
		t.Errorf("Expected at least 2 transitive deps, got %d", len(nodeA.AllDeps))
	}
}

func TestDetectConflicts(t *testing.T) {
	g := NewDependencyGraph()

	// A requires B version 1.0
	// C requires B version 2.0
	g.AddModule("libB", "cc_library", []Dependency{})
	g.AddModule("libA", "cc_library", []Dependency{
		{Name: "libB", Version: "1.0"},
	})
	g.AddModule("libC", "cc_library", []Dependency{
		{Name: "libB", Version: "2.0"},
	})

	conflicts := g.detectConflicts()

	if len(conflicts) != 1 {
		t.Errorf("Expected 1 conflict, got %d", len(conflicts))
	} else {
		if conflicts[0].DepName != "libB" {
			t.Errorf("Expected conflict on libB, got %s", conflicts[0].DepName)
		}
	}
}

func TestTopologicalSort(t *testing.T) {
	g := NewDependencyGraph()

	// D has no deps
	// B and C depend on D
	// A depends on B and C
	g.AddModule("D", "cc_library", []Dependency{})
	g.AddModule("C", "cc_library", []Dependency{
		{Name: "D", Version: "1.0"},
	})
	g.AddModule("B", "cc_library", []Dependency{
		{Name: "D", Version: "1.0"},
	})
	g.AddModule("A", "cc_library", []Dependency{
		{Name: "B", Version: "1.0"},
		{Name: "C", Version: "1.0"},
	})

	order, err := g.topologicalSort()
	if err != nil {
		t.Fatalf("Topological sort failed: %v", err)
	}

	// D should come before B and C
	// B and C should come before A
	dPos := -1
	aPos := -1
	for i, mod := range order {
		if mod == "D" {
			dPos = i
		}
		if mod == "A" {
			aPos = i
		}
	}

	if dPos == -1 || aPos == -1 {
		t.Error("Expected all modules in order")
	}
	if dPos >= aPos {
		t.Error("Expected D to come before A")
	}
}

func TestCircularDependency(t *testing.T) {
	g := NewDependencyGraph()

	// A -> B -> C -> A (cycle)
	g.AddModule("A", "cc_library", []Dependency{
		{Name: "B", Version: "1.0"},
	})
	g.AddModule("B", "cc_library", []Dependency{
		{Name: "C", Version: "1.0"},
	})
	g.AddModule("C", "cc_library", []Dependency{
		{Name: "A", Version: "1.0"},
	})

	_, err := g.topologicalSort()
	if err == nil {
		t.Error("Expected error for circular dependency")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("Expected circular dependency error, got: %v", err)
	}
}

func TestResolveDependencies(t *testing.T) {
	g := NewDependencyGraph()

	g.AddModule("libC", "cc_library", []Dependency{})
	g.AddModule("libB", "cc_library", []Dependency{
		{Name: "libC", Version: "1.0"},
	})
	g.AddModule("libA", "cc_library", []Dependency{
		{Name: "libB", Version: "1.0"},
		{Name: "libC", Version: "1.0"},
	})

	resolution := g.ResolveDependencies()

	if !resolution.Success {
		t.Error("Expected successful resolution")
	}
	if len(resolution.Order) != 3 {
		t.Errorf("Expected 3 modules in order, got %d", len(resolution.Order))
	}
}

func TestVisualize(t *testing.T) {
	g := NewDependencyGraph()
	g.AddModule("libA", "cc_library", []Dependency{
		{Name: "libB", Version: "1.0"},
	})

	visualization := g.Visualize()

	if !strings.Contains(visualization, "Dependency Graph") {
		t.Error("Expected visualization header")
	}
	if !strings.Contains(visualization, "libA") {
		t.Error("Expected libA in visualization")
	}
	if !strings.Contains(visualization, "libB") {
		t.Error("Expected libB in visualization")
	}
}

func TestGetDependents(t *testing.T) {
	g := NewDependencyGraph()

	// B and C depend on A
	g.AddModule("A", "cc_library", []Dependency{})
	g.AddModule("B", "cc_library", []Dependency{
		{Name: "A", Version: "1.0"},
	})
	g.AddModule("C", "cc_library", []Dependency{
		{Name: "A", Version: "1.0"},
	})

	dependents := g.GetDependents("A")
	if len(dependents) != 2 {
		t.Errorf("Expected 2 dependents, got %d", len(dependents))
	}
}

func TestGetDependencies(t *testing.T) {
	g := NewDependencyGraph()
	g.AddModule("A", "cc_library", []Dependency{
		{Name: "B", Version: "1.0"},
		{Name: "C", Version: "1.0"},
	})

	deps := g.GetDependencies("A")
	if len(deps) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(deps))
	}
}

func TestGetModule(t *testing.T) {
	g := NewDependencyGraph()
	g.AddModule("libA", "cc_library", []Dependency{})

	node, exists := g.GetModule("libA")
	if !exists {
		t.Fatal("Expected module to exist")
	}
	if node.Name != "libA" {
		t.Errorf("Expected name libA, got %s", node.Name)
	}

	_, exists = g.GetModule("nonexistent")
	if exists {
		t.Error("Expected nonexistent module to not exist")
	}
}

func TestGetAllModules(t *testing.T) {
	g := NewDependencyGraph()
	g.AddModule("A", "cc_library", []Dependency{})
	g.AddModule("B", "cc_library", []Dependency{})
	g.AddModule("C", "cc_library", []Dependency{})

	modules := g.GetAllModules()
	if len(modules) != 3 {
		t.Errorf("Expected 3 modules, got %d", len(modules))
	}
}
