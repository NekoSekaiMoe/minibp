package build

import (
	"bytes"
	"strings"
	"testing"

	"minibp/lib/namespace"
	"minibp/lib/parser"
)

func TestCollectModulesWithNames(t *testing.T) {
	eval := parser.NewEvaluator()
	opts := Options{Arch: "arm64", SrcDir: "."}
	defs := []parser.Definition{
		&parser.Module{
			Type: "cc_binary",
			Map: &parser.Map{Properties: []*parser.Property{
				{Name: "name", Value: &parser.String{Value: "app"}},
				{Name: "srcs", Value: &parser.List{Values: []parser.Expression{
					&parser.String{Value: "main.c"},
				}}},
			}},
			Arch: map[string]*parser.Map{
				"arm64": {
					Properties: []*parser.Property{
						{Name: "cflags", Value: &parser.List{Values: []parser.Expression{
							&parser.String{Value: "-DARM64"},
						}}},
					},
				},
			},
		},
	}

	modules, err := CollectModulesWithNames(defs, eval, opts, func(m *parser.Module, key string) string {
		if key != "name" {
			return ""
		}
		for _, prop := range m.Map.Properties {
			if prop.Name == "name" {
				if s, ok := prop.Value.(*parser.String); ok {
					return s.Value
				}
			}
		}
		return ""
	})
	if err != nil {
		t.Fatalf("CollectModulesWithNames failed: %v", err)
	}

	mod := modules["app"]
	if mod == nil {
		t.Fatal("Expected module app to be collected")
	}
	var found bool
	for _, prop := range mod.Map.Properties {
		if prop.Name == "cflags" {
			if list, ok := prop.Value.(*parser.List); ok && len(list.Values) == 1 {
				if s, ok := list.Values[0].(*parser.String); ok && s.Value == "-DARM64" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Fatalf("Expected merged arch cflags on collected module, got %#v", mod.Map.Properties)
	}
}

func TestBuildGraphIncludesModuleReferenceDataDeps(t *testing.T) {
	eval := parser.NewEvaluator()
	modules := map[string]*parser.Module{
		"payload": {
			Type: "filegroup",
			Map: &parser.Map{Properties: []*parser.Property{
				{Name: "name", Value: &parser.String{Value: "payload"}},
			}},
		},
		"runner": {
			Type: "python_test_host",
			Map: &parser.Map{Properties: []*parser.Property{
				{Name: "name", Value: &parser.String{Value: "runner"}},
				{Name: "data", Value: &parser.List{Values: []parser.Expression{
					&parser.String{Value: ":payload"},
					&parser.String{Value: "plain.txt"},
				}}},
			}},
		},
		"plain.txt": {
			Type: "filegroup",
			Map: &parser.Map{Properties: []*parser.Property{
				{Name: "name", Value: &parser.String{Value: "plain.txt"}},
				{Name: "srcs", Value: &parser.List{Values: []parser.Expression{
					&parser.String{Value: "plain.txt"},
				}}},
			}},
		},
	}
	namespaces := map[string]*namespace.Info{}

	graph := BuildGraph(modules, namespaces, eval)
	levels, err := graph.TopoSort()
	if err != nil {
		t.Fatalf("TopoSort failed: %v", err)
	}
	if len(levels) != 2 {
		t.Fatalf("Expected 2 levels in graph, got %d", len(levels))
	}
	payloadFound := false
	for _, mod := range levels[0] {
		if mod == "payload" {
			payloadFound = true
		}
	}
	if !payloadFound {
		t.Fatalf("Expected 'payload' in the first level, got %v", levels[0])
	}
	runnerFound := false
	for _, mod := range levels[1] {
		if mod == "runner" {
			runnerFound = true
		}
	}
	if !runnerFound {
		t.Fatalf("Expected 'runner' in the second level, got %v", levels[1])
	}
}

func TestNewGeneratorGeneratesBuild(t *testing.T) {
	graph := NewGraph()
	modules := map[string]*parser.Module{
		"smoke": {
			Type: "python_test_host",
			Map: &parser.Map{Properties: []*parser.Property{
				{Name: "name", Value: &parser.String{Value: "smoke"}},
				{Name: "srcs", Value: &parser.List{Values: []parser.Expression{
					&parser.String{Value: "smoke_test.py"},
				}}},
			}},
		},
	}
	graph.AddNode("smoke", modules["smoke"])

	opts := Options{
		SrcDir:  ".",
		OutFile: "build.ninja",
		Inputs:  []string{"Android.bp"},
	}
	gen := NewGenerator(graph, modules, opts)

	var buf bytes.Buffer
	if err := gen.Generate(&buf); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "build smoke.test.py: python_test smoke_test.py") {
		t.Fatalf("Expected python test build edge, got: %s", out)
	}
	if !strings.Contains(out, "build smoke: phony smoke.test.py") {
		t.Fatalf("Expected phony module target, got: %s", out)
	}
}

func TestGraphTopoSortFailsOnMissingDependency(t *testing.T) {
	graph := NewGraph()
	graph.AddNode("runner", &parser.Module{
		Type: "python_test_host",
		Map: &parser.Map{Properties: []*parser.Property{
			{Name: "name", Value: &parser.String{Value: "runner"}},
		}},
	})
	graph.AddEdge("runner", "missing")

	_, err := graph.TopoSort()
	if err == nil || !strings.Contains(err.Error(), "referenced in dependency graph does not exist") {
		t.Fatalf("Expected missing dependency error, got %v", err)
	}
}

func TestGraphTopoSortFailsOnCycle(t *testing.T) {
	graph := NewGraph()
	graph.AddNode("a", &parser.Module{Type: "phony", Map: &parser.Map{Properties: []*parser.Property{
		{Name: "name", Value: &parser.String{Value: "a"}},
	}}})
	graph.AddNode("b", &parser.Module{Type: "phony", Map: &parser.Map{Properties: []*parser.Property{
		{Name: "name", Value: &parser.String{Value: "b"}},
	}}})
	graph.AddEdge("a", "b")
	graph.AddEdge("b", "a")

	_, err := graph.TopoSort()
	if err == nil || !strings.Contains(err.Error(), "circular dependency") {
		t.Fatalf("Expected circular dependency error, got %v", err)
	}
}

func TestNewGeneratorAddsRegenAndPathPrefix(t *testing.T) {
	graph := NewGraph()
	modules := map[string]*parser.Module{
		"tool": {
			Type: "python_binary_host",
			Map: &parser.Map{Properties: []*parser.Property{
				{Name: "name", Value: &parser.String{Value: "tool"}},
				{Name: "srcs", Value: &parser.List{Values: []parser.Expression{
					&parser.String{Value: "tools/main.py"},
				}}},
			}},
		},
	}
	graph.AddNode("tool", modules["tool"])

	opts := Options{
		SrcDir:  "examples",
		OutFile: "out/build.ninja",
		Inputs:  []string{"examples/Android.bp"},
	}
	gen := NewGenerator(graph, modules, opts)

	var buf bytes.Buffer
	if err := gen.Generate(&buf); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "rule regen") || !strings.Contains(out, "build out/build.ninja: regen") {
		t.Fatalf("Expected regen rule and build edge, got: %s", out)
	}
	if !strings.Contains(out, "build tool.py: python_copy ../examples/tools/main.py") {
		t.Fatalf("Expected prefixed input path, got: %s", out)
	}
}
