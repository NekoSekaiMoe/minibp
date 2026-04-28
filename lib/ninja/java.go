// Package ninja implements Java build rules for minibp.
// This file provides compilation and packaging rules for Java modules defined in Blueprint files.
// It handles the complete Java build pipeline: compiling .java sources with javac,
// then packaging .class files into .jar archives using the jar command.
//
// The Java rules support:
//   - java_library: Produces .jar files from Java sources for use as libraries
//   - java_binary: Produces executable .jar files with main class manifest
//   - java_library_static: Produces static .a.jar files for precompiled distributions
//   - java_library_host: Produces host-specific .jar files for build tools
//   - java_binary_host: Produces host-specific executable .jar files
//   - java_test: Produces test .jar files with test framework support
//   - java_import: Imports pre-built .jar files without recompilation
//
// Build process overview:
//  1. javac compiles .java files to .class files in a staging directory ({name}_classes)
//  2. jar packages the .class files into a .jar archive
//  3. For executables, a manifest with Main-Class is created
//
// Key design decisions:
//   - Output naming: Uses "{name}.jar", "lib{name}.a.jar", "{name}-host.jar", "{name}-test.jar"
//   - Staging directory: Each module uses "{name}_classes" to isolate .class files
//   - Stamp files: Intermediate .stamp files track successful javac compilation
//   - Host variants: Use "-host" suffix to distinguish from device variants
//   - Executable JARs: Use "jar cfe" or manifest to embed main class
//   - Cross-platform: Uses "cp" on Unix or "cmd /c copy" on Windows for java_import
//
// Each Java module type implements the BuildRule interface:
//   - Name() string: Returns the module type name
//   - NinjaRule(ctx) string: Returns ninja rule definitions for javac and jar
//   - Outputs(m, ctx) []string: Returns output file paths
//   - NinjaEdge(m, ctx) string: Returns ninja build edges for compilation and packaging
//   - Desc(m, src) string: Returns a short description for ninja's progress output
package ninja

import (
	"fmt"
	"minibp/lib/parser"
	"runtime"
	"strings"
)

// sanitizeManifestValue sanitizes a string value for use in a Java manifest file.
// It replaces newline and carriage return characters with spaces to prevent
// manifest file corruption. Manifest files use newlines as record separators,
// so embedded newlines in values would break the format.
//
// Parameters:
//   - s: The string to sanitize (typically a main class name or class path)
//
// Returns:
//   - The sanitized string with newlines replaced by spaces
//
// Edge cases:
//   - Empty string: Returns empty string unchanged
//   - String without newlines: Returns original string unchanged
func sanitizeManifestValue(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' {
			return ' '
		}
		return r
	}, s)
}

// sanitizeOutdir sanitizes a module name for use as an output directory name.
// It prevents directory traversal attacks and ensures the output directory name
// is a simple identifier without path separators.
//
// If the name contains path separators (/) or backslashes (\) or parent directory
// references (..), it replaces those characters with underscores to create a safe name.
// This prevents accidental or malicious creation of nested directories.
//
// Parameters:
//   - name: The module name to sanitize for use as a directory name
//
// Returns:
//   - The sanitized directory name safe for use as a filesystem path component
//
// Edge cases:
//   - Clean name (no path characters): Returns original name unchanged
//   - Name with path characters: Replaces / and \ with _ to flatten the path
//   - Name with "..": Also gets sanitized since / is replaced
func sanitizeOutdir(name string) string {
	if strings.Contains(name, "/") || strings.Contains(name, "\\") ||
		strings.Contains(name, "..") {
		return strings.Map(func(r rune) rune {
			if r == '/' || r == '\\' {
				return '_'
			}
			return r
		}, name)
	}
	return name
}

// javaLibrary implements a Java library build rule.
// Java libraries are built by compiling Java source files and packaging them into .jar archives.
// The build process:
//   - javac compiles .java source files to .class files in a staging directory
//   - jar packages the .class files into a .jar archive
//
// This rule produces standard Java library JARs (e.g., name.jar) used as dependencies
// by other Java modules or binaries.
//
// Implements the BuildRule interface:
//   - Name() string: Returns "java_library"
//   - NinjaRule(ctx) string: Returns ninja compilation and packaging rules
//   - Outputs(m, ctx) []string: Returns the .jar output path
//   - NinjaEdge(m, ctx) string: Returns ninja build edges for compilation and packaging
//   - Desc(m, src) string: Returns "jar" for packaging, "javac" for compilation
type javaLibrary struct{}

// Name returns the module type name for java_library.
// This name is used to match module types in Blueprint files (e.g., java_library { ... }).
// Java libraries produce .jar files that can be used as dependencies by other Java modules.
func (r *javaLibrary) Name() string { return "java_library" }

// NinjaRule defines the ninja compilation and archiving rules for Java libraries.
// Creates two rules:
//   - javac_lib: Compiles Java sources to .class files in the specified outdir
//   - Uses -d flag to specify output directory for .class files
//   - jar_create: Packages .class files from outdir into a .jar archive
//   - Uses -C flag to change to outdir before adding files
//   - The "." adds all files from the outdir to the JAR
//
// Parameters:
//   - ctx: Rule render context (not used directly, but required by interface)
//
// Returns:
//   - Ninja rule definitions as formatted string
//
// Edge cases:
//   - This rule doesn't use toolchain context (always uses system javac and jar)
func (r *javaLibrary) NinjaRule(ctx RuleRenderContext) string {

	return `rule javac_lib

  command = javac -d $outdir $in $flags

rule jar_create

  command = jar cf $out -C $outdir .

`

}

// Outputs returns the output paths for Java libraries.
// Returns nil if the module has no name (invalid module).
// Output format: {name}.jar
//
// Parameters:
//   - m: Module being evaluated (must have "name" property)
//   - ctx: Rule render context (unused for Java libraries)
//
// Returns:
//   - List containing the JAR output path (e.g., ["foo.jar"])
//
// Edge cases:
//   - Empty name: Returns nil (cannot determine output path)
//   - No architecture suffix for Java (Java is platform-independent at bytecode level)
func (r *javaLibrary) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	return []string{fmt.Sprintf("%s.jar", name)}
}

// NinjaEdge generates ninja build edges for compiling and packaging Java sources.
// Returns empty string if name is empty or no sources are provided (invalid module).
//
// Build edges generated:
//  1. {name}.stamp: Depends on source files, compiles with javac to staging directory
//     - outdir variable points to {name}_classes staging directory
//     - flags variable contains javaflags from module
//  2. {name}.jar: Depends on stamp file, packages .class files with jar
//     - outdir variable reused for jar command
//
// The .stamp file is an empty marker file that represents successful compilation.
// Ninja uses it to track whether recompilation is needed.
//
// Parameters:
//   - m: Module being evaluated (must have "name" and "srcs" properties)
//   - ctx: Rule render context (unused for Java libraries)
//
// Returns:
//   - Ninja build edge string for compilation and packaging
//   - Empty string if module has no name or no source files
//
// Edge cases:
//   - Empty srcs: Returns "" (no compilation needed)
//   - Missing name: Returns "" (cannot determine output path)
//   - Special characters in name: Sanitized via sanitizeOutdir for outdir
func (r *javaLibrary) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string {
	name := getName(m)
	srcs := getSrcs(m)
	if name == "" || len(srcs) == 0 {
		return ""
	}

	javaflags := getJavaflags(m)
	out := r.Outputs(m, ctx)[0]
	outdir := sanitizeOutdir(name) + "_classes"

	var edges strings.Builder
	edges.WriteString(fmt.Sprintf("build %s.stamp: javac_lib %s\n outdir = %s\n flags = %s\n", name, strings.Join(srcs, " "), outdir, javaflags))
	edges.WriteString(fmt.Sprintf("build %s: jar_create %s.stamp\n outdir = %s\n", out, name, outdir))
	return edges.String()
}

// Desc returns a short description of the build action for ninja's progress output.
// Returns "jar" for the final packaging step (srcFile == "").
// Returns "javac" for individual source compilations (srcFile != "").
//
// Parameters:
//   - m: Module being evaluated (unused in this implementation)
//   - srcFile: Source file path; empty means this is a packaging step
//
// Returns:
//   - Description string for ninja's build log
func (r *javaLibrary) Desc(m *parser.Module, srcFile string) string {
	if srcFile == "" {
		return "jar"
	}
	return "javac"
}

// javaBinary implements a Java binary (executable) build rule.
// Java binaries are compiled Java programs that can be executed with proper classpath.
// The build process:
//   - javac compiles .java source files to .class files in a staging directory
//   - jar cf creates a JAR archive with a manifest containing Main-Class
//   - The manifest also includes Class-Path for runtime dependencies
//
// Unlike javaLibrary, this rule provides a .run target for execution
// with proper classpath that includes all dependencies.
//
// Required properties:
//   - name: The binary name (used for output JAR file name)
//   - main_class: The fully qualified name of the main class (e.g., "com.example.Main")
//   - srcs: Source files to compile
//
// Implements the BuildRule interface:
//   - Name() string: Returns "java_binary"
//   - NinjaRule(ctx) string: Returns ninja rules for javac and executable JAR creation
//   - Outputs(m, ctx) []string: Returns the .jar output path
//   - NinjaEdge(m, ctx) string: Returns ninja build edges
//   - Desc(m, src) string: Returns "jar" for packaging, "javac" for compilation
type javaBinary struct{}

// Name returns the module type name for java_binary.
// This name is used to match module types in Blueprint files (e.g., java_binary { ... }).
// Java binaries are executable JARs with a Main-Class manifest entry.
func (r *javaBinary) Name() string {
	return "java_binary"
}

// NinjaRule defines the ninja compilation, packaging, and execution rules for Java binaries.
// Creates three rules:
//   - javac_bin: Compiles Java sources to .class files in the outdir
//   - Uses -d flag to specify output directory for .class files
//   - jar_create_executable: Packages .class files into executable JAR with manifest
//   - Creates MANIFEST.MF with Manifest-Version, Main-Class, and Class-Path
//   - Uses "jar cfm" to create JAR with manifest file
//   - Main-Class specifies the entry point for java -jar command
//   - Class-Path specifies runtime dependencies (for java -jar classpath)
//
// Parameters:
//   - ctx: Rule render context (not used directly, but required by interface)
//
// Returns:
//   - Ninja rule definitions as formatted string
//
// Edge cases:
//   - This rule doesn't use toolchain context (always uses system javac and jar)
//   - Manifest values are sanitized via sanitizeManifestValue to prevent corruption
func (r *javaBinary) NinjaRule(ctx RuleRenderContext) string {
	return `rule javac_bin
  command = javac -d $outdir $in $flags

rule jar_create_executable
  command = echo "Manifest-Version: 1.0" > $outdir/MANIFEST.MF && echo "Main-Class: $main_class" >> $outdir/MANIFEST.MF && echo "Class-Path: $class_path" >> $outdir/MANIFEST.MF && jar cfm $out $outdir/MANIFEST.MF -C $outdir .

`
}

// Outputs returns the output paths for Java binaries.
// Returns nil if the module has no name (invalid module).
// Output format: {name}.jar
// The resulting JAR is executable via "java -jar {name}.jar".
//
// Parameters:
//   - m: Module being evaluated (must have "name" property)
//   - ctx: Rule render context (unused for Java binaries)
//
// Returns:
//   - List containing the executable JAR output path (e.g., ["foo.jar"])
//
// Edge cases:
//   - Empty name: Returns nil (cannot determine output path)
//   - No architecture suffix for Java (Java bytecode is platform-independent)
func (r *javaBinary) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	return []string{fmt.Sprintf("%s.jar", name)}
}

// NinjaEdge generates ninja build edges for compiling and packaging Java binaries.
// Returns empty string if name is empty, no sources provided, or main_class is missing.
//
// Build edges generated:
//  1. {name}.stamp: Depends on source files, compiles with javac to staging directory
//     - outdir variable points to {name}_classes staging directory
//     - flags variable contains javaflags from module
//  2. {name}.jar: Depends on stamp file, creates executable JAR with manifest
//     - outdir variable reused for jar command
//     - main_class variable specifies the Main-Class in manifest
//     - class_path variable specifies runtime Class-Path (typically ".")
//
// The manifest is created inline using echo and >> redirection to build the
// MANIFEST.MF file before running jar cfm.
//
// Parameters:
//   - m: Module being evaluated (must have "name", "srcs", and "main_class" properties)
//   - ctx: Rule render context (unused for Java binaries)
//
// Returns:
//   - Ninja build edge string for compilation and packaging
//   - Empty string if module has no name, no source files, or no main_class
//
// Edge cases:
//   - Empty srcs: Returns "" (no sources to compile)
//   - Missing name: Returns "" (cannot determine output path)
//   - Missing main_class: Returns "" (JAR cannot be executed without main class)
//   - Special characters in main_class: Sanitized via sanitizeManifestValue
func (r *javaBinary) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string {
	name := getName(m)
	srcs := getSrcs(m)
	mainClass := GetStringProp(m, "main_class")
	if name == "" || len(srcs) == 0 || mainClass == "" {
		return ""
	}

	javaflags := getJavaflags(m)
	out := r.Outputs(m, ctx)[0]
	outdir := sanitizeOutdir(name) + "_classes"
	safeMainClass := sanitizeManifestValue(mainClass)

	var edges strings.Builder
	edges.WriteString(fmt.Sprintf("build %s.stamp: javac_bin %s\n outdir = %s\n flags = %s\n", name, strings.Join(srcs, " "), outdir, javaflags))
	edges.WriteString(fmt.Sprintf("build %s: jar_create_executable %s.stamp\n outdir = %s\n main_class = %s\n class_path = .\n", out, name, outdir, safeMainClass))
	return edges.String()
}

// Desc returns a short description of the build action for ninja's progress output.
// Returns "jar" for the final packaging step (srcFile == "").
// Returns "javac" for individual source compilations (srcFile != "").
//
// Parameters:
//   - m: Module being evaluated (unused in this implementation)
//   - srcFile: Source file path; empty means this is a packaging step
//
// Returns:
//   - Description string for ninja's build log
func (r *javaBinary) Desc(m *parser.Module, srcFile string) string {
	if srcFile == "" {
		return "jar"
	}
	return "javac"
}

// javaLibraryStatic implements a static Java library build rule.
// Static libraries are used for linking into larger Java applications or for creating
// precompiled library distributions.
//
// The output naming convention uses the "lib*.a.jar" prefix (e.g., libfoo.a.jar)
// to distinguish static libraries from regular dynamic Java libraries.
// This naming helps build systems identify statically linkable artifacts.
//
// Note: In Java, "static" doesn't mean the same as in C/C++. It's a convention
// for libraries that are intended to be bundled into final applications.
//
// Implements the BuildRule interface:
//   - Name() string: Returns "java_library_static"
//   - NinjaRule(ctx) string: Returns ninja rules (same as java_library)
//   - Outputs(m, ctx) []string: Returns "lib{name}.a.jar"
//   - NinjaEdge(m, ctx) string: Returns ninja build edges
//   - Desc(m, src) string: Returns "jar" for packaging, "javac" for compilation
type javaLibraryStatic struct{}

// Name returns the module type name for java_library_static.
// This name is used to match module types in Blueprint files (e.g., java_library_static { ... }).
// Static Java libraries use "lib*.a.jar" naming convention.
func (r *javaLibraryStatic) Name() string {

	return "java_library_static"

}

// NinjaRule defines the ninja compilation and archiving rules for static Java libraries.
// Identical to javaLibrary's rules since the build process is the same.
// Only the output naming differs (lib*.a.jar instead of *.jar).
//
// Creates two rules:
//   - javac_lib: Compiles Java sources to .class files in the outdir
//   - Uses -d flag to specify output directory
//   - jar_create: Packages .class files into a .jar archive
//   - Uses -C flag to change to outdir before adding files
//
// Parameters:
//   - ctx: Rule render context (not used directly, but required by interface)
//
// Returns:
//   - Ninja rule definitions as formatted string
//
// Edge cases:
//   - This rule doesn't use toolchain context (always uses system javac and jar)
func (r *javaLibraryStatic) NinjaRule(ctx RuleRenderContext) string {

	return `rule javac_lib



  command = javac -d $outdir $in $flags



rule jar_create



  command = jar cf $out -C $outdir .



`

}

// Outputs returns the output paths for static Java libraries.
// Returns nil if the module has no name (invalid module).
// Output format: lib{name}.a.jar
// The ".a" prefix indicates static/archive semantics similar to Unix .a archive files.
//
// Parameters:
//   - m: Module being evaluated (must have "name" property)
//   - ctx: Rule render context (unused for Java libraries)
//
// Returns:
//   - List containing the static JAR output path (e.g., ["libfoo.a.jar"])
//
// Edge cases:
//   - Empty name: Returns nil (cannot determine output path)
//   - "lib" prefix handling: Not automatically added (name is used as-is after "lib" prefix)
func (r *javaLibraryStatic) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	return []string{fmt.Sprintf("lib%s.a.jar", name)}
}

// NinjaEdge generates ninja build edges for static library compilation and packaging.
// Returns empty string if name is empty or no sources are provided (invalid module).
//
// Build edges generated:
//  1. {name}.stamp: Depends on source files, compiles with javac to staging directory
//     - outdir variable points to {name}_classes staging directory
//     - flags variable contains javaflags from module
//  2. lib{name}.a.jar: Depends on stamp file, packages .class files with jar
//     - outdir variable reused for jar command
//
// Note: The stamp file uses the simple name, not the lib*.a.jar name, for consistency
// with the build system convention of tracking compilation with simple names.
//
// Parameters:
//   - m: Module being evaluated (must have "name" and "srcs" properties)
//   - ctx: Rule render context (unused for Java libraries)
//
// Returns:
//   - Ninja build edge string for compilation and packaging
//   - Empty string if module has no name or no source files
//
// Edge cases:
//   - Empty srcs: Returns "" (no sources to compile)
//   - Missing name: Returns "" (cannot determine output path)
//   - Special characters in name: Sanitized via sanitizeOutdir for outdir
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

// Desc returns a short description of the build action for ninja's progress output.
// Returns "jar" for the final packaging step (srcFile == "").
// Returns "javac" for individual source compilations (srcFile != "").
//
// Parameters:
//   - m: Module being evaluated (unused in this implementation)
//   - srcFile: Source file path; empty means this is a packaging step
//
// Returns:
//   - Description string for ninja's build log
func (r *javaLibraryStatic) Desc(m *parser.Module, srcFile string) string {
	if srcFile == "" {
		return "jar"
	}
	return "javac"
}

// javaLibraryHost implements a Java library build rule for host builds.
// Host-specific libraries are compiled to run on the build host system rather than
// the target device or emulator.
//
// The output naming convention appends "-host" suffix (e.g., name-host.jar) to
// distinguish host artifacts from target/device artifacts. This is essential
// when cross-compiling for Android, where build tools may run on Linux/Mac
// but produce artifacts for Android devices.
//
// Use cases:
//   - Build tools that run during the build process
//   - Code generators that create source files
//   - Utilities needed by the build system itself
//
// Implements the BuildRule interface:
//   - Name() string: Returns "java_library_host"
//   - NinjaRule(ctx) string: Returns ninja rules (same as java_library)
//   - Outputs(m, ctx) []string: Returns "{name}-host.jar"
//   - NinjaEdge(m, ctx) string: Returns ninja build edges
//   - Desc(m, src) string: Returns "jar" for packaging, "javac" for compilation
type javaLibraryHost struct{}

// Name returns the module type name for java_library_host.
// This name is used to match module types in Blueprint files (e.g., java_library_host { ... }).
// Host libraries are intended to run on the build machine, not the target device.
func (r *javaLibraryHost) Name() string { return "java_library_host" }

// NinjaRule defines the ninja compilation and archiving rules for host libraries.
// Identical to javaLibrary's rules since the build process is the same.
// Only the output naming differs (appends "-host" suffix).
//
// Creates two rules:
//   - javac_lib: Compiles Java sources to .class files in the outdir
//   - Uses -d flag to specify output directory
//   - jar_create: Packages .class files into a .jar archive
//   - Uses -C flag to change to outdir before adding files
//
// Parameters:
//   - ctx: Rule render context (not used directly, but required by interface)
//
// Returns:
//   - Ninja rule definitions as formatted string
//
// Edge cases:
//   - This rule doesn't use toolchain context (always uses system javac and jar)
func (r *javaLibraryHost) NinjaRule(ctx RuleRenderContext) string {

	return `rule javac_lib

  command = javac -d $outdir $in $flags

rule jar_create

  command = jar cf $out -C $outdir .

`

}

// Outputs returns the output paths for host libraries.
// Returns nil if the module has no name (invalid module).
// Output format: {name}-host.jar
// The "-host" suffix identifies this as a host-native artifact.
//
// Parameters:
//   - m: Module being evaluated (must have "name" property)
//   - ctx: Rule render context (unused for Java libraries)
//
// Returns:
//   - List containing the host JAR output path (e.g., ["foo-host.jar"])
//
// Edge cases:
//   - Empty name: Returns nil (cannot determine output path)
//   - "-host" suffix distinguishes from device artifacts in cross-compilation
func (r *javaLibraryHost) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	return []string{fmt.Sprintf("%s-host.jar", name)}
}

// NinjaEdge generates ninja build edges for host library compilation and packaging.
// Returns empty string if name is empty or no sources are provided (invalid module).
//
// Build edges generated:
//  1. {name}.stamp: Depends on source files, compiles with javac to staging directory
//     - outdir variable points to {name}_classes staging directory
//     - flags variable contains javaflags from module
//  2. {name}-host.jar: Depends on stamp file, packages .class files with jar
//     - outdir variable reused for jar command
//
// Host variants are used for build tools, generators, and utilities that must
// run on the build host during the build process.
//
// Parameters:
//   - m: Module being evaluated (must have "name" and "srcs" properties)
//   - ctx: Rule render context (unused for Java libraries)
//
// Returns:
//   - Ninja build edge string for compilation and packaging
//   - Empty string if module has no name or no source files
//
// Edge cases:
//   - Empty srcs: Returns "" (no sources to compile)
//   - Missing name: Returns "" (cannot determine output path)
//   - Special characters in name: Sanitized via sanitizeOutdir for outdir
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

// Desc returns a short description of the build action for ninja's progress output.
// Returns "jar" for the final packaging step (srcFile == "").
// Returns "javac" for individual source compilations (srcFile != "").
//
// Parameters:
//   - m: Module being evaluated (unused in this implementation)
//   - srcFile: Source file path; empty means this is a packaging step
//
// Returns:
//   - Description string for ninja's build log
func (r *javaLibraryHost) Desc(m *parser.Module, srcFile string) string {
	if srcFile == "" {
		return "jar"
	}
	return "javac"
}

// javaBinaryHost implements a Java binary build rule for host builds.
// Host-specific binaries are compiled to run on the build host system rather than
// the target device or emulator.
//
// Like javaBinary, this produces executable JARs with a main class manifest,
// but the output uses the "-host" suffix (e.g., name-host.jar) to identify
// it as a host-native artifact. Used for build tools, generators, and other
// utilities that must run during the host-side build process.
//
// Required properties:
//   - name: The binary name (used for output JAR file name)
//   - main_class: The fully qualified name of the main class
//   - srcs: Source files to compile
//
// Implements the BuildRule interface:
//   - Name() string: Returns "java_binary_host"
//   - NinjaRule(ctx) string: Returns ninja rules for javac and executable JAR
//   - Outputs(m, ctx) []string: Returns "{name}-host.jar"
//   - NinjaEdge(m, ctx) string: Returns ninja build edges
//   - Desc(m, src) string: Returns "jar" for packaging, "javac" for compilation
type javaBinaryHost struct{}

// Name returns the module type name for java_binary_host.
// This name is used to match module types in Blueprint files (e.g., java_binary_host { ... }).
// Host binaries run on the build machine, not the target device.
func (r *javaBinaryHost) Name() string { return "java_binary_host" }

// NinjaRule defines the ninja compilation and executable JAR creation rules.
// Identical to javaBinary's rules since the build process is the same.
// Only the output naming differs (appends "-host" suffix).
//
// Creates two rules:
//   - javac_bin: Compiles Java sources to .class files in the outdir
//   - Uses -d flag to specify output directory
//   - jar_create_executable: Creates executable JAR with main class
//   - Uses "jar cfe" to set entry point directly (simpler than manifest)
//   - -C flag changes to outdir before adding files
//
// Parameters:
//   - ctx: Rule render context (not used directly, but required by interface)
//
// Returns:
//   - Ninja rule definitions as formatted string
//
// Edge cases:
//   - This rule doesn't use toolchain context (always uses system javac and jar)
func (r *javaBinaryHost) NinjaRule(ctx RuleRenderContext) string {

	return `rule javac_bin

  command = javac -d $outdir $in $flags

rule jar_create_executable

  command = jar cfe $out $main_class -C $outdir .

`

}

// Outputs returns the output paths for host binaries.
// Returns nil if the module has no name (invalid module).
// Output format: {name}-host.jar
// The "-host" suffix identifies this as a host-native executable artifact.
//
// Parameters:
//   - m: Module being evaluated (must have "name" property)
//   - ctx: Rule render context (unused for Java binaries)
//
// Returns:
//   - List containing the host executable JAR output path (e.g., ["foo-host.jar"])
//
// Edge cases:
//   - Empty name: Returns nil (cannot determine output path)
//   - "-host" suffix distinguishes from device artifacts in cross-compilation
func (r *javaBinaryHost) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	return []string{fmt.Sprintf("%s-host.jar", name)}
}

// NinjaEdge generates ninja build edges for host binary compilation and packaging.
// Returns empty string if name is empty, no sources provided, or main_class is missing.
//
// Build edges generated:
//  1. {name}.stamp: Depends on source files, compiles with javac to staging directory
//     - outdir variable points to {name}_classes staging directory
//     - flags variable contains javaflags from module
//  2. {name}-host.jar: Depends on stamp file, creates executable JAR with main_class
//     - outdir variable reused for jar command
//     - main_class variable specifies the Main-Class for java -jar
//
// Uses "jar cfe" which directly sets the entry point without needing a manifest file.
//
// Parameters:
//   - m: Module being evaluated (must have "name", "srcs", and "main_class" properties)
//   - ctx: Rule render context (unused for Java binaries)
//
// Returns:
//   - Ninja build edge string for compilation and packaging
//   - Empty string if module has no name, no source files, or no main_class
//
// Edge cases:
//   - Empty srcs: Returns "" (no sources to compile)
//   - Missing name: Returns "" (cannot determine output path)
//   - Missing main_class: Returns "" (host binaries still need a main class)
//   - Special characters in name: Sanitized via sanitizeOutdir for outdir
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

// Desc returns a short description of the build action.
func (r *javaBinaryHost) Desc(m *parser.Module, srcFile string) string {
	if srcFile == "" {
		return "jar"
	}
	return "javac"
}

// javaTest implements a Java test build rule.
// Java tests are compiled test classes packaged as test JARs with test option support.
//
// The output naming convention uses the "-test" suffix (e.g., name-test.jar) to
// identify test artifacts. Supports test-specific flags and arguments via the
// test_options and test_config properties. Test JARs are typically executed
// by test runners like JUnit or Android's test framework.
//
// Required properties:
//   - name: The test name (used for output JAR file name)
//   - srcs: Test source files to compile
//   - Optional: test_options for additional test arguments
//
// Implements the BuildRule interface:
//   - Name() string: Returns "java_test"
//   - NinjaRule(ctx) string: Returns ninja rules for javac and test JAR
//   - Outputs(m, ctx) []string: Returns "{name}-test.jar"
//   - NinjaEdge(m, ctx) string: Returns ninja build edges
//   - Desc(m, src) string: Returns "jar" for packaging, "javac" for compilation
type javaTest struct{}

// Name returns the module type name for java_test.
// This name is used to match module types in Blueprint files (e.g., java_test { ... }).
// Test JARs are identified by the "-test" suffix and run by test frameworks.
func (r *javaTest) Name() string { return "java_test" }

// NinjaRule defines the ninja compilation and test JAR creation rules.
// Creates two rules:
//   - javac_test: Compiles test sources with test-specific flags
//   - Uses -d flag to specify output directory
//   - jar_test: Packages test .class files into a test JAR
//   - Uses -C flag to change to outdir before adding files
//
// Test JARs are typically executed by test runners like JUnit.
//
// Parameters:
//   - ctx: Rule render context (not used directly, but required by interface)
//
// Returns:
//   - Ninja rule definitions as formatted string
//
// Edge cases:
//   - This rule doesn't use toolchain context (always uses system javac and jar)
func (r *javaTest) NinjaRule(ctx RuleRenderContext) string {

	return `rule javac_test

  command = javac -d $outdir $in $flags

rule jar_test

  command = jar cf $out -C $outdir .

`

}

// Outputs returns the output paths for test modules.
// Returns nil if the module has no name (invalid module).
// Output format: {name}-test.jar
// The "-test" suffix identifies this as a test artifact.
//
// Parameters:
//   - m: Module being evaluated (must have "name" property)
//   - ctx: Rule render context (unused for Java tests)
//
// Returns:
//   - List containing the test JAR output path (e.g., ["foo-test.jar"])
//
// Edge cases:
//   - Empty name: Returns nil (cannot determine output path)
//   - "-test" suffix distinguishes from regular libraries and binaries
func (r *javaTest) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	return []string{fmt.Sprintf("%s-test.jar", name)}
}

// NinjaEdge generates ninja build edges for test compilation and packaging.
// Returns empty string if name is empty or no sources are provided (invalid module).
//
// Build edges generated:
//  1. {name}.stamp: Depends on source files, compiles with javac to staging directory
//     - outdir variable points to {name}_classes staging directory
//     - flags variable contains javaflags from module (may include test-specific flags)
//  2. {name}-test.jar: Depends on stamp file, packages .class files with jar
//     - outdir variable reused for jar command
//  3. Optional test_args variable: Additional arguments for test execution
//     - Set if test_options property contains arguments
//
// The test_args variable can be used by test runners to pass arguments to the test.
//
// Parameters:
//   - m: Module being evaluated (must have "name" and "srcs" properties)
//   - ctx: Rule render context (unused for Java tests)
//
// Returns:
//   - Ninja build edge string for compilation and packaging
//   - Empty string if module has no name or no source files
//
// Edge cases:
//   - Empty srcs: Returns "" (no sources to compile)
//   - Missing name: Returns "" (cannot determine output path)
//   - Special characters in name: Sanitized via sanitizeOutdir for outdir
//   - test_options property: Adds test_args variable to build edge
func (r *javaTest) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string {
	name := getName(m)
	srcs := getSrcs(m)
	if name == "" || len(srcs) == 0 {
		return ""
	}

	javaflags := getJavaflags(m)
	out := r.Outputs(m, ctx)[0]
	outdir := name + "_classes"
	testArgs := getTestOptionArgs(m)

	var edges strings.Builder
	edges.WriteString(fmt.Sprintf("build %s.stamp: javac_test %s\n outdir = %s\n flags = %s\n", name, strings.Join(srcs, " "), outdir, javaflags))
	edges.WriteString(fmt.Sprintf("build %s: jar_test %s.stamp\n outdir = %s\n", out, name, outdir))
	if testArgs != "" {
		edges.WriteString(fmt.Sprintf(" test_args = %s\n", testArgs))
	}
	return edges.String()
}

// Desc returns a short description of the build action.
func (r *javaTest) Desc(m *parser.Module, srcFile string) string {
	if srcFile == "" {
		return "jar"
	}
	return "javac"
}

// javaImport implements a prebuilt JAR import rule.
// This rule copies pre-built .jar files into the build tree without recompilation.
// It is used for importing external JAR dependencies or precompiled libraries
// that should not be rebuilt from source.
//
// The rule copies source JAR files directly to the output location, supporting
// cross-platform builds by using "cp" on Unix or "cmd /c copy" on Windows.
// This allows prebuilt binaries to be integrated into the dependency graph.
//
// Use cases:
//   - Third-party libraries distributed as precompiled JARs
//   - JARs built by external build systems
//   - Proprietary libraries without source code
//
// Implements the BuildRule interface:
//   - Name() string: Returns "java_import"
//   - NinjaRule(ctx) string: Returns ninja copy rule
//   - Outputs(m, ctx) []string: Returns "{name}.jar"
//   - NinjaEdge(m, ctx) string: Returns ninja build edge for copying
//   - Desc(m, src) string: Returns "cp" for copy action
type javaImport struct{}

// Name returns the module type name for java_import.
// This name is used to match module types in Blueprint files (e.g., java_import { ... }).
// Imported JARs are not recompiled; they are copied directly to the output.
func (r *javaImport) Name() string { return "java_import" }

// NinjaRule defines the ninja copy rule for importing prebuilt JARs.
// Selects the appropriate copy command based on host operating system:
//   - Unix/Linux/Mac: Uses "cp $in $out"
//   - Windows: Uses "cmd /c copy $in $out"
//
// The copy command is embedded directly in the rule definition.
// This allows prebuilt JARs to be used as dependencies without recompilation.
//
// Parameters:
//   - ctx: Rule render context (not used directly, but required by interface)
//
// Returns:
//   - Ninja rule definition as formatted string
//
// Edge cases:
//   - Windows detection: Uses runtime.GOOS to detect Windows OS
//   - No compilation needed: This rule only copies files
func (r *javaImport) NinjaRule(ctx RuleRenderContext) string {

	copyCmd := "cp $in $out"

	if runtime.GOOS == "windows" {

		copyCmd = "cmd /c copy $in $out"

	}

	return `rule java_import

  command = ` + copyCmd + `

`

}

// Outputs returns the output paths for imported JARs.
// Returns nil if the module has no name (invalid module).
// Output format: {name}.jar
// The imported JAR maintains its original name in the build tree.
//
// Parameters:
//   - m: Module being evaluated (must have "name" property)
//   - ctx: Rule render context (unused for Java imports)
//
// Returns:
//   - List containing the JAR output path (e.g., ["foo.jar"])
//
// Edge cases:
//   - Empty name: Returns nil (cannot determine output path)
//   - No architecture suffix for Java (platform-independent bytecode)
func (r *javaImport) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	return []string{fmt.Sprintf("%s.jar", name)}
}

// NinjaEdge generates ninja build edges for importing prebuilt JARs.
// Returns empty string if no sources are provided (nothing to import).
//
// Build edges generated:
//   - {name}.jar: Depends on source JAR(s), copies to output location
//   - Uses java_import rule which invokes cp or cmd /c copy
//
// Parameters:
//   - m: Module being evaluated (must have "srcs" property with prebuilt JAR paths)
//   - ctx: Rule render context (unused for Java imports)
//
// Returns:
//   - Ninja build edge string for copying
//   - Empty string if module has no source files
//
// Edge cases:
//   - Empty srcs: Returns "" (no files to import)
//   - Multiple srcs: All are passed as inputs to the copy rule
func (r *javaImport) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string {
	srcs := getSrcs(m)
	if len(srcs) == 0 {
		return ""
	}

	out := r.Outputs(m, ctx)[0]
	return fmt.Sprintf("build %s: java_import %s\n", out, strings.Join(srcs, " "))
}

// Desc returns a short description of the build action for ninja's progress output.
// Always returns "cp" since java_import only performs file copying.
//
// Parameters:
//   - m: Module being evaluated (unused in this implementation)
//   - srcFile: Source file path (unused, always returns "cp")
//
// Returns:
//   - "cp" as the description for all java_import operations
func (r *javaImport) Desc(m *parser.Module, srcFile string) string {
	return "cp"
}
