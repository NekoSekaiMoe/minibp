// ninja/rules.go - Ninja rule definitions for minibp
// This file defines all the build rules for different module types.
// Each rule implements the BuildRule interface to generate Ninja syntax.
package ninja

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"minibp/parser"
)

// BuildRule is the interface for all ninja rule implementations.
// Each module type (cc_library, go_binary, java_library, etc.) implements this interface
// to translate module definitions into Ninja build file syntax.
type BuildRule interface {
	// Name returns the rule module type name (e.g., "cc_library", "go_binary").
	Name() string
	// NinjaRule returns the Ninja rule definitions for this build rule.
	// This outputs the "rule" declarations that define how to build outputs.
	NinjaRule(ctx RuleRenderContext) string
	// NinjaEdge returns the Ninja build edges for a module.
	// This outputs the "build" declarations that define specific build steps.
	NinjaEdge(m *parser.Module, ctx RuleRenderContext) string
	// Outputs returns the output files produced by this module.
	Outputs(m *parser.Module, ctx RuleRenderContext) []string
	// Desc returns a description of what action is performed.
	// This is used for generating comments in the build file.
	Desc(m *parser.Module, srcFile string) string
}

// RuleRenderContext holds the toolchain configuration for rendering rules.
// This includes compilers, archivers, and flag configurations.
type RuleRenderContext struct {
	CC         string // C compiler command (e.g., gcc, clang)
	CXX        string // C++ compiler command (e.g., g++, clang++)
	AR         string // Static library archiver (e.g., ar)
	ArchSuffix string // Architecture suffix for output files (e.g., "_x86_64")
	CFlags     string // Global C/C++ compiler flags
	LdFlags    string // Global linker flags
}

// DefaultRuleRenderContext returns a RuleRenderContext with default toolchain values.
// Uses common GNU/Linux development tools as defaults.
func DefaultRuleRenderContext() RuleRenderContext {
	return RuleRenderContext{
		CC:  "gcc",
		CXX: "g++",
		AR:  "ar",
	}
}

// libOutputName generates the output name for a library.
// It prefixes the name with "lib" and adds the architecture suffix and extension.
// This follows the Unix convention of lib<name>.so or lib<name>.a.
func libOutputName(name, archSuffix, ext string) string {
	return "lib" + name + archSuffix + ext
}

// sharedLibOutputName generates the output name for a shared library (.so).
// Uses .so extension with architecture suffix.
func sharedLibOutputName(name string, ctx RuleRenderContext) string {
	return libOutputName(name, ctx.ArchSuffix, ".so")
}

// staticLibOutputName generates the output name for a static library (.a).
// Uses .a extension with architecture suffix.
func staticLibOutputName(name string, ctx RuleRenderContext) string {
	return libOutputName(name, ctx.ArchSuffix, ".a")
}

// GetStringProp retrieves a string property value from a module.
// Looks up the property by name and returns its string value.
// Returns empty string if the property doesn't exist or isn't a string.
func GetStringProp(m *parser.Module, name string) string {
	if m.Map == nil {
		return ""
	}
	for _, prop := range m.Map.Properties {
		if prop.Name == name {
			if s, ok := prop.Value.(*parser.String); ok {
				return s.Value
			}
		}
	}
	return ""
}

// GetStringPropEval retrieves a string property value with optional evaluation.
// If the property is a string, returns its value directly.
// If an evaluator is provided, attempts to evaluate the property value.
// This is useful for properties that can contain template expressions.
func GetStringPropEval(m *parser.Module, name string, eval *parser.Evaluator) string {
	if m.Map == nil {
		return ""
	}
	for _, prop := range m.Map.Properties {
		if prop.Name == name {
			if s, ok := prop.Value.(*parser.String); ok {
				return s.Value
			}
			if eval != nil {
				val := eval.Eval(prop.Value)
				if s, ok := val.(string); ok {
					return s
				}
			}
		}
	}
	return ""
}

// getBoolProp retrieves a boolean property value from a module.
// Looks up the property by name and returns its boolean value.
// Returns false if the property doesn't exist or isn't a boolean.
func getBoolProp(m *parser.Module, name string) bool {
	if m.Map == nil {
		return false
	}
	for _, prop := range m.Map.Properties {
		if prop.Name == name {
			if b, ok := prop.Value.(*parser.Bool); ok {
				return b.Value
			}
		}
	}
	return false
}

// GetListProp retrieves a list property value from a module.
// Looks up the property by name and returns a slice of strings.
// Returns nil if the property doesn't exist or isn't a list.
func GetListProp(m *parser.Module, name string) []string {
	if m.Map == nil {
		return nil
	}
	for _, prop := range m.Map.Properties {
		if prop.Name == name {
			if l, ok := prop.Value.(*parser.List); ok {
				var result []string
				for _, v := range l.Values {
					if s, ok := v.(*parser.String); ok {
						result = append(result, s.Value)
					}
				}
				return result
			}
		}
	}
	return nil
}

// GetListPropEval retrieves a list property value with optional evaluation.
// If an evaluator is provided, evaluates each element in the list.
// Returns nil if the property doesn't exist or isn't a list.
func GetListPropEval(m *parser.Module, name string, eval *parser.Evaluator) []string {
	if m.Map == nil {
		return nil
	}
	for _, prop := range m.Map.Properties {
		if prop.Name == name {
			if l, ok := prop.Value.(*parser.List); ok {
				return parser.EvalToStringList(l, eval)
			}
		}
	}
	return nil
}

// getCflags retrieves C compiler flags from a module.
// Combines all cflags property values into a single space-separated string.
func getCflags(m *parser.Module) string { return strings.Join(GetListProp(m, "cflags"), " ") }

// getCppflags retrieves C++ compiler flags from a module.
// Combines all cppflags property values into a single space-separated string.
func getCppflags(m *parser.Module) string { return strings.Join(GetListProp(m, "cppflags"), " ") }

// getLdflags retrieves linker flags from a module.
// Combines all ldflags property values into a single space-separated string.
func getLdflags(m *parser.Module) string { return strings.Join(GetListProp(m, "ldflags"), " ") }

// getGoflags retrieves Go compiler flags from a module.
// Combines all goflags property values into a single space-separated string.
func getGoflags(m *parser.Module) string { return strings.Join(GetListProp(m, "goflags"), " ") }

// getJavaflags retrieves Java compiler flags from a module.
// Combines all javaflags property values into a single space-separated string.
func getJavaflags(m *parser.Module) string { return strings.Join(GetListProp(m, "javaflags"), " ") }

// getExportIncludeDirs retrieves exported include directories from a module.
// These are directories that should be added to dependent modules' include paths.
func getExportIncludeDirs(m *parser.Module) []string { return GetListProp(m, "export_include_dirs") }

// getExportedHeaders retrieves exported header files from a module.
// These are header files that should be available to dependent modules.
func getExportedHeaders(m *parser.Module) []string { return GetListProp(m, "exported_headers") }

// getName retrieves the module name from a module.
func getName(m *parser.Module) string { return GetStringProp(m, "name") }

// getSrcs retrieves source file paths from a module.
func getSrcs(m *parser.Module) []string { return GetListProp(m, "srcs") }

// formatSrcs combines source file paths into a single space-separated string.
func formatSrcs(srcs []string) string { return strings.Join(srcs, " ") }

// objectOutputName generates a unique object file name for a source file.
// It transforms the source path into a valid object file name by:
// - Removing path prefixes (./ and ../)
// - Replacing path separators, colons, and spaces with underscores
// - Appending the module name as a prefix
// This ensures unique output files even for sources with the same base name in different directories.
func objectOutputName(moduleName, src string) string {
	clean := filepath.Clean(src)
	clean = strings.TrimPrefix(clean, "./")
	clean = strings.TrimPrefix(clean, "../")
	name := strings.TrimSuffix(clean, filepath.Ext(clean))
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", " ", "_")
	name = replacer.Replace(name)
	name = strings.Trim(name, "._")
	if name == "" {
		name = "obj"
	}
	return moduleName + "_" + name + ".o"
}

// joinFlags combines multiple flag strings into a single space-separated string.
// Empty flags are filtered out to avoid adding unnecessary spaces to commands.
func joinFlags(parts ...string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	return strings.Join(filtered, " ")
}

// ============================================================================
// cc_library - C library (static by default, shared if shared: true)
// ============================================================================

// ccLibrary implements a C/C++ library rule that can be either static or shared.
// By default, produces a static library (.a). If "shared: true" is set,
// produces a shared library (.so) instead.
type ccLibrary struct{}

// Name returns "cc_library" for this build rule.
func (r *ccLibrary) Name() string { return "cc_library" }

// NinjaRule returns the Ninja rule definitions for compiling and archiving C code.
// Defines three rules: cc_compile (compile .c to .o), cc_archive (create static lib),
// and cc_shared (create shared library).
func (r *ccLibrary) NinjaRule(ctx RuleRenderContext) string {
	return fmt.Sprintf(`rule cc_compile
  command = %s -c $in -o $out $flags -MMD -MF $out.d
  depfile = $out.d
  deps = gcc

rule cc_archive
  command = %s rcs $out $in

rule cc_shared
  command = %s -shared -o $out $in $flags
 `, ctx.CC, ctx.AR, ctx.CC)
}

// ccLibraryStatic implements a C static library rule (always produces .a).
type ccLibraryStatic struct{}

// Name returns "cc_library_static" for this build rule.
func (r *ccLibraryStatic) Name() string { return "cc_library_static" }

// NinjaRule returns the Ninja rule definitions for compiling and archiving C code.
// Only defines cc_compile and cc_archive rules (no shared library support).
func (r *ccLibraryStatic) NinjaRule(ctx RuleRenderContext) string {
	return fmt.Sprintf(`rule cc_compile
  command = %s -c $in -o $out $flags -MMD -MF $out.d
  depfile = $out.d
  deps = gcc

rule cc_archive
  command = %s rcs $out $in
 `, ctx.CC, ctx.AR)
}

// ccLibraryShared implements a C shared library rule (always produces .so).
// Unlike ccLibrary, this always produces a shared library, never static.
type ccLibraryShared struct{}

// Name returns "cc_library_shared" for this build rule.
func (r *ccLibraryShared) Name() string { return "cc_library_shared" }

// NinjaRule returns the Ninja rule definitions for compiling C code and creating shared libs.
// Defines cc_compile and cc_shared rules (no archive rule).
func (r *ccLibraryShared) NinjaRule(ctx RuleRenderContext) string {
	return fmt.Sprintf(`rule cc_compile
  command = %s -c $in -o $out $flags -MMD -MF $out.d
  depfile = $out.d
  deps = gcc

rule cc_shared
  command = %s -shared -o $out $in $flags
 `, ctx.CC, ctx.CC)
}

// Outputs returns the output files produced by a cc_library module.
// If the "shared" property is true, returns the .so file path.
// Otherwise, returns the .a file path.
// The output name includes any architecture suffix.
func (r *ccLibrary) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	suffix := ctx.ArchSuffix
	if getBoolProp(m, "shared") {
		return []string{fmt.Sprintf("lib%s%s.so", name, suffix)}
	}
	return []string{fmt.Sprintf("lib%s%s.a", name, suffix)}
}

// NinjaEdge generates the build edges for a cc_library module.
// For each source file, creates a compile edge to produce an object file.
// Then creates either an archive edge (for static) or shared library edge (for shared).
// Handles shared_libs dependencies when building shared libraries.
func (r *ccLibrary) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string {
	name := getName(m)
	srcs := getSrcs(m)
	if name == "" || len(srcs) == 0 {
		return ""
	}
	shared := getBoolProp(m, "shared")
	cflags := joinFlags(ctx.CFlags, getCflags(m))
	ldflags := joinFlags(ctx.LdFlags, getLdflags(m))
	var sharedInputs []string
	sharedLibs := GetListProp(m, "shared_libs")
	if shared && len(sharedLibs) > 0 {
		for _, dep := range sharedLibs {
			depName := strings.TrimPrefix(dep, ":")
			sharedInputs = append(sharedInputs, sharedLibOutputName(depName, ctx))
			ldflags = joinFlags(ldflags, "-l"+depName)
		}
	}
	var edges strings.Builder
	var objFiles []string

	for _, src := range srcs {
		obj := objectOutputName(name, src)
		objFiles = append(objFiles, obj)
		edges.WriteString(fmt.Sprintf("build %s: cc_compile %s\n flags = %s\n", obj, src, cflags))
	}

	out := r.Outputs(m, ctx)[0]
	if shared {
		allInputs := append(objFiles, sharedInputs...)
		edges.WriteString(fmt.Sprintf("build %s: cc_shared %s\n flags = %s\n", out, strings.Join(allInputs, " "), ldflags))
	} else {
		edges.WriteString(fmt.Sprintf("build %s: cc_archive %s\n", out, strings.Join(objFiles, " ")))
	}
	return edges.String()
}

// Desc returns a description of the action performed by this module.
// For source files, returns "gcc" (compiler).
// For the library output, returns "ar" (archiver) or "cc_shared".
func (r *ccLibrary) Desc(m *parser.Module, srcFile string) string {
	if srcFile == "" {
		if getBoolProp(m, "shared") {
			return "cc_shared"
		}
		return "ar"
	}
	return "gcc"
}

// ============================================================================
// cc_library_static
// ============================================================================

// Outputs returns the static library output file path for a cc_library_static module.
func (r *ccLibraryStatic) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	return []string{fmt.Sprintf("lib%s%s.a", name, ctx.ArchSuffix)}
}

// NinjaEdge generates build edges for a static C library.
// Compiles each source file to an object file, then archives them into a .a library.
func (r *ccLibraryStatic) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string {
	name := getName(m)
	srcs := getSrcs(m)
	if name == "" || len(srcs) == 0 {
		return ""
	}
	cflags := joinFlags(ctx.CFlags, getCflags(m))
	var edges strings.Builder
	var objFiles []string
	for _, src := range srcs {
		obj := objectOutputName(name, src)
		objFiles = append(objFiles, obj)
		edges.WriteString(fmt.Sprintf("build %s: cc_compile %s\n flags = %s\n", obj, src, cflags))
	}
	out := r.Outputs(m, ctx)[0]
	edges.WriteString(fmt.Sprintf("build %s: cc_archive %s\n", out, strings.Join(objFiles, " ")))
	return edges.String()
}

// Desc returns a description for this build rule.
// Returns "ar" for the library output, "gcc" for individual source files.
func (r *ccLibraryStatic) Desc(m *parser.Module, srcFile string) string {
	if srcFile == "" {
		return "ar"
	}
	return "gcc"
}

// ============================================================================
// cc_library_shared
// ============================================================================

// Outputs returns the shared library output file path for a cc_library_shared module.
func (r *ccLibraryShared) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	return []string{fmt.Sprintf("lib%s%s.so", name, ctx.ArchSuffix)}
}

// NinjaEdge generates build edges for a shared C library.
// Compiles each source file to an object file, then links them into a .so with shared library dependencies.
func (r *ccLibraryShared) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string {
	name := getName(m)
	srcs := getSrcs(m)
	if name == "" || len(srcs) == 0 {
		return ""
	}
	cflags := joinFlags(ctx.CFlags, getCflags(m))
	ldflags := joinFlags(ctx.LdFlags, getLdflags(m))
	var sharedInputs []string
	sharedLibs := GetListProp(m, "shared_libs")
	for _, dep := range sharedLibs {
		depName := strings.TrimPrefix(dep, ":")
		sharedInputs = append(sharedInputs, sharedLibOutputName(depName, ctx))
		ldflags = joinFlags(ldflags, "-l"+depName)
	}
	var edges strings.Builder
	var objFiles []string
	for _, src := range srcs {
		obj := objectOutputName(name, src)
		objFiles = append(objFiles, obj)
		edges.WriteString(fmt.Sprintf("build %s: cc_compile %s\n flags = %s\n", obj, src, cflags))
	}
	out := r.Outputs(m, ctx)[0]
	allInputs := append(objFiles, sharedInputs...)
	edges.WriteString(fmt.Sprintf("build %s: cc_shared %s\n flags = %s\n", out, strings.Join(allInputs, " "), ldflags))
	return edges.String()
}

// Desc returns a description for this build rule.
// Returns "cc_shared" for the library output, "gcc" for individual source files.
func (r *ccLibraryShared) Desc(m *parser.Module, srcFile string) string {
	if srcFile == "" {
		return "cc_shared"
	}
	return "gcc"
}

// ============================================================================
// cc_object
// ============================================================================

// ccObject implements a C object file rule for compiling individual .c files to .o files.
// Produces one object file per source file, which can be later linked into libraries or binaries.
type ccObject struct{}

// Name returns "cc_object" for this build rule.
func (r *ccObject) Name() string { return "cc_object" }

// NinjaRule returns the Ninja rule definition for compiling C files.
// Uses the same cc_compile rule as cc_library.
func (r *ccObject) NinjaRule(ctx RuleRenderContext) string {
	return fmt.Sprintf(`rule cc_compile
  command = %s -c $in -o $out $flags -MMD -MF $out.d
  depfile = $out.d
  deps = gcc
 `, ctx.CC)
}

// Outputs returns the object file outputs for a cc_object module.
// If there's only one source, returns a simple output name.
// If multiple sources, each gets a unique output based on its path.
func (r *ccObject) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	srcs := getSrcs(m)
	if name == "" {
		return nil
	}
	if len(srcs) <= 1 {
		return []string{fmt.Sprintf("%s%s.o", name, ctx.ArchSuffix)}
	}
	outputs := make([]string, 0, len(srcs))
	for _, src := range srcs {
		outputs = append(outputs, objectOutputName(name, src))
	}
	return outputs
}
func (r *ccObject) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string {
	name := getName(m)
	srcs := getSrcs(m)
	if name == "" || len(srcs) == 0 {
		return ""
	}
	cflags := joinFlags(ctx.CFlags, getCflags(m))
	if len(srcs) == 1 {
		out := r.Outputs(m, ctx)[0]
		return fmt.Sprintf("build %s: cc_compile %s\n flags = %s\n", out, srcs[0], cflags)
	}
	var edges strings.Builder
	outputs := r.Outputs(m, ctx)
	for i, src := range srcs {
		edges.WriteString(fmt.Sprintf("build %s: cc_compile %s\n flags = %s\n", outputs[i], src, cflags))
	}
	return edges.String()
}
func (r *ccObject) Desc(m *parser.Module, srcFile string) string { return "gcc" }

// ============================================================================
// cc_binary
// ============================================================================
type ccBinary struct{}

func (r *ccBinary) Name() string { return "cc_binary" }
func (r *ccBinary) NinjaRule(ctx RuleRenderContext) string {
	return fmt.Sprintf(`rule cc_compile
  command = %s -c $in -o $out $flags -MMD -MF $out.d
  depfile = $out.d
  deps = gcc

rule cc_link
  command = %s -o $out $in $flags
 `, ctx.CC, ctx.CC)
}
func (r *ccBinary) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	return []string{name + ctx.ArchSuffix}
}

func (r *ccBinary) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string {
	name := getName(m)
	srcs := getSrcs(m)
	deps := GetListProp(m, "deps")
	sharedLibs := GetListProp(m, "shared_libs")
	if name == "" || len(srcs) == 0 {
		return ""
	}
	cflags := joinFlags(ctx.CFlags, getCflags(m))
	ldflags := joinFlags(ctx.LdFlags, getLdflags(m))
	linkFlags := ldflags
	var libFiles []string
	for _, dep := range deps {
		depName := strings.TrimPrefix(dep, ":")
		libFiles = append(libFiles, staticLibOutputName(depName, ctx))
	}
	for _, dep := range sharedLibs {
		depName := strings.TrimPrefix(dep, ":")
		libFiles = append(libFiles, sharedLibOutputName(depName, ctx))
		linkFlags = joinFlags(linkFlags, "-l"+depName)
	}
	var edges strings.Builder
	var objFiles []string
	for _, src := range srcs {
		obj := objectOutputName(name, src)
		objFiles = append(objFiles, obj)
		edges.WriteString(fmt.Sprintf("build %s: cc_compile %s\n flags = %s\n", obj, src, cflags))
	}
	out := r.Outputs(m, ctx)[0]
	allInputs := append(objFiles, libFiles...)
	edges.WriteString(fmt.Sprintf("build %s: cc_link %s\n flags = %s\n", out, strings.Join(allInputs, " "), linkFlags))
	return edges.String()
}
func (r *ccBinary) Desc(m *parser.Module, srcFile string) string {
	if srcFile == "" {
		return "cc_link"
	}
	return "gcc"
}

// ============================================================================
// cpp_library
// ============================================================================
type cppLibrary struct{}

func (r *cppLibrary) Name() string { return "cpp_library" }
func (r *cppLibrary) NinjaRule(ctx RuleRenderContext) string {
	return fmt.Sprintf(`rule cpp_compile
  command = %s -c $in -o $out $flags -MMD -MF $out.d
  depfile = $out.d
  deps = gcc

rule cpp_archive
  command = %s rcs $out $in

rule cpp_shared
  command = %s -shared -o $out $in $flags
 `, ctx.CXX, ctx.AR, ctx.CXX)
}
func (r *cppLibrary) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	suffix := ctx.ArchSuffix
	if getBoolProp(m, "shared") {
		return []string{fmt.Sprintf("lib%s%s.so", name, suffix)}
	}
	return []string{fmt.Sprintf("lib%s%s.a", name, suffix)}
}
func (r *cppLibrary) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string {
	name := getName(m)
	srcs := getSrcs(m)
	if name == "" || len(srcs) == 0 {
		return ""
	}
	shared := getBoolProp(m, "shared")
	cflags := getCflags(m)
	cppflags := getCppflags(m)
	ldflags := getLdflags(m)
	allFlags := joinFlags(cflags, cppflags)
	var sharedInputs []string
	if shared {
		sharedLibs := GetListProp(m, "shared_libs")
		for _, dep := range sharedLibs {
			depName := strings.TrimPrefix(dep, ":")
			sharedInputs = append(sharedInputs, sharedLibOutputName(depName, ctx))
			ldflags = joinFlags(ldflags, "-l"+depName)
		}
	}
	var edges strings.Builder
	var objFiles []string
	for _, src := range srcs {
		obj := objectOutputName(name, src)
		objFiles = append(objFiles, obj)
		edges.WriteString(fmt.Sprintf("build %s: cpp_compile %s\n flags = %s\n", obj, src, allFlags))
	}
	out := r.Outputs(m, ctx)[0]
	if shared {
		allInputs := append(objFiles, sharedInputs...)
		edges.WriteString(fmt.Sprintf("build %s: cpp_shared %s\n flags = %s\n", out, strings.Join(allInputs, " "), ldflags))
	} else {
		edges.WriteString(fmt.Sprintf("build %s: cpp_archive %s\n", out, strings.Join(objFiles, " ")))
	}
	return edges.String()
}
func (r *cppLibrary) Desc(m *parser.Module, srcFile string) string {
	if srcFile == "" {
		if getBoolProp(m, "shared") {
			return "cpp_shared"
		}
		return "ar"
	}
	return "g++"
}

// ============================================================================
// cpp_binary
// ============================================================================
type cppBinary struct{}

func (r *cppBinary) Name() string { return "cpp_binary" }
func (r *cppBinary) NinjaRule(ctx RuleRenderContext) string {
	return fmt.Sprintf(`rule cpp_compile
  command = %s -c $in -o $out $flags -MMD -MF $out.d
  depfile = $out.d
  deps = gcc

rule cpp_link
  command = %s -o $out $in $flags
 `, ctx.CXX, ctx.CXX)
}
func (r *cppBinary) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	return []string{name + ctx.ArchSuffix}
}
func (r *cppBinary) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string {
	name := getName(m)
	srcs := getSrcs(m)
	deps := GetListProp(m, "deps")
	sharedLibs := GetListProp(m, "shared_libs")
	if name == "" || len(srcs) == 0 {
		return ""
	}
	cflags := joinFlags(ctx.CFlags, getCflags(m))
	cppflags := getCppflags(m)
	ldflags := joinFlags(ctx.LdFlags, getLdflags(m))
	allFlags := joinFlags(cflags, cppflags)
	linkFlags := ldflags
	var libFiles []string
	for _, dep := range deps {
		depName := strings.TrimPrefix(dep, ":")
		libFiles = append(libFiles, staticLibOutputName(depName, ctx))
	}
	for _, dep := range sharedLibs {
		depName := strings.TrimPrefix(dep, ":")
		libFiles = append(libFiles, sharedLibOutputName(depName, ctx))
		linkFlags = joinFlags(linkFlags, "-l"+depName)
	}
	var edges strings.Builder
	var objFiles []string
	for _, src := range srcs {
		obj := objectOutputName(name, src)
		objFiles = append(objFiles, obj)
		edges.WriteString(fmt.Sprintf("build %s: cpp_compile %s\n flags = %s\n", obj, src, allFlags))
	}
	out := r.Outputs(m, ctx)[0]
	allInputs := append(objFiles, libFiles...)
	edges.WriteString(fmt.Sprintf("build %s: cpp_link %s\n flags = %s\n", out, strings.Join(allInputs, " "), linkFlags))
	return edges.String()
}

func (r *cppBinary) Desc(m *parser.Module, srcFile string) string {
	if srcFile == "" {
		return "cpp_link"
	}
	return "g++"
}

// ============================================================================
// go_library - Go library (.a)
// ============================================================================

// goLibrary implements a Go library rule that produces a .a archive file.
// Uses Go's -buildmode=archive to create an archive for linking.
type goLibrary struct{}

// Name returns "go_library" for this build rule.
func (r *goLibrary) Name() string { return "go_library" }

// NinjaRule returns the Ninja rule for building Go archives.
// Uses "go build -buildmode=archive" to create static archives.
func (r *goLibrary) NinjaRule(ctx RuleRenderContext) string {
	return `rule go_build_archive
 command = go build -buildmode=archive -o $out $in
 `
}

// Outputs returns the archive output file path for a go_library module.
func (r *goLibrary) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	return []string{fmt.Sprintf("%s.a", name)}
}

// NinjaEdge generates the build edge for a Go library.
func (r *goLibrary) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string {
	name := getName(m)
	srcs := getSrcs(m)
	if name == "" || len(srcs) == 0 {
		return ""
	}
	goflags := getGoflags(m)
	out := r.Outputs(m, ctx)[0]
	return fmt.Sprintf("build %s: go_build_archive %s\n flags = %s\n", out, strings.Join(srcs, " "), goflags)
}

// Desc returns a description for this build rule.
func (r *goLibrary) Desc(m *parser.Module, srcFile string) string { return "go" }

// ============================================================================
// go_binary - Go executable
// ============================================================================

// goBinary implements a Go binary rule that produces an executable.
type goBinary struct{}

// Name returns "go_binary" for this build rule.
func (r *goBinary) Name() string { return "go_binary" }

// NinjaRule returns the Ninja rule for building Go binaries.
// Uses "go build" to create executables.
func (r *goBinary) NinjaRule(ctx RuleRenderContext) string {
	return `rule go_build
 command = go build -o $out $in
 `
}

// Outputs returns the executable output file path for a go_binary module.
func (r *goBinary) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	return []string{name}
}

// NinjaEdge generates the build edge for a Go binary.
// Includes dependencies as order-only dependencies (using | syntax).
func (r *goBinary) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string {
	name := getName(m)
	srcs := getSrcs(m)
	deps := GetListProp(m, "deps")
	if name == "" || len(srcs) == 0 {
		return ""
	}
	goflags := getGoflags(m)
	out := r.Outputs(m, ctx)[0]

	var libFiles []string
	for _, dep := range deps {
		depName := strings.TrimPrefix(dep, ":")
		libFiles = append(libFiles, depName+".a")
	}

	srcStr := strings.Join(srcs, " ")
	if len(libFiles) > 0 {
		libStr := strings.Join(libFiles, " ")
		return fmt.Sprintf("build %s: go_build %s | %s\n flags = %s\n", out, srcStr, libStr, goflags)
	}
	return fmt.Sprintf("build %s: go_build %s\n flags = %s\n", out, srcStr, goflags)
}

// Desc returns a description for this build rule.
func (r *goBinary) Desc(m *parser.Module, srcFile string) string { return "go" }

// ============================================================================
// go_test
// ============================================================================
type goTest struct{}

func (r *goTest) Name() string { return "go_test" }
func (r *goTest) NinjaRule(ctx RuleRenderContext) string {
	return `rule go_test
 command = go test -c -o $out $pkg
`
}

func (r *goTest) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	return []string{fmt.Sprintf("%s.test", name)}
}
func (r *goTest) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string {
	name := getName(m)
	srcs := getSrcs(m)
	if name == "" || len(srcs) == 0 {
		return ""
	}
	goflags := getGoflags(m)
	out := r.Outputs(m, ctx)[0]
	// Extract package path from first source file
	// Convert "dag/graph_test.go" to "./dag"
	pkgPath := "./" + filepath.Dir(srcs[0])

	// Build the test binary
	// Use go test -c which requires a package path
	return fmt.Sprintf("build %s: go_test\n pkg = %s\n flags = %s\n", out, pkgPath, goflags)
}
func (r *goTest) Desc(m *parser.Module, srcFile string) string { return "go test" }

// ============================================================================
// java_library
// ============================================================================
type javaLibrary struct{}

func (r *javaLibrary) Name() string { return "java_library" }
func (r *javaLibrary) NinjaRule(ctx RuleRenderContext) string {
	return `rule javac_lib
  command = javac -d $outdir $in $flags

rule jar_create
  command = jar cf $out -C $outdir .
`
}
func (r *javaLibrary) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	return []string{fmt.Sprintf("%s.jar", name)}
}
func (r *javaLibrary) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string {
	name := getName(m)
	srcs := getSrcs(m)
	if name == "" || len(srcs) == 0 {
		return ""
	}
	javaflags := getJavaflags(m)
	out := r.Outputs(m, ctx)[0]
	outdir := name + "_classes"
	var edges strings.Builder
	edges.WriteString(fmt.Sprintf("build %s.stamp: javac_lib %s\n outdir = %s\n flags = %s\n", name, strings.Join(srcs, " "), outdir, javaflags))
	edges.WriteString(fmt.Sprintf("build %s: jar_create %s.stamp\n outdir = %s\n", out, name, outdir))
	return edges.String()
}
func (r *javaLibrary) Desc(m *parser.Module, srcFile string) string {
	if srcFile == "" {
		return "jar"
	}
	return "javac"
}

// ============================================================================
// java_binary
// ============================================================================
type javaBinary struct{}

func (r *javaBinary) Name() string { return "java_binary" }
func (r *javaBinary) NinjaRule(ctx RuleRenderContext) string {
	return `rule javac_bin
  command = javac -d $outdir $in $flags

rule jar_create_executable
  command = jar cfe $out $main_class -C $outdir .
`
}
func (r *javaBinary) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	return []string{fmt.Sprintf("%s.jar", name)}
}
func (r *javaBinary) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string {
	name := getName(m)
	srcs := getSrcs(m)
	mainClass := GetStringProp(m, "main_class")
	if name == "" || len(srcs) == 0 || mainClass == "" {
		return ""
	}
	javaflags := getJavaflags(m)
	out := r.Outputs(m, ctx)[0]
	outdir := name + "_classes"
	var edges strings.Builder
	edges.WriteString(fmt.Sprintf("build %s.stamp: javac_bin %s\n outdir = %s\n flags = %s\n", name, strings.Join(srcs, " "), outdir, javaflags))
	edges.WriteString(fmt.Sprintf("build %s: jar_create_executable %s.stamp\n outdir = %s\n main_class = %s\n", out, name, outdir, mainClass))
	return edges.String()
}
func (r *javaBinary) Desc(m *parser.Module, srcFile string) string {
	if srcFile == "" {
		return "jar"
	}
	return "javac"
}

// ============================================================================
// java_library_static
// ============================================================================
type javaLibraryStatic struct{}

func (r *javaLibraryStatic) Name() string { return "java_library_static" }
func (r *javaLibraryStatic) NinjaRule(ctx RuleRenderContext) string {
	return `rule javac_lib
  command = javac -d $outdir $in $flags

rule jar_create
  command = jar cf $out -C $outdir .
`
}
func (r *javaLibraryStatic) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	return []string{fmt.Sprintf("lib%s.a.jar", name)}
}
func (r *javaLibraryStatic) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string {
	name := getName(m)
	srcs := getSrcs(m)
	if name == "" || len(srcs) == 0 {
		return ""
	}
	javaflags := getJavaflags(m)
	out := r.Outputs(m, ctx)[0]
	outdir := name + "_classes"
	var edges strings.Builder
	edges.WriteString(fmt.Sprintf("build %s.stamp: javac_lib %s\n outdir = %s\n flags = %s\n", name, strings.Join(srcs, " "), outdir, javaflags))
	edges.WriteString(fmt.Sprintf("build %s: jar_create %s.stamp\n outdir = %s\n", out, name, outdir))
	return edges.String()
}
func (r *javaLibraryStatic) Desc(m *parser.Module, srcFile string) string {
	if srcFile == "" {
		return "jar"
	}
	return "javac"
}

// ============================================================================
// java_library_host
// ============================================================================
type javaLibraryHost struct{}

func (r *javaLibraryHost) Name() string { return "java_library_host" }
func (r *javaLibraryHost) NinjaRule(ctx RuleRenderContext) string {
	return `rule javac_lib
  command = javac -d $outdir $in $flags

rule jar_create
  command = jar cf $out -C $outdir .
`
}
func (r *javaLibraryHost) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	return []string{fmt.Sprintf("%s-host.jar", name)}
}
func (r *javaLibraryHost) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string {
	name := getName(m)
	srcs := getSrcs(m)
	if name == "" || len(srcs) == 0 {
		return ""
	}
	javaflags := getJavaflags(m)
	out := r.Outputs(m, ctx)[0]
	outdir := name + "_classes"
	var edges strings.Builder
	edges.WriteString(fmt.Sprintf("build %s.stamp: javac_lib %s\n outdir = %s\n flags = %s\n", name, strings.Join(srcs, " "), outdir, javaflags))
	edges.WriteString(fmt.Sprintf("build %s: jar_create %s.stamp\n outdir = %s\n", out, name, outdir))
	return edges.String()
}
func (r *javaLibraryHost) Desc(m *parser.Module, srcFile string) string {
	if srcFile == "" {
		return "jar"
	}
	return "javac"
}

// ============================================================================
// java_binary_host
// ============================================================================
type javaBinaryHost struct{}

func (r *javaBinaryHost) Name() string { return "java_binary_host" }
func (r *javaBinaryHost) NinjaRule(ctx RuleRenderContext) string {
	return `rule javac_bin
  command = javac -d $outdir $in $flags

rule jar_create_executable
  command = jar cfe $out $main_class -C $outdir .
`
}
func (r *javaBinaryHost) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	return []string{fmt.Sprintf("%s-host.jar", name)}
}
func (r *javaBinaryHost) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string {
	name := getName(m)
	srcs := getSrcs(m)
	mainClass := GetStringProp(m, "main_class")
	if name == "" || len(srcs) == 0 || mainClass == "" {
		return ""
	}
	javaflags := getJavaflags(m)
	out := r.Outputs(m, ctx)[0]
	outdir := name + "_classes"
	var edges strings.Builder
	edges.WriteString(fmt.Sprintf("build %s.stamp: javac_bin %s\n outdir = %s\n flags = %s\n", name, strings.Join(srcs, " "), outdir, javaflags))
	edges.WriteString(fmt.Sprintf("build %s: jar_create_executable %s.stamp\n outdir = %s\n main_class = %s\n", out, name, outdir, mainClass))
	return edges.String()
}
func (r *javaBinaryHost) Desc(m *parser.Module, srcFile string) string {
	if srcFile == "" {
		return "jar"
	}
	return "javac"
}

// ============================================================================
// java_test
// ============================================================================
type javaTest struct{}

func (r *javaTest) Name() string { return "java_test" }
func (r *javaTest) NinjaRule(ctx RuleRenderContext) string {
	return `rule javac_test
  command = javac -d $outdir $in $flags

rule jar_test
  command = jar cf $out -C $outdir .
`
}
func (r *javaTest) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	return []string{fmt.Sprintf("%s-test.jar", name)}
}
func (r *javaTest) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string {
	name := getName(m)
	srcs := getSrcs(m)
	if name == "" || len(srcs) == 0 {
		return ""
	}
	javaflags := getJavaflags(m)
	out := r.Outputs(m, ctx)[0]
	outdir := name + "_classes"
	var edges strings.Builder
	edges.WriteString(fmt.Sprintf("build %s.stamp: javac_test %s\n outdir = %s\n flags = %s\n", name, strings.Join(srcs, " "), outdir, javaflags))
	edges.WriteString(fmt.Sprintf("build %s: jar_test %s.stamp\n outdir = %s\n", out, name, outdir))
	return edges.String()
}
func (r *javaTest) Desc(m *parser.Module, srcFile string) string {
	if srcFile == "" {
		return "jar"
	}
	return "javac"
}

// ============================================================================
// java_import
// ============================================================================
type javaImport struct{}

func (r *javaImport) Name() string { return "java_import" }
func (r *javaImport) NinjaRule(ctx RuleRenderContext) string {
	copyCmd := "cp $in $out"
	if runtime.GOOS == "windows" {
		copyCmd = "cmd /c copy $in $out"
	}
	return `rule java_import
 command = ` + copyCmd + `
`
}
func (r *javaImport) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	return []string{fmt.Sprintf("%s.jar", name)}
}
func (r *javaImport) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string {
	srcs := getSrcs(m)
	if len(srcs) == 0 {
		return ""
	}
	out := r.Outputs(m, ctx)[0]
	return fmt.Sprintf("build %s: java_import %s\n", out, strings.Join(srcs, " "))
}
func (r *javaImport) Desc(m *parser.Module, srcFile string) string { return "cp" }

// ============================================================================
// filegroup
// ============================================================================
type filegroup struct{}

func (r *filegroup) Name() string                                             { return "filegroup" }
func (r *filegroup) NinjaRule(ctx RuleRenderContext) string                   { return "" }
func (r *filegroup) Outputs(m *parser.Module, ctx RuleRenderContext) []string { return nil }
func (r *filegroup) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string { return "" }
func (r *filegroup) Desc(m *parser.Module, srcFile string) string             { return "filegroup" }

// ============================================================================
// custom
// ============================================================================
type customRule struct{}

func (r *customRule) Name() string { return "custom" }
func (r *customRule) NinjaRule(ctx RuleRenderContext) string {
	return `rule custom_command
 command = $cmd
`
}
func (r *customRule) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	return GetListProp(m, "outs")
}
func (r *customRule) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string {
	return customRuleEdge(m, "")
}

func customRuleEdge(m *parser.Module, workDir string) string {
	srcs := GetListProp(m, "srcs")
	outs := GetListProp(m, "outs")
	cmd := GetStringProp(m, "cmd")
	excludeDirs := GetListProp(m, "exclude_dirs")
	if len(outs) == 0 || cmd == "" {
		return ""
	}
	outStr := strings.Join(outs, " ")
	srcStr := strings.Join(srcs, " ")
	escapedOuts := strings.Join(escapeList(outs), " ")
	escapedSrcs := strings.Join(escapeList(srcs), " ")

	actualCmd := cmd
	actualCmd = strings.ReplaceAll(actualCmd, "$in", srcStr)
	actualCmd = strings.ReplaceAll(actualCmd, "$out", outStr)

	if len(excludeDirs) > 0 && workDir != "" {
		excluded := make(map[string]bool)
		for _, dir := range excludeDirs {
			excluded[dir] = true
		}
		var pkgList []string
		filepath.Walk(workDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || !info.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(workDir, path)
			if rel == "." || strings.HasPrefix(rel, ".") {
				return nil
			}
			parts := strings.SplitN(rel, string(filepath.Separator), 2)
			if len(parts) > 0 && excluded[parts[0]] {
				return filepath.SkipDir
			}
			files, _ := os.ReadDir(path)
			for _, f := range files {
				if !f.IsDir() && strings.HasSuffix(f.Name(), ".go") && !strings.HasPrefix(f.Name(), "_") && !strings.HasSuffix(f.Name(), "_test.go") {
					pkgList = append(pkgList, "./"+rel)
					break
				}
			}
			return nil
		})
		actualCmd = strings.ReplaceAll(actualCmd, "./...", strings.Join(pkgList, " "))
	}

	var result strings.Builder
	if srcStr == "" {
		result.WriteString(fmt.Sprintf("build %s: custom_command\n", escapedOuts))
	} else {
		result.WriteString(fmt.Sprintf("build %s: custom_command %s\n", escapedOuts, escapedSrcs))
	}
	result.WriteString(fmt.Sprintf(" cmd = %s\n", actualCmd))

	return result.String()
}
func (r *customRule) Desc(m *parser.Module, srcFile string) string { return "custom" }

// GetAllRules returns all available rule implementations
// ============================================================================
// cc_library_headers - Header library (exports headers for other modules)
// ============================================================================
type ccLibraryHeaders struct{}

func (r *ccLibraryHeaders) Name() string                           { return "cc_library_headers" }
func (r *ccLibraryHeaders) NinjaRule(ctx RuleRenderContext) string { return "" }
func (r *ccLibraryHeaders) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	return []string{name + ".h"}
}
func (r *ccLibraryHeaders) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string {
	return ""
}
func (r *ccLibraryHeaders) Desc(m *parser.Module, srcFile string) string { return "" }

// ============================================================================
// proto_library - Protocol Buffer library
// ============================================================================
type protoLibraryRule struct{}

func (r *protoLibraryRule) Name() string { return "proto_library" }
func (r *protoLibraryRule) NinjaRule(ctx RuleRenderContext) string {
	return `rule protoc
  command = protoc --proto_path=. $proto_paths $include_flags $plugin_flags --$out_type_out=$proto_out $in
  description = PROTOC $in
`
}
func (r *protoLibraryRule) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	outType := GetStringProp(m, "out")
	if outType == "" {
		outType = "cc"
	}
	srcs := getSrcs(m)
	var outs []string
	for _, src := range srcs {
		base := strings.TrimSuffix(filepath.Base(src), ".proto")
		switch outType {
		case "cc":
			outs = append(outs, base+".pb.h", base+".pb.cc")
		case "go":
			outs = append(outs, base+".pb.go")
		case "java":
			outs = append(outs, base+".java")
		case "python":
			outs = append(outs, base+"_pb2.py")
		default:
			outs = append(outs, base+".pb."+outType)
		}
	}
	return outs
}
func (r *protoLibraryRule) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string {
	name := getName(m)
	srcs := getSrcs(m)
	if name == "" || len(srcs) == 0 {
		return ""
	}
	outType := GetStringProp(m, "out")
	if outType == "" {
		outType = "cc"
	}
	protoPaths := GetListProp(m, "proto_paths")
	plugins := GetListProp(m, "plugins")
	includeDirs := GetListProp(m, "include_dirs")

	protoPathFlags := ""
	for _, p := range protoPaths {
		protoPathFlags += " --proto_path=" + p
	}

	includeFlags := ""
	for _, d := range includeDirs {
		includeFlags += " --proto_path=" + d
	}

	pluginFlags := ""
	for _, pl := range plugins {
		pluginFlags += " --plugin=" + pl
	}

	protoOut := name + "_proto_out"

	outs := r.Outputs(m, ctx)
	if len(outs) == 0 {
		return ""
	}

	return fmt.Sprintf("build %s: protoc %s\n proto_paths = %s\n include_flags = %s\n plugin_flags = %s\n out_type = %s\n proto_out = %s\n",
		strings.Join(outs, " "),
		strings.Join(srcs, " "),
		protoPathFlags,
		includeFlags,
		pluginFlags,
		outType,
		protoOut,
	)
}
func (r *protoLibraryRule) Desc(m *parser.Module, srcFile string) string { return "protoc" }

// ============================================================================
// proto_gen - Protocol Buffer code generation
// ============================================================================
type protoGenRule struct{}

func (r *protoGenRule) Name() string { return "proto_gen" }
func (r *protoGenRule) NinjaRule(ctx RuleRenderContext) string {
	return `rule protoc_gen
  command = protoc --proto_path=. $proto_paths $include_flags $plugin_flags --$out_type_out=$proto_out $in
  description = PROTOC $in
`
}
func (r *protoGenRule) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	outType := GetStringProp(m, "out")
	if outType == "" {
		outType = "cc"
	}
	srcs := getSrcs(m)
	var outs []string
	for _, src := range srcs {
		base := strings.TrimSuffix(filepath.Base(src), ".proto")
		switch outType {
		case "cc":
			outs = append(outs, name+"_"+base+".pb.h", name+"_"+base+".pb.cc")
		case "go":
			outs = append(outs, name+"_"+base+".pb.go")
		default:
			outs = append(outs, name+"_"+base+".pb."+outType)
		}
	}
	return outs
}
func (r *protoGenRule) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string {
	name := getName(m)
	srcs := getSrcs(m)
	if name == "" || len(srcs) == 0 {
		return ""
	}
	outType := GetStringProp(m, "out")
	if outType == "" {
		outType = "cc"
	}
	protoPaths := GetListProp(m, "proto_paths")
	plugins := GetListProp(m, "plugins")
	includeDirs := GetListProp(m, "include_dirs")

	protoPathFlags := ""
	for _, p := range protoPaths {
		protoPathFlags += " --proto_path=" + p
	}

	includeFlags := ""
	for _, d := range includeDirs {
		includeFlags += " --proto_path=" + d
	}

	pluginFlags := ""
	for _, pl := range plugins {
		pluginFlags += " --plugin=" + pl
	}

	protoOut := name + "_proto_out"

	outs := r.Outputs(m, ctx)
	if len(outs) == 0 {
		return ""
	}

	return fmt.Sprintf("build %s: protoc_gen %s\n proto_paths = %s\n include_flags = %s\n plugin_flags = %s\n out_type = %s\n proto_out = %s\n",
		strings.Join(outs, " "),
		strings.Join(srcs, " "),
		protoPathFlags,
		includeFlags,
		pluginFlags,
		outType,
		protoOut,
	)
}
func (r *protoGenRule) Desc(m *parser.Module, srcFile string) string { return "protoc" }

func GetAllRules() []BuildRule {
	return []BuildRule{
		&ccLibrary{}, &ccLibraryStatic{}, &ccLibraryShared{}, &ccObject{}, &ccBinary{},
		&cppLibrary{}, &cppBinary{}, &ccLibraryHeaders{},
		&goLibrary{}, &goBinary{}, &goTest{},
		&javaLibrary{}, &javaLibraryStatic{}, &javaLibraryHost{}, &javaBinary{}, &javaBinaryHost{}, &javaTest{}, &javaImport{},
		&filegroup{}, &customRule{},
		&protoLibraryRule{}, &protoGenRule{},
	}
}

// GetRule returns a rule by name
func GetRule(name string) BuildRule {
	for _, r := range GetAllRules() {
		if r.Name() == name {
			return r
		}
	}
	return nil
}

// ExpandGlob expands glob patterns
func ExpandGlob(patterns []string, exclude []string) []string {
	var result []string
	seen := make(map[string]bool)

	for _, pattern := range patterns {
		if strings.Contains(pattern, "**") {
			dir := "."
			suffix := ""
			if idx := strings.Index(pattern, "/**"); idx >= 0 {
				dir = pattern[:idx]
				suffix = pattern[idx+3:]
			}
			filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return nil
				}
				if suffix == "" || strings.HasSuffix(path, suffix) {
					if !seen[path] {
						for _, ex := range exclude {
							if matched, _ := filepath.Match(ex, path); matched {
								return nil
							}
						}
						result = append(result, path)
						seen[path] = true
					}
				}
				return nil
			})
		} else {
			matches, _ := filepath.Glob(pattern)
			for _, m := range matches {
				if !seen[m] {
					result = append(result, m)
					seen[m] = true
				}
			}
		}
	}
	return result
}
