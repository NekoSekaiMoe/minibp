package glob

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"minibp/lib/parser"
)

func TestExpandInModuleNoGlobs(t *testing.T) {
	m := &parser.Module{
		Type: "cc_binary",
		Map: &parser.Map{Properties: []*parser.Property{
			{Name: "name", Value: &parser.String{Value: "app"}},
			{Name: "srcs", Value: &parser.List{Values: []parser.Expression{
				&parser.String{Value: "main.c"},
				&parser.String{Value: "util.c"},
			}}},
		}},
	}
	err := ExpandInModule(m, ".")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list := m.Map.Properties[1].Value.(*parser.List)
	if len(list.Values) != 2 {
		t.Fatalf("expected 2 srcs, got %d", len(list.Values))
	}
}

func TestExpandInModuleNilMap(t *testing.T) {
	m := &parser.Module{Type: "phony"}
	if err := ExpandInModule(m, "."); err != nil {
		t.Fatalf("unexpected error for nil map: %v", err)
	}
}

func TestExpandInModuleDeduplicates(t *testing.T) {
	m := &parser.Module{
		Type: "cc_binary",
		Map: &parser.Map{Properties: []*parser.Property{
			{Name: "srcs", Value: &parser.List{Values: []parser.Expression{
				&parser.String{Value: "main.c"},
				&parser.String{Value: "main.c"},
			}}},
		}},
	}
	err := ExpandInModule(m, ".")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var srcsList *parser.List
	for _, prop := range m.Map.Properties {
		if prop.Name == "srcs" {
			srcsList = prop.Value.(*parser.List)
			break
		}
	}
	if srcsList == nil {
		t.Fatal("srcs property not found")
	}
	if len(srcsList.Values) != 1 {
		t.Fatalf("expected 1 deduplicated src, got %d", len(srcsList.Values))
	}
}

func TestRecursiveGlobRoot(t *testing.T) {
	tests := []struct {
		pattern string
		baseDir string
		want    string
	}{
		{"**/*.c", ".", "."},
		{"src/**/*.c", ".", filepath.Join(".", "src")},
		{"lib/parser/**/*.go", "/project", filepath.Join("/project", "lib", "parser")},
		{"*.c", ".", "."},
	}
	for _, tc := range tests {
		got := recursiveGlobRoot(tc.pattern, tc.baseDir)
		if got != tc.want {
			t.Errorf("recursiveGlobRoot(%q, %q) = %q, want %q", tc.pattern, tc.baseDir, got, tc.want)
		}
	}
}

func TestMatchRecursivePattern(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"**/*.c", "foo.c", true},
		{"**/*.c", "dir/foo.c", true},
		{"**/*.c", "a/b/c/foo.c", true},
		{"**/*.c", "foo.h", false},
		{"src/**/*.go", "src/main.go", true},
		{"src/**/*.go", "src/lib/util.go", true},
		{"src/**/*.go", "lib/util.go", false},
		{"**/test_*.go", "test_main.go", true},
		{"**/test_*.go", "pkg/test_helper.go", true},
		{"**/test_*.go", "main.go", false},
	}
	for _, tc := range tests {
		got := matchRecursivePattern(tc.pattern, tc.path)
		if got != tc.want {
			t.Errorf("matchRecursivePattern(%q, %q) = %v, want %v", tc.pattern, tc.path, got, tc.want)
		}
	}
}

func TestExpandGlobSimple(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("pkg p"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("text"), 0644); err != nil {
		t.Fatal(err)
	}

	matches, err := expandGlob("*.go", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matches) != 1 || matches[0] != "a.go" {
		t.Errorf("expected [a.go], got %v", matches)
	}
}

func TestExpandGlobRecursive(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "pkg")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("pkg main"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "util.go"), []byte("pkg util"), 0644); err != nil {
		t.Fatal(err)
	}

	matches, err := expandGlob("**/*.go", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sort.Strings(matches)
	want := []string{"main.go", "pkg/util.go"}
	if !reflect.DeepEqual(matches, want) {
		t.Errorf("expected %v, got %v", want, matches)
	}
}

func TestExpandGlobNoMatches(t *testing.T) {
	dir := t.TempDir()
	matches, err := expandGlob("*.xyz", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matches) != 0 {
		t.Errorf("expected no matches, got %v", matches)
	}
}

func TestSplitGlobParts(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"", nil},
		{"a", []string{"a"}},
		{"a/b/c", []string{"a", "b", "c"}},
	}
	for _, tc := range tests {
		got := splitGlobParts(tc.input)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("splitGlobParts(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}
