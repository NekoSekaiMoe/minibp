// ninja/writer.go - Ninja build file writer
// This file provides utilities for writing Ninja build file syntax.
// Ninja is a build system that uses build files with explicit dependency graphs.
// Writer translates high-level build concepts into proper Ninja syntax.
package ninja

import (
	"fmt"
	"io"
	"strings"
)

// Writer wraps an io.Writer and provides methods to write Ninja build file syntax.
// It handles escaping of special characters and proper formatting.
type Writer struct {
	w io.Writer
}

// NewWriter creates a new Writer that writes Ninja syntax to the provided writer.
// The returned Writer can be used to write rules, build edges, variables, and comments.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

// ninjaEscape escapes special characters in Ninja build file values.
// Ninja uses $ for variable expansion, : for separators, and # for comments.
// This function escapes these characters by prefixing them with $.
// For example, "$" becomes "$$", ":" becomes "$:", "#" becomes "$#".
func ninjaEscape(s string) string {
	replacer := strings.NewReplacer(
		"$", "$$",
		":", "$:",
		"#", "$#",
	)
	return replacer.Replace(s)
}

// ninjaEscapePath escapes a path for use in Ninja build files.
// It escapes special characters and also escapes spaces.
// Spaces are escaped as "$ " to prevent them from being treated as separators.
func ninjaEscapePath(s string) string {
	return strings.ReplaceAll(ninjaEscape(s), " ", "$ ")
}

// escapeList applies ninjaEscapePath to each string in the values slice.
// It returns a new slice with all values properly escaped for Ninja paths.
func escapeList(values []string) []string {
	result := make([]string, 0, len(values))
	for _, v := range values {
		result = append(result, ninjaEscapePath(v))
	}
	return result
}

// Rule writes a Ninja rule definition to the build file.
// A rule defines a command template that can be reused across multiple build edges.
// The name parameter specifies the rule name (e.g., "cc_compile").
// The command parameter specifies the command template to execute.
// Additional deps can be provided to specify dependency files (e.g., .d files for header dependencies).
// Rule templates can use Ninja variables like $in (input file) and $out (output file).
func (w *Writer) Rule(name, command string, deps ...string) {
	fmt.Fprintf(w.w, "rule %s\n", ninjaEscapePath(name))
	fmt.Fprintf(w.w, "  command = %s\n", ninjaEscape(command))
	if len(deps) > 0 && deps[0] != "" {
		fmt.Fprintf(w.w, "  deps = %s\n", strings.Join(escapeList(deps), " "))
	}
	fmt.Fprintln(w.w)
}

// Build writes a Ninja build edge to the build file.
// A build edge defines a transformation from inputs to output using a rule.
// The output parameter specifies the output file path.
// The rule parameter specifies which rule to use for building.
// The inputs parameter specifies the input files needed for this build edge.
// The deps parameter specifies additional file dependencies that trigger rebuilds.
// The pipe character (|) in Ninja syntax separates order-only dependencies.
func (w *Writer) Build(output, rule string, inputs []string, deps []string) {
	fmt.Fprintf(w.w, "build %s: %s", ninjaEscapePath(output), ninjaEscapePath(rule))
	if len(inputs) > 0 {
		fmt.Fprintf(w.w, " %s", strings.Join(escapeList(inputs), " "))
	}
	if len(deps) > 0 {
		fmt.Fprintf(w.w, " | %s", strings.Join(escapeList(deps), " "))
	}
	fmt.Fprintln(w.w)
	fmt.Fprintln(w.w)
}

// BuildWithVars writes a Ninja build edge with additional variables.
// This is similar to Build but allows specifying custom variables for this specific edge.
// The orderOnly parameter specifies dependencies that must be built first but don't cause rebuilds.
// Variables like "flags" can be defined to pass custom parameters to the rule.
// The double pipe (||) syntax marks order-only dependencies in Ninja.
func (w *Writer) BuildWithVars(output, rule string, inputs []string, orderOnly []string, vars map[string]string) {
	fmt.Fprintf(w.w, "build %s: %s", ninjaEscapePath(output), ninjaEscapePath(rule))
	if len(inputs) > 0 {
		fmt.Fprintf(w.w, " %s", strings.Join(escapeList(inputs), " "))
	}
	if len(orderOnly) > 0 {
		fmt.Fprintf(w.w, " || %s", strings.Join(escapeList(orderOnly), " "))
	}
	fmt.Fprintln(w.w)
	for k, v := range vars {
		fmt.Fprintf(w.w, "  %s = %s\n", ninjaEscape(k), ninjaEscape(v))
	}
	fmt.Fprintln(w.w)
}

// Variable writes a Ninja variable definition to the build file.
// Variables can be used in rules and build edges using $variable_name syntax.
// The name parameter specifies the variable name.
// The value parameter specifies the variable value.
func (w *Writer) Variable(name, value string) {
	fmt.Fprintf(w.w, "%s = %s\n", ninjaEscape(name), ninjaEscape(value))
}

// Comment writes a Ninja comment to the build file.
// Comments start with # and are ignored by Ninja but useful for documentation.
// If text is empty, writes an empty line for formatting purposes.
func (w *Writer) Comment(text string) {
	if text != "" {
		fmt.Fprintf(w.w, "# %s\n", text)
	} else {
		fmt.Fprintln(w.w)
	}
}

// Desc writes a description comment for a build edge in Ninja format.
// This follows the Bazel/Blaze style description format used by many build tools.
// sourceDir is the source directory containing the module.
// moduleName is the name of the module being built.
// action describes what action is being performed (e.g., "gcc", "ar", "javac").
// srcFile is optional and specifies the source file for this specific action.
func (w *Writer) Desc(sourceDir, moduleName, action string, srcFile ...string) {
	srcStr := ""
	if len(srcFile) > 0 && srcFile[0] != "" {
		srcStr = " " + srcFile[0]
	}
	fmt.Fprintf(w.w, "# //%s:%s %s%s\n", sourceDir, moduleName, action, srcStr)
}

// Subninja includes another Ninja build file as a sub-build.
// This allows splitting build files into modular pieces.
// The path parameter specifies the path to the sub-Ninja file to include.
func (w *Writer) Subninja(path string) {
	fmt.Fprintf(w.w, "subninja %s\n\n", ninjaEscapePath(path))
}

// Include includes a Ninja build file at the point where it's invoked.
// Unlike subninja, included files are processed in place.
// The path parameter specifies the path to the Ninja file to include.
func (w *Writer) Include(path string) {
	fmt.Fprintf(w.w, "include %s\n\n", ninjaEscapePath(path))
}

// Phony creates a phony build target that aliases other targets.
// This is useful for creating convenience targets that build multiple outputs.
// The output parameter specifies the name of the phony target.
// The inputs parameter specifies the actual targets this phony target represents.
// Running "ninja output" will build all the input targets.
func (w *Writer) Phony(output string, inputs []string) {
	fmt.Fprintf(w.w, "build %s: phony %s\n", ninjaEscapePath(output), strings.Join(escapeList(inputs), " "))
}

// Default specifies the default targets to build when running "ninja" without arguments.
// Multiple targets can be specified; Ninja will build the first one by default.
// The targets parameter specifies which targets should be built by default.
func (w *Writer) Default(targets []string) {
	fmt.Fprintf(w.w, "default %s\n", strings.Join(escapeList(targets), " "))
}
