// ninja/go.go - Go build rules for minibp
// This file implements the BuildRule interface for Go language modules.
// It provides rules for building Go libraries, binaries, and test executables.
//
// The Go rules support:
//   - go_library: Produces Go archive files (.a)
//   - go_binary: Produces standalone executables
//   - go_test: Produces test executables
//
// Key features:
//   - Cross-compilation via GOOS/GOARCH environment variables
//   - Multiple target variants via target { ... } properties
//   - Build flags (goflags) and linker flags (ldflags)
//   - Dependency resolution via deps property
//
// Key design decisions:
//   - Output naming: Uses "{name}{suffix}" for binaries, "{name}{suffix}.a" for libraries
//   - Variants: Cross-compilation targets specified via target { goos, goarch }
//   - Suffix format: "_{goos}_{goarch}" for variant-specific outputs
//   - Dependency linking: .a files linked via implicit dependencies
//
// Each Go module type implements the BuildRule interface:
//   - Name() string: Returns the module type name
//   - NinjaRule(ctx) string: Returns ninja rule definitions
//   - Outputs(m, ctx) []string: Returns output file paths
//   - NinjaEdge(m, ctx) string: Returns ninja build edges
//   - Desc(m, src) string: Returns a short description
//
// This file provides Go compilation and linking rules for the Ninja build system.
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
// Supported properties:
//   - name: The library name (used for output file name)
//   - srcs: Source files to compile
//   - goflags: Additional flags passed to the Go compiler
//   - ldflags: Linker flags injected via -ldflags
//   - target: Map of target variants with goos/goarch properties
type goLibrary struct{}

func (r *goLibrary) Name() string { return "go_library" }

// NinjaRule defines the ninja compilation rule for Go archives.
// Uses go build with -buildmode=archive to produce .a files.
// Environment variables ${GOOS_GOARCH} control cross-compilation target.
func (r *goLibrary) NinjaRule(ctx RuleRenderContext) string {
	return `rule go_build_archive
  command = env ${GOOS_GOARCH} go build -buildmode=archive -o $out $in

`
}

// Outputs returns the output paths for Go libraries.
// Returns nil if the module has no name (invalid module).
// Output format: {name}{suffix}.a
// Suffix is "_goos_goarch" when cross-compiling, empty otherwise.
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
//  1. Get target variants from "target" property
//  2. If no variants, generate single edge for host platform
//  3. If variants exist, generate one edge per variant
//  4. Sort variants alphabetically for deterministic output
//
// Edge cases:
//   - Empty srcs: Returns "" (nothing to compile)
//   - Missing name: Returns "" (cannot determine output path)
//   - No variants: Uses ninjaEdgeForVariant with empty strings
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
// Edge cases:
//   - Empty goos/goarch: No environment variables set, empty suffix
//   - Empty ldflags: Uses standard build command without -ldflags
//   - Non-empty ldflags: Injects -ldflags before -o
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

// Desc returns a short description of the build action.
func (r *goLibrary) Desc(m *parser.Module, srcFile string) string {
	return "go"
}

// goBinary implements a Go binary rule.
// Go binaries are standalone executable files produced by the Go compiler.
// Unlike libraries, binaries are linked with all dependencies into a single output.
// Supported properties:
//   - name: The binary name (used for output file name)
//   - srcs: Source files to compile
//   - deps: List of go_library dependencies (linked as .a files)
//   - goflags: Additional flags passed to the Go compiler
//   - ldflags: Linker flags injected via -ldflags
//   - target: Map of target variants with goos/goarch properties
type goBinary struct{}

func (r *goBinary) Name() string { return "go_binary" }

// NinjaRule defines the ninja linking rule for Go binaries.
// Uses go build without -buildmode to produce standalone executables.
// Environment variables ${GOOS_GOARCH} control cross-compilation target.
func (r *goBinary) NinjaRule(ctx RuleRenderContext) string {
	return `rule go_build
  command = env ${GOOS_GOARCH} go build -o $out $in

`
}

// Outputs returns the output paths for Go binaries.
// Returns nil if the module has no name (invalid module).
// Output format: {name}{suffix}
// No extension since Go binaries are platform-specific executables.
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

// NinjaEdge generates ninja build edges for Go binary compilation.
// Handles multiple target variants for cross-compilation.
//
// Build algorithm:
//  1. Get target variants from "target" property
//  2. If no variants, generate single edge for host platform
//  3. If variants exist, generate one edge per variant
//  4. Dependencies (.a files) are linked as implicit inputs
//
// Edge cases:
//   - Empty srcs: Returns "" (nothing to compile)
//   - Missing name: Returns "" (cannot determine output path)
//   - No deps: Generates edge without implicit dependencies
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
//
// Build edge format with deps:
//
//	{name}{suffix}: Depends on source files | lib1.a lib2.a ...
//	  flags = goflags
//	  cmd = [GOOS=X GOARCH=Y] go build [-ldflags "..."] -o $out $in
//
// Build edge format without deps:
//
//	{name}{suffix}: Depends on source files
//	  flags = goflags
//	  cmd = [GOOS=X GOARCH=Y] go build [-ldflags "..."] -o $out $in
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

// Desc returns a short description of the build action.
func (r *goBinary) Desc(m *parser.Module, srcFile string) string {
	return "go"
}

// goTest implements a Go test rule.
// Go test binaries are compiled test executables produced by `go test -c`.
// Test files are identified by the _test.go suffix convention.
// Supported properties:
//   - name: The test binary name (used for output file name)
//   - srcs: Source files to compile (including _test.go files)
//   - goflags: Additional flags passed to `go test`
//   - ldflags: Linker flags injected via -ldflags
//   - target: Map of target variants with goos/goarch properties
type goTest struct{}

func (r *goTest) Name() string { return "go_test" }

// NinjaRule defines the ninja test compilation rule.
// Uses `go test -c` to compile test executables.
// Environment variables ${GOOS_GOARCH} control cross-compilation target.
func (r *goTest) NinjaRule(ctx RuleRenderContext) string {
	return `rule go_test
  command = env ${GOOS_GOARCH} go test -c -o $out $pkg

`
}

// Outputs returns the output paths for Go test binaries.
// Returns nil if the module has no name (invalid module).
// Output format: {name}{suffix}.test
// The ".test" extension identifies test executables.
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
//  1. Get target variants from "target" property
//  2. If no variants, generate single edge for host platform
//  3. If variants exist, generate one edge per variant
//
// Note: Unlike goBinary, tests use pkg parameter (directory path) instead of
// individual source files, since `go test -c` expects a package path.
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

// Desc returns a short description of the build action.
func (r *goTest) Desc(m *parser.Module, srcFile string) string {
	return "go test"
}

func escapeLdflags(ldflags string) string {
	ldflags = strings.ReplaceAll(ldflags, `\`, `\\`)
	ldflags = strings.ReplaceAll(ldflags, `"`, `\"`)
	ldflags = strings.ReplaceAll(ldflags, "$", `\$`)
	ldflags = strings.ReplaceAll(ldflags, "`", "\\`")
	ldflags = strings.ReplaceAll(ldflags, ";", `\;`)
	return ldflags
}

func goBuildCmd(ldflags string, buildMode string) string {
	if ldflags != "" {
		return fmt.Sprintf("go build %s -ldflags \"%s\" -o $out $in", buildMode, escapeLdflags(ldflags))
	}
	return fmt.Sprintf("go build %s -o $out $in", buildMode)
}

func goTestCmd(ldflags string) string {
	if ldflags != "" {
		return fmt.Sprintf("go test -ldflags \"%s\" -c -o $out $pkg", escapeLdflags(ldflags))
	}
	return "go test -c -o $out $pkg"
}
// It also returns whether this is a cross-compilation scenario.
// If goos/goarch are different from runtime, they're considered cross-compilation.
//
// The returned (goos, goarch) values are normalized:
//   - Empty goarch defaults to runtime.GOARCH
//   - Empty goos defaults to runtime.GOOS
//
// The isCrossCompile return value indicates whether the target differs
// from the host platform (for output suffix generation).
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
