package errors

import (
	"strings"
	"testing"
)

func TestErrorCategoryString(t *testing.T) {
	tests := []struct {
		category ErrorCategory
		expected string
	}{
		{SyntaxError, "SyntaxError"},
		{DependencyError, "DependencyError"},
		{CircularDependency, "CircularDependency"},
		{FileNotFoundError, "FileNotFoundError"},
		{DuplicateDefinition, "DuplicateDefinition"},
		{TypeMismatch, "TypeMismatch"},
		{MissingProperty, "MissingProperty"},
		{InvalidValue, "InvalidValue"},
		{Uncategorized, "Uncategorized"},
	}
	for _, test := range tests {
		result := test.category.String()
		if result != test.expected {
			t.Errorf("Category %v returned %s, expected %s", test.category, result, test.expected)
		}
	}
}

func TestNewError(t *testing.T) {
	err := NewError(SyntaxError, "test error")
	if err.Category != SyntaxError {
		t.Errorf("Expected category SyntaxError, got %v", err.Category)
	}
	if err.Message != "test error" {
		t.Errorf("Expected message 'test error', got %s", err.Message)
	}
}

func TestWithLocation(t *testing.T) {
	err := NewError(SyntaxError, "test").
		WithLocation("test.bp", 10, 5)
	if err.Location.File != "test.bp" {
		t.Errorf("Expected file 'test.bp', got %s", err.Location.File)
	}
	if err.Location.Line != 10 {
		t.Errorf("Expected line 10, got %d", err.Location.Line)
	}
	if err.Location.Column != 5 {
		t.Errorf("Expected column 5, got %d", err.Location.Column)
	}
}

func TestWithContent(t *testing.T) {
	content := "cc_library { name: \"test\" }"
	err := NewError(SyntaxError, "test").
		WithLocation("test.bp", 1, 0).
		WithContent(content)
	if err.Location.Content != content {
		t.Errorf("Expected content '%s', got %s", content, err.Location.Content)
	}
}

func TestWithSuggestion(t *testing.T) {
	suggestion := "Check your syntax"
	err := NewError(SyntaxError, "test").WithSuggestion(suggestion)
	if err.Suggestion != suggestion {
		t.Errorf("Expected suggestion '%s', got %s", suggestion, err.Suggestion)
	}
}

func TestCircularDependencyError(t *testing.T) {
	chain := []string{"libA", "libB", "libC", "libA"}
	err := Circular(chain)
	if err.Category != CircularDependency {
		t.Errorf("Expected category CircularDependency, got %v", err.Category)
	}
	if err.Context == nil {
		t.Fatal("Expected context to be set")
	}
	if len(err.Context.DependencyChain) != 4 {
		t.Errorf("Expected 4 items in chain, got %d", len(err.Context.DependencyChain))
	}
}

func TestNotFoundError(t *testing.T) {
	err := NotFound("/path/to/file.bp")
	if err.Category != FileNotFoundError {
		t.Errorf("Expected category FileNotFoundError, got %v", err.Category)
	}
	if !strings.Contains(err.Message, "/path/to/file.bp") {
		t.Errorf("Expected message to contain path, got %s", err.Message)
	}
}

func TestDuplicateError(t *testing.T) {
	err := Duplicate("mylib", "src.bp", 15)
	if err.Category != DuplicateDefinition {
		t.Errorf("Expected category DuplicateDefinition, got %v", err.Category)
	}
	if !strings.Contains(err.Message, "mylib") {
		t.Errorf("Expected message to contain name, got %s", err.Message)
	}
}

func TestFormat(t *testing.T) {
	err := NewError(SyntaxError, "unexpected token").
		WithLocation("test.bp", 5, 10).
		WithContent("cc_library { name: \"test\" }").
		WithSuggestion("Check your syntax")

	formatted := err.Format()
	if !strings.Contains(formatted, "SyntaxError") {
		t.Error("Expected format to contain category")
	}
	if !strings.Contains(formatted, "test.bp:5") {
		t.Error("Expected format to contain location")
	}
	if !strings.Contains(formatted, "Suggestion:") {
		t.Error("Expected format to contain suggestion")
	}
}

func TestErrorImplementsErrorInterface(t *testing.T) {
	err := NewError(SyntaxError, "test")
	var _ error = err // Compile-time check
}

func TestHelperFunctions(t *testing.T) {
	tests := []struct {
		name     string
		err      *BuildError
		category ErrorCategory
	}{
		{"Syntax", Syntax("test"), SyntaxError},
		{"Dependency", Dependency("test"), DependencyError},
		{"Config", Config("test"), ConfigurationError},
		{"Type", Type("mod", "prop", "int", "string"), TypeMismatch},
		{"Missing", Missing("mod", "prop"), MissingProperty},
		{"Invalid", Invalid("mod", "prop", "val", "reason"), InvalidValue},
	}
	for _, test := range tests {
		if test.err.Category != test.category {
			t.Errorf("%s: expected category %v, got %v", test.name, test.category, test.err.Category)
		}
	}
}

func TestErrorContext(t *testing.T) {
	ctx := &ErrorContext{
		Snippet:      "code snippet",
		RelatedFiles: []string{"file1.bp", "file2.bp"},
		DependencyChain: []string{"A", "B", "C"},
	}
	err := NewError(DependencyError, "test").WithContext(ctx)
	if err.Context != ctx {
		t.Error("Expected context to be set")
	}
}