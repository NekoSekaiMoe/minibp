// Package ninja implements Go build rules for minibp.
// This file provides compilation and linking rules for Go modules defined in Blueprint files.
// It handles the complete Go build pipeline: compiling Go sources with the go toolchain,
// and linking them into archives (.a) or executables.
//
// The Go rules support:
//   - go_library: Produces Go archive files (.a) for linking into other Go packages
//   - go_binary: Produces standalone executables for the target platform
//   - go_test: Produces test executables compiled with `go test -c`
//
// Key features:
//   - Cross-compilation via GOOS/GOARCH environment variables
//   - Multiple target variants via target { ... } properties
//   - Build flags (goflags) and linker flags (ldflags)
//   - Dependency resolution via deps property (links .a files)
//
// Build process overview:
//   1. go build -buildmode=archive compiles sources into .a archives (libraries)
//   2. go build compiles sources into standalone executables (binaries)
//   3. go test -c compiles test sources into test executables
//
// Key design decisions:
//   - Output naming: Uses "{name}{suffix}" for binaries, "{name}{suffix}.a" for libraries
//   - Variants: Cross-compilation targets specified via target { goos, goarch }
//   - Suffix format: "_{goos}_{goarch}" for variant-specific outputs
//   - Dependency linking: .a files linked via implicit dependencies (| separator in ninja)
//   - Package path: For tests, derived from first source file's directory
//
// Each Go module type implements the BuildRule interface:
//   - Name() string: Returns the module type name
//   - NinjaRule(ctx) string: Returns ninja rule definitions for go build commands
//   - Outputs(m, ctx) []string: Returns output file paths
//   - NinjaEdge(m, ctx) string: Returns ninja build edges for compilation/linking
//   - Desc(m, src) string: Returns a short description for ninja's progress output
package ninja

import (
	"fmt"
	"minibp/lib/parser"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// goLibrary implements a Go library rule.
// Go libraries produce .a archive files that can be linked into binaries.
// They can have multiple target variants for cross-compilation.
//
// Supported properties:
//   - name: The library name (used for output file name)
//   - srcs: Source files to compile
//   - goflags: Additional flags passed to the Go compiler
//   - ldflags: Linker flags injected via -ldflags
//   - target: Map of target variants with goos/goarch properties
//
// Target variants example in Blueprint:
//
//	go_library {
//	  name: "mylib",
//	  srcs: ["mylib.go"],
//	  target: {
//	    linux_amd64: {
//	      goos: "linux",
//	      goarch: "amd64",
//	    },
//	    windows_386: {
//	      goos: "windows",
//	      goarch: "386",
//	    },
//	  },
//	}
//
// Implements the BuildRule interface:
//   - Name() string: Returns "go_library"
//   - NinjaRule(ctx) string: Returns ninja rule for go build -buildmode=archive
//   - Outputs(m, ctx) []string: Returns "{name}{suffix}.a"
//   - NinjaEdge(m, ctx) string: Returns ninja build edges for compilation
//   - Desc(m, src) string: Returns "go" as description
type goLibrary struct{}

// Name returns the module type name for go_library.
// This name is used to match module types in Blueprint files (e.g., go_library { ... }).
// Go libraries produce .a archives that can be linked into Go binaries.
func (r *goLibrary) Name() string { return "go_library" }

// NinjaRule defines the ninja compilation rule for Go archives.
// Uses "go build -buildmode=archive" to produce .a files.
// Environment variables ${GOOS_GOARCH} control cross-compilation target.
//
// The rule uses env command to set GOOS/GOARCH environment variables:
//   env ${GOOS_GOARCH} go build -buildmode=archive -o $out $in
//
// Parameters:
//   - ctx: Rule render context (not used directly, but required by interface)
//
// Returns:
//   - Ninja rule definition as formatted string
//
// Edge cases:
//   - This rule doesn't use toolchain context (always uses system Go toolchain)
//   - GOOS_GOARCH variable is set per-variant in NinjaEdge
func (r *goLibrary) NinjaRule(ctx RuleRenderContext) string {
	return `rule go_build_archive
  command = env ${GOOS_GOARCH} go build -buildmode=archive -o $out $in

`
}

// Outputs returns the output paths for Go libraries.
// Returns nil if the module has no name (invalid module).
// Output format: {name}{suffix}.a
// Suffix is "_{goos}_{goarch}" when cross-compiling, empty otherwise.
//
// Parameters:
//   - m: Module being evaluated (must have "name" property)
//   - ctx: Rule render context with GOOS and GOARCH for cross-compilation
//
// Returns:
//   - List containing the Go archive output path (e.g., ["foo.a"] or ["foo_linux_amd64.a"])
//
// Edge cases:
//   - Empty name: Returns nil (cannot determine output path)
//   - No cross-compilation: Returns "{name}.a" without suffix
//   - Cross-compilation: Returns "{name}_{goos}_{goarch}.a" with context values
func (r *goLibrary) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	goos, goarch, isCrossCompile := goosAndArch(ctx)
	if !isCrossCompile {
		return []string{fmt.Sprintf("%s.a", name)}
	}
	return []string{fmt.Sprintf("%s_%s_%s.a", name, goos, goarch)}
}

// NinjaEdge generates ninja build edges for Go library compilation.
// Handles multiple target variants for cross-compilation.
//
// Build algorithm:
//  1. Get module name and source files, exit early if missing
//  2. Get target variants from "target" property
//  3. If no variants, generate single edge for host platform
//     - Uses goos/goarch from context (or runtime defaults)
//  4. If variants exist, generate one edge per variant
//     - Sort variants alphabetically for deterministic output
//  5. Each variant calls ninjaEdgeForVariant
//
// Parameters:
//   - m: Module being evaluated (must have "name", "srcs", optionally "target" properties)
//   - ctx: Rule render context with GOOS/GOARCH for default cross-compilation
//
// Returns:
//   - Ninja build edge string for compilation (may be multi-line for multiple variants)
//   - Empty string if module has no name or no source files
//
// Edge cases:
//   - Empty srcs or name: Returns "" (nothing to compile)
//   - No variants: Uses host platform (context GOOS/GOARCH or runtime defaults)
//   - Multiple variants: Generates sorted edges for deterministic output
//   - Variant with empty goos/goarch: Uses runtime.GOOS/GOARCH as defaults
func (r *goLibrary) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string {
	name := getName(m)
	srcs := getSrcs(m)
	if name == "" || len(srcs) == 0 {
		return ""
	}

	variants := getGoTargetVariants(m)
	if len(variants) == 0 {
		goos, goarch, isCrossCompile := goosAndArch(ctx)
		if !isCrossCompile {
			goos = ""
			goarch = ""
		}
		return r.ninjaEdgeForVariant(m, ctx, goos, goarch)
	}

	var edges strings.Builder
	sorted := make([]string, len(variants))
	copy(sorted, variants)
	sort.Strings(sorted)
	for _, v := range sorted {
		goos := getGoTargetProp(m, v, "goos")
		goarch := getGoTargetProp(m, v, "goarch")
		edges.WriteString(r.ninjaEdgeForVariant(m, ctx, goos, goarch))
	}
	return edges.String()
}

// ninjaEdgeForVariant generates a build edge for a specific Go target variant.
// Called once per variant or once for the host platform if no variants exist.
//
// Parameters:
//   - goos: Target operating system (e.g., "linux", "windows", "darwin")
//   - goarch: Target architecture (e.g., "amd64", "arm64", "386")
//
// Build edge format:
//
//	{name}{suffix}.a: Depends on source files
//	  flags = goflags
//	  cmd = [GOOS=X GOARCH=Y] go build -buildmode=archive [-ldflags "..."] -o $out $in
//	  GOOS_GOARCH = GOOS=X GOARCH=Y
//
// The GOOS_GOARCH variable is used by the ninja rule to set environment variables.
// The cmd variable provides the full command for display in ninja's output.
//
// Edge cases:
//   - Empty goos/goarch: No environment variables set, empty suffix
//   - Empty ldflags: Uses standard build command without -ldflags
//   - Non-empty ldflags: Injects -ldflags before -o using escapeLdflags
func (r *goLibrary) ninjaEdgeForVariant(m *parser.Module, ctx RuleRenderContext, goos, goarch string) string {
	name := getName(m)
	srcs := getSrcs(m)
	goflags := getGoflags(m)
	ldflags := getLdflags(m)

	envVar, suffix, _, _ := goVariantEnvVars(goos, goarch)
	out := fmt.Sprintf("%s%s.a", name, suffix)

	var cmd string
	cmd = goBuildCmd(ldflags, "-buildmode=archive")
	if envVar != "" {
		cmd = envVar + " " + cmd
	}

	return fmt.Sprintf("build %s: go_build_archive %s\n flags = %s\n cmd = %s\n GOOS_GOARCH = %s\n",
		out, strings.Join(srcs, " "), goflags, cmd, envVar)
}

// Desc returns a short description of the build action for ninja's progress output.
// Always returns "go" since go_library only performs Go compilation.
//
// Parameters:
//   - m: Module being evaluated (unused in this implementation)
//   - srcFile: Source file path (unused, always returns "go")
//
// Returns:
//   - "go" as the description for all Go library compilations
func (r *goLibrary) Desc(m *parser.Module, srcFile string) string {
	return "go"
}

// goBinary implements a Go binary rule.
// Go binaries are standalone executable files produced by the Go compiler.
// Unlike libraries, binaries are linked with all dependencies into a single output.
//
// Supported properties:
//   - name: The binary name (used for output file name)
//   - srcs: Source files to compile
//   - deps: List of go_library dependencies (linked as .a files)
//   - goflags: Additional flags passed to the Go compiler
//   - ldflags: Linker flags injected via -ldflags
//   - target: Map of target variants with goos/goarch properties
//
// Use cases:
//   - Command-line tools
//   - Server applications
//   - Build utilities and tools
//
// Implements the BuildRule interface:
//   - Name() string: Returns "go_binary"
//   - NinjaRule(ctx) string: Returns ninja rule for go build
//   - Outputs(m, ctx) []string: Returns "{name}{suffix}"
//   - NinjaEdge(m, ctx) string: Returns ninja build edges
//   - Desc(m, src) string: Returns "go" as description
type goBinary struct{}

// Name returns the module type name for go_binary.
// This name is used to match module types in Blueprint files (e.g., go_binary { ... }).
// Go binaries are standalone executables that can be run directly.
func (r *goBinary) Name() string { return "go_binary" }

// NinjaRule defines the ninja linking rule for Go binaries.
// Uses "go build" without -buildmode to produce standalone executables.
// Environment variables ${GOOS_GOARCH} control cross-compilation target.
//
// The rule uses env command to set GOOS/GOARCH environment variables:
//   env ${GOOS_GOARCH} go build -o $out $in
//
// Parameters:
//   - ctx: Rule render context (not used directly, but required by interface)
//
// Returns:
//   - Ninja rule definition as formatted string
//
// Edge cases:
//   - This rule doesn't use toolchain context (always uses system Go toolchain)
//   - GOOS_GOARCH variable is set per-variant in NinjaEdge
func (r *goBinary) NinjaRule(ctx RuleRenderContext) string {
	return `rule go_build
  command = env ${GOOS_GOARCH} go build -o $out $in

`
}

// Outputs returns the output paths for Go binaries.
// Returns nil if the module has no name (invalid module).
// Output format: {name}{suffix}
// No file extension since Go binaries are platform-specific executables.
// Suffix is "_{goos}_{goarch}" when cross-compiling, empty otherwise.
//
// Parameters:
//   - m: Module being evaluated (must have "name" property)
//   - ctx: Rule render context with GOOS and GOARCH for cross-compilation
//
// Returns:
//   - List containing the Go binary output path (e.g., ["foo"] or ["foo_linux_amd64"])
//
// Edge cases:
//   - Empty name: Returns nil (cannot determine output path)
//   - No cross-compilation: Returns "{name}" without suffix
//   - Cross-compilation: Returns "{name}_{goos}_{goarch}" with context values
func (r *goBinary) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	goos, goarch, isCrossCompile := goosAndArch(ctx)
	if !isCrossCompile {
		return []string{name}
	}
	return []string{fmt.Sprintf("%s_%s_%s", name, goos, goarch)}
}

// NinjaEdge generates ninja build edges for Go binary compilation and linking.
// Handles multiple target variants for cross-compilation.
//
// Build algorithm:
//  1. Get module name and source files, exit early if missing
//  2. Get target variants from "target" property
//  3. If no variants, generate single edge for host platform
//     - Uses goos/goarch from context (or runtime defaults)
//  4. If variants exist, generate one edge per variant
//     - Sort variants alphabetically for deterministic output
//  5. Each variant calls ninjaEdgeForVariant
//  6. Dependencies (.a files) are linked as implicit inputs
//
// Parameters:
//   - m: Module being evaluated (must have "name", "srcs", optionally "deps" and "target")
//   - ctx: Rule render context with GOOS/GOARCH for default cross-compilation
//
// Returns:
//   - Ninja build edge string for compilation and linking (may be multi-line)
//   - Empty string if module has no name or no source files
//
// Edge cases:
//   - Empty srcs or name: Returns "" (nothing to compile)
//   - No variants: Uses host platform (context GOOS/GOARCH or runtime defaults)
//   - No deps: Generates edge without implicit dependencies
//   - Multiple variants: Generates sorted edges for deterministic output
//   - Variant with empty goos/goarch: Uses runtime.GOOS/GOARCH as defaults
func (r *goBinary) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string {
	name := getName(m)
	srcs := getSrcs(m)
	if name == "" || len(srcs) == 0 {
		return ""
	}

	variants := getGoTargetVariants(m)
	if len(variants) == 0 {
		goos, goarch, isCrossCompile := goosAndArch(ctx)
		if !isCrossCompile {
			goos = ""
			goarch = ""
		}
		return r.ninjaEdgeForVariant(m, ctx, goos, goarch)
	}

	var edges strings.Builder
	sorted := make([]string, len(variants))
	copy(sorted, variants)
	sort.Strings(sorted)
	for _, v := range sorted {
		goos := getGoTargetProp(m, v, "goos")
		goarch := getGoTargetProp(m, v, "goarch")
		edges.WriteString(r.ninjaEdgeForVariant(m, ctx, goos, goarch))
	}
	return edges.String()
}

// ninjaEdgeForVariant generates a build edge for a specific Go binary variant.
//
// Dependencies are resolved by:
//  1. Stripping ":" prefix from dep names (module reference syntax)
//  2. Appending ".a" extension to get archive file names
//  3. Adding as implicit dependencies (|) so ninja tracks them
//  4. Using ctx.PathPrefix if set (for namespace support)
//
// Build edge format with deps:
//
//	{name}{suffix}: Depends on source files | lib1.a lib2.a ...
//	  flags = goflags
//	  cmd = [GOOS=X GOARCH=Y] go build [-ldflags "..."] -o $out $in
//	  GOOS_GOARCH = GOOS=X GOARCH=Y
//
// Build edge format without deps:
//
//	{name}{suffix}: Depends on source files
//	  flags = goflags
//	  cmd = [GOOS=X GOARCH=Y] go build [-ldflags "..."] -o $out $in
//	  GOOS_GOARCH = GOOS=X GOARCH=Y
//
// Parameters:
//   - goos: Target operating system (e.g., "linux", "windows")
//   - goarch: Target architecture (e.g., "amd64", "arm64")
//
// Edge cases:
//   - Empty goos/goarch: No environment variables set, no suffix
//   - Empty ldflags: Uses standard build command without -ldflags
//   - Non-empty ldflags: Injects -ldflags using escapeLdflags
//   - Empty deps: No implicit dependencies (no | separator)
func (r *goBinary) ninjaEdgeForVariant(m *parser.Module, ctx RuleRenderContext, goos, goarch string) string {
	name := getName(m)
	srcs := getSrcs(m)
	deps := GetListProp(m, "deps")
	goflags := getGoflags(m)
	ldflags := getLdflags(m)

	envVar, suffix, _, _ := goVariantEnvVars(goos, goarch)
	out := name + suffix

	var libFiles []string
	for _, dep := range deps {
		depName := strings.TrimPrefix(dep, ":")
		libFiles = append(libFiles, ctx.PathPrefix+depName+suffix+".a")
	}

	srcStr := strings.Join(srcs, " ")

	var cmd string
	cmd = goBuildCmd(ldflags, "")
	if envVar != "" {
		cmd = envVar + " " + cmd
	}

	// Link dependencies as implicit inputs using | separator.
	// This tells ninja to track dependencies but not to rebuild when they change.
	if len(libFiles) > 0 {
		libStr := strings.Join(libFiles, " ")
		return fmt.Sprintf("build %s: go_build %s | %s\n flags = %s\n cmd = %s\n GOOS_GOARCH = %s\n",
			out, srcStr, libStr, goflags, cmd, envVar)
	}

	return fmt.Sprintf("build %s: go_build %s\n flags = %s\n cmd = %s\n GOOS_GOARCH = %s\n",
		out, srcStr, goflags, cmd, envVar)
}

// Desc returns a short description of the build action for ninja's progress output.
// Always returns "go" since go_binary only performs Go compilation/linking.
//
// Parameters:
//   - m: Module being evaluated (unused in this implementation)
//   - srcFile: Source file path (unused, always returns "go")
//
// Returns:
//   - "go" as the description for all Go binary builds
func (r *goBinary) Desc(m *parser.Module, srcFile string) string {
	return "go"
}

// goTest implements a Go test rule.
// Go test binaries are compiled test executables produced by `go test -c`.
// Test files are identified by the _test.go suffix convention.
//
// Supported properties:
//   - name: The test binary name (used for output file name)
//   - srcs: Source files to compile (including _test.go files)
//   - goflags: Additional flags passed to `go test`
//   - ldflags: Linker flags injected via -ldflags
//   - target: Map of target variants with goos/goarch properties
//
// Unlike goBinary, tests use `go test -c` which:
//   - Automatically includes test dependencies
//   - Compiles test files (*_test.go)
//   - Produces a standalone test executable
//
// Implements the BuildRule interface:
//   - Name() string: Returns "go_test"
//   - NinjaRule(ctx) string: Returns ninja rule for go test -c
//   - Outputs(m, ctx) []string: Returns "{name}{suffix}.test"
//   - NinjaEdge(m, ctx) string: Returns ninja build edges
//   - Desc(m, src) string: Returns "go test" as description
type goTest struct{}

// Name returns the module type name for go_test.
// This name is used to match module types in Blueprint files (e.g., go_test { ... }).
// Go test binaries are compiled with `go test -c` and include test frameworks.
func (r *goTest) Name() string { return "go_test" }

// NinjaRule defines the ninja test compilation rule.
// Uses `go test -c` to compile test executables.
// Environment variables ${GOOS_GOARCH} control cross-compilation target.
//
// The rule uses env command to set GOOS/GOARCH environment variables:
//   env ${GOOS_GOARCH} go test -c -o $out $pkg
//
// Note: Unlike go build, go test -c takes a package path ($pkg) not source files.
//
// Parameters:
//   - ctx: Rule render context (not used directly, but required by interface)
//
// Returns:
//   - Ninja rule definition as formatted string
//
// Edge cases:
//   - This rule doesn't use toolchain context (always uses system Go toolchain)
//   - GOOS_GOARCH variable is set per-variant in NinjaEdge
func (r *goTest) NinjaRule(ctx RuleRenderContext) string {
	return `rule go_test
  command = env ${GOOS_GOARCH} go test -c -o $out $pkg

`
}

// Outputs returns the output paths for Go test binaries.
// Returns nil if the module has no name (invalid module).
// Output format: {name}{suffix}.test
// The ".test" extension identifies test executables.
// Suffix is "_{goos}_{goarch}" when cross-compiling, empty otherwise.
//
// Parameters:
//   - m: Module being evaluated (must have "name" property)
//   - ctx: Rule render context with GOOS and GOARCH for cross-compilation
//
// Returns:
//   - List containing the Go test binary output path (e.g., ["foo.test"] or ["foo_linux_amd64.test"])
//
// Edge cases:
//   - Empty name: Returns nil (cannot determine output path)
//   - No cross-compilation: Returns "{name}.test" without suffix
//   - Cross-compilation: Returns "{name}_{goos}_{goarch}.test" with context values
func (r *goTest) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	goos, goarch, isCrossCompile := goosAndArch(ctx)
	if !isCrossCompile {
		return []string{fmt.Sprintf("%s.test", name)}
	}
	return []string{fmt.Sprintf("%s_%s_%s.test", name, goos, goarch)}
}

// NinjaEdge generates ninja build edges for Go test compilation.
// Handles multiple target variants for cross-compilation.
//
// Build algorithm:
//  1. Get module name and source files, exit early if missing
//  2. Get target variants from "target" property
//  3. If no variants, generate single edge for host platform
//     - Uses goos/goarch from context (or runtime defaults)
//  4. If variants exist, generate one edge per variant
//     - Sort variants alphabetically for deterministic output
//  5. Each variant calls ninjaEdgeForVariant
//
// Note: Unlike goBinary, tests use pkg parameter (directory path) instead of
// individual source files, since `go test -c` expects a package path.
//
// Parameters:
//   - m: Module being evaluated (must have "name", "srcs", optionally "target" properties)
//   - ctx: Rule render context with GOOS/GOARCH for default cross-compilation
//
// Returns:
//   - Ninja build edge string for test compilation (may be multi-line for multiple variants)
//   - Empty string if module has no name or no source files
//
// Edge cases:
//   - Empty srcs or name: Returns "" (nothing to compile)
//   - No variants: Uses host platform (context GOOS/GOARCH or runtime defaults)
//   - Multiple variants: Generates sorted edges for deterministic output
//   - Variant with empty goos/goarch: Uses runtime.GOOS/GOARCH as defaults
func (r *goTest) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string {
	name := getName(m)
	srcs := getSrcs(m)
	if name == "" || len(srcs) == 0 {
		return ""
	}

	variants := getGoTargetVariants(m)
	if len(variants) == 0 {
		goos, goarch, isCrossCompile := goosAndArch(ctx)
		if !isCrossCompile {
			goos = ""
			goarch = ""
		}
		return r.ninjaEdgeForVariant(m, ctx, goos, goarch)
	}

	var edges strings.Builder
	sorted := make([]string, len(variants))
	copy(sorted, variants)
	sort.Strings(sorted)
	for _, v := range sorted {
		goos := getGoTargetProp(m, v, "goos")
		goarch := getGoTargetProp(m, v, "goarch")
		edges.WriteString(r.ninjaEdgeForVariant(m, ctx, goos, goarch))
	}
	return edges.String()
}

// ninjaEdgeForVariant generates a build edge for a specific Go test variant.
//
// The package path is derived from the first source file's directory:
//  1. Get the first source file from srcs
//  2. Extract its directory using filepath.Dir
//  3. Prepend "./" to get a relative package path
//
// Example: srcs[0] = "foo/bar_test.go"
//
//	pkgPath = "./" + filepath.Dir("foo/bar_test.go") = "./foo"
//
// Build edge format:
//
//	{name}{suffix}.test: go_test
//	  pkg = ./package_directory
//	  flags = goflags
//	  cmd = [GOOS=X GOARCH=Y] go test [-ldflags "..."] -c -o $out $pkg
//	  GOOS_GOARCH = GOOS=X GOARCH=Y
//
// Note: Unlike go build, go test -c takes a package path not source files.
//
// Parameters:
//   - goos: Target operating system (e.g., "linux", "windows")
//   - goarch: Target architecture (e.g., "amd64", "arm64")
//
// Edge cases:
//   - Empty goos/goarch: No environment variables set, no suffix
//   - Empty ldflags: Uses standard go test -c command without -ldflags
//   - Non-empty ldflags: Injects -ldflags using escapeLdflags
//   - First src must exist: Uses srcs[0] to derive package directory
func (r *goTest) ninjaEdgeForVariant(m *parser.Module, ctx RuleRenderContext, goos, goarch string) string {
	name := getName(m)
	srcs := getSrcs(m)
	goflags := getGoflags(m)
	ldflags := getLdflags(m)
	pkgPath := "./" + filepath.Dir(srcs[0])

	envVar, suffix, _, _ := goVariantEnvVars(goos, goarch)
	out := fmt.Sprintf("%s%s.test", name, suffix)

	var cmd string
	cmd = goTestCmd(ldflags)
	if envVar != "" {
		cmd = envVar + " " + cmd
	}

	return fmt.Sprintf("build %s: go_test\n pkg = %s\n flags = %s\n cmd = %s\n GOOS_GOARCH = %s\n",
		out, pkgPath, goflags, cmd, envVar)
}

// Desc returns a short description of the build action for ninja's progress output.
// Always returns "go test" since go_test performs Go test compilation.
//
// Parameters:
//   - m: Module being evaluated (unused in this implementation)
//   - srcFile: Source file path (unused, always returns "go test")
//
// Returns:
//   - "go test" as the description for all Go test compilations
func (r *goTest) Desc(m *parser.Module, srcFile string) string {
	return "go test"
}

// escapeLdflags escapes special characters in ldflags for use in ninja build files.
// Ninja uses $ for variables, so $ must be esaped as $$.
// Other characters that need escaping:
//   - Backslash (\): Escaped as \\ to prevent interpretation as escape character
//   - Double quote ("): Escaped as \" to preserve quotes in the string
//   - Backtick (`): Escaped as \` for shell compatibility
//   - Semicolon (;): Escaped as \; to prevent command separation
//
// Parameters:
//   - ldflags: The ldflags string to escape
//
// Returns:
//   - The escaped string safe for use in ninja variable assignments
//
// Edge cases:
//   - Empty string: Returns empty string unchanged
//   - String without special characters: Returns original string unchanged
func escapeLdflags(ldflags string) string {
	ldflags = strings.ReplaceAll(ldflags, `\`, `\\`)
	ldflags = strings.ReplaceAll(ldflags, `"`, `\"`)
	ldflags = strings.ReplaceAll(ldflags, "$", `\$`)
	ldflags = strings.ReplaceAll(ldflags, "`", "\\`")
	ldflags = strings.ReplaceAll(ldflags, ";", `\;`)
	return ldflags
}

// goBuildCmd constructs the go build command with optional build mode and linker flags.
// It's used for building both libraries (-buildmode=archive) and binaries (no build mode).
//
// Parameters:
//   - ldflags: Linker flags to inject via -ldflags (may be empty)
//   - buildMode: Build mode flag (e.g., "-buildmode=archive" for libraries, "" for binaries)
//
// Returns:
//   - The complete go build command string
//
// Edge cases:
//   - Empty ldflags: Returns command without -ldflags
//   - Empty buildMode: Returns "go build -o $out $in" (for binaries)
//   - Non-empty ldflags: Escapes special characters via escapeLdflags
//
// Example outputs:
//   - goBuildCmd("", "") -> "go build -o $out $in"
//   - goBuildCmd("", "-buildmode=archive") -> "go build -buildmode=archive -o $out $in"
//   - goBuildCmd("-s -w", "") -> "go build -ldflags \"-s -w\" -o $out $in"
func goBuildCmd(ldflags string, buildMode string) string {
	if ldflags != "" {
		return fmt.Sprintf("go build %s -ldflags \"%s\" -o $out $in", buildMode, escapeLdflags(ldflags))
	}
	return fmt.Sprintf("go build %s -o $out $in", buildMode)
}

// goTestCmd constructs the go test command with optional linker flags.
// It's used for compiling test executables with `go test -c`.
//
// Parameters:
//   - ldflags: Linker flags to inject via -ldflags (may be empty)
//
// Returns:
//   - The complete go test command string
//
// Edge cases:
//   - Empty ldflags: Returns "go test -c -o $out $pkg" (standard test build)
//   - Non-empty ldflags: Escapes special characters via escapeLdflags
//
// Example outputs:
//   - goTestCmd("") -> "go test -c -o $out $pkg"
//   - goTestCmd("-s -w") -> "go test -ldflags \"-s -w\" -c -o $out $pkg"
func goTestCmd(ldflags string) string {
	if ldflags != "" {
		return fmt.Sprintf("go test -ldflags \"%s\" -c -o $out $pkg", escapeLdflags(ldflags))
	}
	return "go test -c -o $out $pkg"
}
// goosAndArch returns the GOOS and GOARCH values from context, with defaults from runtime.
// It also returns whether this is a cross-compilation scenario.
// If goos/goarch are different from runtime, they're considered cross-compilation.
//
// The returned (goos, goarch) values are normalized:
//   - Empty goarch defaults to runtime.GOARCH
//   - Empty goos defaults to runtime.GOOS
//
// The isCrossCompile return value indicates whether the target differs
// from the host platform (for output suffix generation).
//
// Parameters:
//   - ctx: Rule render context containing GOOS and GOARCH values
//
// Returns:
//   - goos: The target operating system (normalized with default)
//   - goarch: The target architecture (normalized with default)
//   - isCrossCompile: true if target differs from host platform
//
// Edge cases:
//   - Empty GOOS/GOARCH in context: Uses runtime.GOOS/GOARCH as defaults
//   - Same as runtime: isCrossCompile returns false (native build)
//   - Different from runtime: isCrossCompile returns true (cross-compilation)
func goosAndArch(ctx RuleRenderContext) (goos, goarch string, isCrossCompile bool) {
	goos = ctx.GOOS
	goarch = ctx.GOARCH
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	if goos == "" {
		goos = runtime.GOOS
	}
	isCrossCompile = goarch != runtime.GOARCH || goos != runtime.GOOS
	return
}

// goVariantEnvVars builds the environment variable string and suffix for Go targets.
// It accepts goos/goarch (which may be empty strings) and returns:
//   - envVar: The GOOS/GOARCH environment variable string (e.g., "GOOS=linux GOARCH=amd64")
//   - suffix: The output suffix (e.g., "_linux_amd64", or "" if no cross-compilation)
//   - normGoos, normGoarch: goos/goarch with defaults filled in
//
// The envVar is used by the ninja rule to set environment variables for go build.
// The suffix is appended to output file names for variant identification.
// The normalized values are used for consistent suffix generation.
//
// Parameters:
//   - goos: Target operating system (may be empty)
//   - goarch: Target architecture (may be empty)
//
// Returns:
//   - envVar: Environment variable string for GOOS/GOARCH (empty if no cross-compilation)
//   - suffix: Output file suffix (empty if no cross-compilation)
//   - normGoos: Normalized GOOS (with default filled in)
//   - normGoarch: Normalized GOARCH (with default filled in)
//
// Edge cases:
//   - Both empty: Returns all empty strings (native build)
//   - Only goos set: Returns only GOOS= env var
//   - Only goarch set: Returns only GOARCH= env var
//   - Both set: Returns combined "GOOS=X GOARCH=Y" env var
//   - Empty values default to runtime.GOOS/GOARCH for normalization
func goVariantEnvVars(goos, goarch string) (envVar string, suffix string, normGoos, normGoarch string) {
	normGoos = goos
	normGoarch = goarch
	if normGoarch == "" {
		normGoarch = runtime.GOARCH
	}
	if normGoos == "" {
		normGoos = runtime.GOOS
	}
	if goos != "" || goarch != "" {
		parts := []string{}
		if goos != "" {
			parts = append(parts, "GOOS="+goos)
		}
		if goarch != "" {
			parts = append(parts, "GOARCH="+goarch)
		}
		envVar = strings.Join(parts, " ")
		suffix = "_" + normGoos + "_" + normGoarch
	}
	return
}
