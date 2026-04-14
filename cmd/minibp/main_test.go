// Package main contains unit tests for the minibp build system.
// It tests glob expansion, property merging, dependency graph topological sorting,
// and file handling in the build pipeline.
package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"minibp/parser"
)

// TestExpandGlobRecursiveExtension tests recursive glob expansion (**/*.go)
// that should find .go files at any depth, excluding non-matching extensions.
func TestExpandGlobRecursiveExtension(t *testing.T) {
	baseDir := t.TempDir()
	writeTestFile(t, filepath.Join(baseDir, "root.go"))
	writeTestFile(t, filepath.Join(baseDir, "nested", "child.go"))
	writeTestFile(t, filepath.Join(baseDir, "nested", "child.txt"))

	got, err := expandGlob("**/*.go", baseDir)
	if err != nil {
		t.Fatalf("expandGlob returned error: %v", err)
	}
	sort.Strings(got)

	want := []string{"nested/child.go", "root.go"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expandGlob returned %v, want %v", got, want)
	}
}

// TestExpandGlobRecursiveUnderPrefix tests recursive glob expansion under a prefix directory.
// It should find files in subdirectories but exclude files outside the prefix path.
func TestExpandGlobRecursiveUnderPrefix(t *testing.T) {
	baseDir := t.TempDir()
	writeTestFile(t, filepath.Join(baseDir, "src", "root.go"))
	writeTestFile(t, filepath.Join(baseDir, "src", "deep", "child.go"))
	writeTestFile(t, filepath.Join(baseDir, "other", "outside.go"))

	got, err := expandGlob("src/**/*.go", baseDir)
	if err != nil {
		t.Fatalf("expandGlob returned error: %v", err)
	}
	sort.Strings(got)

	want := []string{"src/deep/child.go", "src/root.go"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expandGlob returned %v, want %v", got, want)
	}
}

// TestExpandGlobNonRecursive tests simple glob expansion without recursion.
// It should only match files in the base directory, not in subdirectories.
func TestExpandGlobNonRecursive(t *testing.T) {
	baseDir := t.TempDir()
	writeTestFile(t, filepath.Join(baseDir, "root.go"))
	writeTestFile(t, filepath.Join(baseDir, "nested", "child.go"))

	got, err := expandGlob("*.go", baseDir)
	if err != nil {
		t.Fatalf("expandGlob returned error: %v", err)
	}
	sort.Strings(got)

	want := []string{"root.go"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expandGlob returned %v, want %v", got, want)
	}
}

// TestMergeVariantPropsBeforeGlobExpansion tests that variant properties are merged
// before glob expansion occurs, so globs in variant-specific configs are properly expanded.
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

	// Merge variant properties first, then expand globs
	mergeVariantProps(mod, "arm64", false, nil)
	if err := expandGlobsInModule(mod, baseDir); err != nil {
		t.Fatalf("expandGlobsInModule returned error: %v", err)
	}

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

// TestExpandGlobNoMatchesReturnsEmpty tests that glob patterns with no matches
// return an empty slice rather than an error.
func TestExpandGlobNoMatchesReturnsEmpty(t *testing.T) {
	baseDir := t.TempDir()
	writeTestFile(t, filepath.Join(baseDir, "main.go"))

	got, err := expandGlob("missing/*.go", baseDir)
	if err != nil {
		t.Fatalf("expandGlob returned error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("Expected no matches, got %v", got)
	}
}

// TestExpandGlobsInModuleDeduplicatesSrcs tests that when multiple glob patterns
// result in the same file, it is only included once in the final srcs list.
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

	if err := expandGlobsInModule(mod, baseDir); err != nil {
		t.Fatalf("expandGlobsInModule returned error: %v", err)
	}

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

// TestExpandGlobsInModuleDropsUnmatchedPatterns tests that glob patterns
// that don't match any files are dropped from the srcs property.
func TestExpandGlobsInModuleDropsUnmatchedPatterns(t *testing.T) {
	baseDir := t.TempDir()
	writeTestFile(t, filepath.Join(baseDir, "common.go"))

	mod := &parser.Module{
		Type: "go_library",
		Map: &parser.Map{Properties: []*parser.Property{
			{Name: "srcs", Value: &parser.List{Values: []parser.Expression{
				&parser.String{Value: "missing/*.go"},
				&parser.String{Value: "common.go"},
			}}},
		}},
	}

	if err := expandGlobsInModule(mod, baseDir); err != nil {
		t.Fatalf("expandGlobsInModule returned error: %v", err)
	}

	srcsProp := findModuleProp(mod, "srcs")
	list := srcsProp.Value.(*parser.List)
	got := make([]string, 0, len(list.Values))
	for _, item := range list.Values {
		got = append(got, item.(*parser.String).Value)
	}

	want := []string{"common.go"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expanded srcs = %v, want %v", got, want)
	}
}

// TestExpandGlobInvalidPatternReturnsError tests that an invalid glob pattern
// (malformed bracket expression) returns an error.
func TestExpandGlobInvalidPatternReturnsError(t *testing.T) {
	baseDir := t.TempDir()
	if _, err := expandGlob("[", baseDir); err == nil {
		t.Fatal("Expected invalid glob pattern error")
	}
}

// TestMergeMapPropsAppendsLists tests that when merging two maps,
// list properties are concatenated rather than replaced.
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

// TestMergeMapPropsOverridesScalar tests that when merging two maps,
// scalar (non-list) properties are overridden by the variant values.
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

// TestGraphTopoSortMissingSourceNode tests that TopoSort returns an error
// when an edge references a module that doesn't exist.
func TestGraphTopoSortMissingSourceNode(t *testing.T) {
	g := NewGraph()
	g.AddNode("dep", &parser.Module{Type: "go_library"})
	g.AddEdge("missing", "dep")

	_, err := g.TopoSort()
	if err == nil {
		t.Fatal("Expected error for missing source node")
	}
	if !strings.Contains(err.Error(), "module 'missing'") {
		t.Fatalf("Unexpected error: %v", err)
	}
}

// TestGraphTopoSortSortsEachLevel tests that TopoSort returns modules
// sorted alphabetically within each build level.
func TestGraphTopoSortSortsEachLevel(t *testing.T) {
	g := NewGraph()
	g.AddNode("c", &parser.Module{Type: "go_library"})
	g.AddNode("a", &parser.Module{Type: "go_library"})
	g.AddNode("b", &parser.Module{Type: "go_library"})

	levels, err := g.TopoSort()
	if err != nil {
		t.Fatalf("TopoSort returned error: %v", err)
	}

	want := [][]string{{"a", "b", "c"}}
	if !reflect.DeepEqual(levels, want) {
		t.Fatalf("TopoSort returned %v, want %v", levels, want)
	}
}

// TestParseDefinitionsFromFilesClosesInputOnParseError tests that input files
// are properly closed even when parsing fails.
func TestParseDefinitionsFromFilesClosesInputOnParseError(t *testing.T) {
	oldOpen := openInputFile
	oldParse := parseBlueprintFile
	t.Cleanup(func() {
		openInputFile = oldOpen
		parseBlueprintFile = oldParse
	})

	tracker := &trackingReadCloser{Reader: strings.NewReader("")}
	openInputFile = func(path string) (io.ReadCloser, error) {
		return tracker, nil
	}
	parseBlueprintFile = func(r io.Reader, fileName string) (*parser.File, error) {
		return nil, errors.New("boom")
	}

	_, err := parseDefinitionsFromFiles([]string{"broken.bp"})
	if err == nil {
		t.Fatal("Expected parseDefinitionsFromFiles error")
	}
	if !tracker.closed {
		t.Fatal("Expected input file to be closed on parse error")
	}
}

// TestGenerateNinjaFileClosesOutputOnGenerateError tests that output files
// are properly closed even when Ninja generation fails.
func TestGenerateNinjaFileClosesOutputOnGenerateError(t *testing.T) {
	oldCreate := createOutputFile
	t.Cleanup(func() {
		createOutputFile = oldCreate
	})

	tracker := &trackingWriteCloser{}
	createOutputFile = func(path string) (io.WriteCloser, error) {
		return tracker, nil
	}

	err := generateNinjaFile("build.ninja", generatorFunc(func(w io.Writer) error {
		return errors.New("boom")
	}))
	if err == nil {
		t.Fatal("Expected generateNinjaFile error")
	}
	if !tracker.closed {
		t.Fatal("Expected output file to be closed on generate error")
	}
}

// trackingReadCloser is a test helper that wraps an io.Reader and tracks whether Close was called.
type trackingReadCloser struct {
	io.Reader
	closed bool
}

// Close marks the tracker as closed.
func (t *trackingReadCloser) Close() error {
	t.closed = true
	return nil
}

// trackingWriteCloser is a test helper that tracks whether Close was called.
type trackingWriteCloser struct {
	closed bool
}

// Write implements io.Writer by discarding data and reporting length.
func (t *trackingWriteCloser) Write(p []byte) (int, error) {
	return len(p), nil
}

// Close marks the tracker as closed.
func (t *trackingWriteCloser) Close() error {
	t.closed = true
	return nil
}

// generatorFunc is an adapter that allows a function to satisfy the Generate method.
type generatorFunc func(io.Writer) error

// Generate calls the underlying function.
func (f generatorFunc) Generate(w io.Writer) error {
	return f(w)
}

// writeTestFile creates a test file with dummy content in the given directory structure.
// It creates parent directories as needed.
func writeTestFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
	if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

// findModuleProp finds a property by name in a module's property map.
// Returns nil if the module or its map is nil.
func findModuleProp(m *parser.Module, name string) *parser.Property {
	if m == nil || m.Map == nil {
		return nil
	}
	return findMapProp(m.Map, name)
}

// findMapProp finds a property by name in a map's properties.
// Returns nil if the map is nil.
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
