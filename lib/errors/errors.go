// Package errors provides enhanced error handling for minibp with categorized errors,
// helpful suggestions, and context information.
package errors

import (
	"fmt"
	"strings"
)

// ErrorCategory represents the type of error
type ErrorCategory int

const (
	Uncategorized       ErrorCategory = iota
	SyntaxError                       // Syntax error
	DependencyError                   // Dependency error
	ConfigurationError                // Configuration error
	FileNotFoundError                 // File not found error
	CircularDependency                // Circular dependency error
	DuplicateDefinition               // Duplicate definition error
	TypeMismatch                      // Type mismatch error
	MissingProperty                   // Missing property error
	InvalidValue                      // Invalid value error
)

func (c ErrorCategory) String() string {
	switch c {
	case SyntaxError:
		return "SyntaxError"
	case DependencyError:
		return "DependencyError"
	case ConfigurationError:
		return "ConfigurationError"
	case FileNotFoundError:
		return "FileNotFoundError"
	case CircularDependency:
		return "CircularDependency"
	case DuplicateDefinition:
		return "DuplicateDefinition"
	case TypeMismatch:
		return "TypeMismatch"
	case MissingProperty:
		return "MissingProperty"
	case InvalidValue:
		return "InvalidValue"
	default:
		return "Uncategorized"
	}
}

// ErrorSeverity represents the severity level of an error
type ErrorSeverity int

const (
	Error ErrorSeverity = iota
	Warning
	Info
)

func (s ErrorSeverity) String() string {
	switch s {
	case Error:
		return "Error"
	case Warning:
		return "Warning"
	case Info:
		return "Info"
	default:
		return "Unknown"
	}
}

// Location represents a position in source code
type Location struct {
	File    string // File name
	Line    int    // Line number
	Column  int    // Column number
	Content string // Content of the current line
}

// ErrorContext provides additional context for an error
type ErrorContext struct {
	Snippet         string   // Code snippet (surrounding lines)
	RelatedFiles    []string // Related files
	DependencyChain []string // Dependency chain (for circular dependencies)
}

// BuildError represents a structured build error
type BuildError struct {
	Category   ErrorCategory
	Severity   ErrorSeverity
	Message    string
	Location   *Location
	Context    *ErrorContext
	Suggestion string // Suggestion for fixing
	Cause      error  // Underlying cause
}

// NewError creates a new BuildError
func NewError(category ErrorCategory, message string) *BuildError {
	return &BuildError{
		Category: category,
		Severity: Error,
		Message:  message,
	}
}

// WithLocation sets the location for the error
func (e *BuildError) WithLocation(file string, line, column int) *BuildError {
	e.Location = &Location{
		File:   file,
		Line:   line,
		Column: column,
	}
	return e
}

// WithContent sets the content at the error location
func (e *BuildError) WithContent(content string) *BuildError {
	if e.Location == nil {
		e.Location = &Location{}
	}
	e.Location.Content = content
	return e
}

// WithContext sets additional context
func (e *BuildError) WithContext(ctx *ErrorContext) *BuildError {
	e.Context = ctx
	return e
}

// WithSuggestion sets a suggestion for fixing the error
func (e *BuildError) WithSuggestion(suggestion string) *BuildError {
	e.Suggestion = suggestion
	return e
}

// WithCause sets the underlying cause
func (e *BuildError) WithCause(cause error) *BuildError {
	e.Cause = cause
	return e
}

// Format formats the error for display
func (e *BuildError) Format() string {
	var sb strings.Builder

	// Error type and message
	sb.WriteString(fmt.Sprintf("[%s] %s: %s\n", e.Category, e.Severity, e.Message))

	// Location information
	if e.Location != nil && e.Location.File != "" {
		loc := fmt.Sprintf("%s:%d", e.Location.File, e.Location.Line)
		if e.Location.Column > 0 {
			loc += fmt.Sprintf(":%d", e.Location.Column)
		}
		sb.WriteString(fmt.Sprintf(" --> %s\n", loc))

		// Code content and pointer
		if e.Location.Content != "" {
			sb.WriteString(" |\n")
			sb.WriteString(fmt.Sprintf("%d | %s\n", e.Location.Line, e.Location.Content))
			sb.WriteString(" | ")
			sb.WriteString(strings.Repeat(" ", e.Location.Column-1))
			sb.WriteString(strings.Repeat("^", 5))
			sb.WriteString("\n")
		}
	}

	// Context code snippet
	if e.Context != nil && e.Context.Snippet != "" {
		sb.WriteString(" |\n")
		sb.WriteString(e.Context.Snippet)
		sb.WriteString(" |\n")
	}

	// Dependency chain (for circular dependencies)
	if e.Context != nil && len(e.Context.DependencyChain) > 0 {
		sb.WriteString("Dependency chain:\n")
		for i, dep := range e.Context.DependencyChain {
			indent := strings.Repeat(" ", i)
			sb.WriteString(fmt.Sprintf("%s -> %s\n", indent, dep))
		}
	}

	// Suggestion
	if e.Suggestion != "" {
		sb.WriteString(fmt.Sprintf("\nSuggestion: %s\n", e.Suggestion))
	}

	// Underlying cause
	if e.Cause != nil {
		sb.WriteString(fmt.Sprintf("\nCause: %v\n", e.Cause))
	}

	return sb.String()
}

// Error implements the error interface
func (e *BuildError) Error() string {
	return e.Format()
}

// Helper functions for creating specific error types

// Syntax creates a syntax error
func Syntax(msg string) *BuildError {
	return NewError(SyntaxError, msg)
}

// Dependency creates a dependency error
func Dependency(msg string) *BuildError {
	return NewError(DependencyError, msg)
}

// Circular creates a circular dependency error
func Circular(chain []string) *BuildError {
	return NewError(CircularDependency, "Circular dependency detected").
		WithContext(&ErrorContext{DependencyChain: chain}).
		WithSuggestion("Check the dependency chain and remove one dependency to break the cycle")
}

// NotFound creates a file not found error
func NotFound(file string) *BuildError {
	return NewError(FileNotFoundError, fmt.Sprintf("File not found: %s", file)).
		WithSuggestion("Check if the file path is correct or if the file exists")
}

// Duplicate creates a duplicate definition error
func Duplicate(name, file string, line int) *BuildError {
	return NewError(DuplicateDefinition, fmt.Sprintf("Duplicate definition: %s", name)).
		WithSuggestion(fmt.Sprintf("Look for previous definition near %s:%d", file, line))
}

// Missing creates a missing property error
func Missing(moduleName, propertyName string) *BuildError {
	return NewError(MissingProperty, fmt.Sprintf("Module %s is missing required property: %s", moduleName, propertyName))
}

// Invalid creates an invalid value error
func Invalid(moduleName, propertyName, value, reason string) *BuildError {
	return NewError(InvalidValue, fmt.Sprintf("Module %s has invalid value for property %s: %s (%s)", moduleName, propertyName, value, reason))
}

// Config creates a configuration error
func Config(msg string) *BuildError {
	return NewError(ConfigurationError, msg)
}

// Type creates a type mismatch error
func Type(moduleName, propertyName, expected, actual string) *BuildError {
	return NewError(TypeMismatch, fmt.Sprintf("Module %s property %s type mismatch: expected %s, got %s", moduleName, propertyName, expected, actual))
}
