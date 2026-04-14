package main

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"minibp/parser"
)

func TestExpandGlobRecursiveExtension(t *testing.T) {
	baseDir := t.TempDir()
	writeTestFile(t, filepath.Join(baseDir, "root.go"))
	writeTestFile(t, filepath.Join(baseDir, "nested", "child.go"))
	writeTestFile(t, filepath.Join(baseDir, "nested", "child.txt"))

	got := expandGlob("**/*.go", baseDir)
	sort.Strings(got)

	want := []string{"nested/child.go", "root.go"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expandGlob returned %v, want %v", got, want)
	}
}

func TestExpandGlobRecursiveUnderPrefix(t *testing.T) {
	baseDir := t.TempDir()
	writeTestFile(t, filepath.Join(baseDir, "src", "root.go"))
	writeTestFile(t, filepath.Join(baseDir, "src", "deep", "child.go"))
	writeTestFile(t, filepath.Join(baseDir, "other", "outside.go"))

	got := expandGlob("src/**/*.go", baseDir)
	sort.Strings(got)

	want := []string{"src/deep/child.go", "src/root.go"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expandGlob returned %v, want %v", got, want)
	}
}

func TestExpandGlobNonRecursive(t *testing.T) {
	baseDir := t.TempDir()
	writeTestFile(t, filepath.Join(baseDir, "root.go"))
	writeTestFile(t, filepath.Join(baseDir, "nested", "child.go"))

	got := expandGlob("*.go", baseDir)
	sort.Strings(got)

	want := []string{"root.go"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expandGlob returned %v, want %v", got, want)
	}
}

func TestMergeVariantPropsBeforeGlobExpansion(t *testing.T) {
	baseDir := t.TempDir()
	writeTestFile(t, filepath.Join(baseDir, "base.go"))
	writeTestFile(t, filepath.Join(baseDir, "arch", "arm64", "extra.go"))

	mod := &parser.Module{
		Type: "go_library",
		Map: &parser.Map{Properties: []*parser.Property{
			{Name: "srcs", Value: &parser.List{Values: []parser.Expression{
				&parser.String{Value: "base.go"},
			}}},
		}},
		Arch: map[string]*parser.Map{
			"arm64": {
				Properties: []*parser.Property{
					{Name: "srcs", Value: &parser.List{Values: []parser.Expression{
						&parser.String{Value: "arch/arm64/*.go"},
					}}},
				},
			},
		},
	}

	mergeVariantProps(mod, "arm64", false, nil)
	expandGlobsInModule(mod, baseDir)

	srcsProp := findModuleProp(mod, "srcs")
	if srcsProp == nil {
		t.Fatal("Missing srcs property")
	}
	list, ok := srcsProp.Value.(*parser.List)
	if !ok {
		t.Fatalf("Expected *parser.List, got %T", srcsProp.Value)
	}

	got := make([]string, 0, len(list.Values))
	for _, item := range list.Values {
		str, ok := item.(*parser.String)
		if !ok {
			t.Fatalf("Expected *parser.String, got %T", item)
		}
		got = append(got, str.Value)
	}

	want := []string{"base.go", "arch/arm64/extra.go"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("merged+expanded srcs = %v, want %v", got, want)
	}
}

func TestExpandGlobNoMatchesReturnsEmpty(t *testing.T) {
	baseDir := t.TempDir()
	writeTestFile(t, filepath.Join(baseDir, "main.go"))

	got := expandGlob("missing/*.go", baseDir)
	if len(got) != 0 {
		t.Fatalf("Expected no matches, got %v", got)
	}
}

func TestExpandGlobsInModuleDeduplicatesSrcs(t *testing.T) {
	baseDir := t.TempDir()
	writeTestFile(t, filepath.Join(baseDir, "common.go"))
	writeTestFile(t, filepath.Join(baseDir, "nested", "extra.go"))

	mod := &parser.Module{
		Type: "go_library",
		Map: &parser.Map{Properties: []*parser.Property{
			{Name: "srcs", Value: &parser.List{Values: []parser.Expression{
				&parser.String{Value: "common.go"},
				&parser.String{Value: "**/*.go"},
				&parser.String{Value: "nested/*.go"},
			}}},
		}},
	}

	expandGlobsInModule(mod, baseDir)

	srcsProp := findModuleProp(mod, "srcs")
	list := srcsProp.Value.(*parser.List)
	got := make([]string, 0, len(list.Values))
	for _, item := range list.Values {
		got = append(got, item.(*parser.String).Value)
	}

	want := []string{"common.go", "nested/extra.go"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expanded srcs = %v, want %v", got, want)
	}
}

func TestMergeMapPropsAppendsLists(t *testing.T) {
	base := &parser.Module{Map: &parser.Map{Properties: []*parser.Property{
		{Name: "srcs", Value: &parser.List{Values: []parser.Expression{
			&parser.String{Value: "base.go"},
		}}},
	}}}
	variant := &parser.Map{Properties: []*parser.Property{
		{Name: "srcs", Value: &parser.List{Values: []parser.Expression{
			&parser.String{Value: "variant.go"},
		}}},
	}}

	mergeMapProps(base, variant)

	srcsProp := findModuleProp(base, "srcs")
	list := srcsProp.Value.(*parser.List)
	got := []string{list.Values[0].(*parser.String).Value, list.Values[1].(*parser.String).Value}
	want := []string{"base.go", "variant.go"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("merged list = %v, want %v", got, want)
	}
}

func TestMergeMapPropsOverridesScalar(t *testing.T) {
	base := &parser.Module{Map: &parser.Map{Properties: []*parser.Property{{
		Name:  "enabled",
		Value: &parser.Bool{Value: false},
	}}}}
	variant := &parser.Map{Properties: []*parser.Property{{
		Name:  "enabled",
		Value: &parser.Bool{Value: true},
	}}}

	mergeMapProps(base, variant)

	prop := findModuleProp(base, "enabled")
	if prop == nil {
		t.Fatal("Missing enabled property")
	}
	val, ok := prop.Value.(*parser.Bool)
	if !ok {
		t.Fatalf("Expected *parser.Bool, got %T", prop.Value)
	}
	if !val.Value {
		t.Fatal("Expected scalar property to be overridden")
	}
}

func writeTestFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
	if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

func findModuleProp(m *parser.Module, name string) *parser.Property {
	if m == nil || m.Map == nil {
		return nil
	}
	return findMapProp(m.Map, name)
}

func findMapProp(m *parser.Map, name string) *parser.Property {
	if m == nil {
		return nil
	}
	for _, prop := range m.Properties {
		if prop.Name == name {
			return prop
		}
	}
	return nil
}
