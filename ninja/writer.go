// ninja/writer.go - Ninja build file writer
package ninja

import (
	"fmt"
	"io"
	"strings"
)

type Writer struct {
	w io.Writer
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

func ninjaEscape(s string) string {
	replacer := strings.NewReplacer(
		"$", "$$",
		":", "$:",
		"#", "$#",
	)
	return replacer.Replace(s)
}

func ninjaEscapePath(s string) string {
	return strings.ReplaceAll(ninjaEscape(s), " ", "$ ")
}

func escapeList(values []string) []string {
	result := make([]string, 0, len(values))
	for _, v := range values {
		result = append(result, ninjaEscapePath(v))
	}
	return result
}

func (w *Writer) Rule(name, command string, deps ...string) {
	fmt.Fprintf(w.w, "rule %s\n", ninjaEscapePath(name))
	fmt.Fprintf(w.w, "  command = %s\n", ninjaEscape(command))
	if len(deps) > 0 && deps[0] != "" {
		fmt.Fprintf(w.w, "  deps = %s\n", strings.Join(escapeList(deps), " "))
	}
	fmt.Fprintln(w.w)
}

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

func (w *Writer) Variable(name, value string) {
	fmt.Fprintf(w.w, "%s = %s\n", ninjaEscape(name), ninjaEscape(value))
}

func (w *Writer) Comment(text string) {
	if text != "" {
		fmt.Fprintf(w.w, "# %s\n", text)
	} else {
		fmt.Fprintln(w.w)
	}
}

func (w *Writer) Desc(sourceDir, moduleName, action string, srcFile ...string) {
	srcStr := ""
	if len(srcFile) > 0 && srcFile[0] != "" {
		srcStr = " " + srcFile[0]
	}
	fmt.Fprintf(w.w, "# //%s:%s %s%s\n", sourceDir, moduleName, action, srcStr)
}

func (w *Writer) Subninja(path string) {
	fmt.Fprintf(w.w, "subninja %s\n\n", ninjaEscapePath(path))
}

func (w *Writer) Include(path string) {
	fmt.Fprintf(w.w, "include %s\n\n", ninjaEscapePath(path))
}

func (w *Writer) Phony(output string, inputs []string) {
	fmt.Fprintf(w.w, "build %s: phony %s\n", ninjaEscapePath(output), strings.Join(escapeList(inputs), " "))
}

func (w *Writer) Default(targets []string) {
	fmt.Fprintf(w.w, "default %s\n", strings.Join(escapeList(targets), " "))
}
