// ninja/helpers.go - Helper functions for ninja rule generation
//
// This file provides utility functions for extracting and transforming
// module properties into format suitable for ninja rule generation.
// These functions bridge between the parser's AST representation and
// the string/slice values needed by the ninja Writer.
//
// Property extraction functions:
//   - GetStringProp: Get string property from module
//   - GetStringPropEval: Get string with variable evaluation
//   - GetListProp: Get list property from module
//   - GetListPropEval: Get list with variable evaluation
//   - GetMapProp: Get map property from module
//   - GetMapStringListProp: Get string list from map property
//
// Flag and include directory helpers:
//   - getCflags, getCppflags, getLdflags: Extract compiler/linker flags
//   - getLocalIncludeDirs, getSystemIncludeDirs: Extract include directories
//   - getExportIncludeDirs, getExportedHeaders: Extract exported headers
//
// Output name generators:
//   - objectOutputName: Generate unique object file names
//   - libOutputName: Generate library output names
//   - sharedLibOutputName: Generate .so library names
//   - staticLibOutputName: Generate .a library names
package ninja

import (
	"minibp/lib/parser"
	"path/filepath"
	"runtime"
	"strings"
)

// GetStringProp retrieves a string property value from a module.
// Returns empty string if property not found or wrong type.
//
// Parameters:
//   - m: The parser.Module to get the property from
//   - name: The property name to look for
//
// Returns:
//   - The string value if found, empty string otherwise
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

// GetStringPropEval retrieves a string property value with variable evaluation.
// If an evaluator is provided, it evaluates any variable references in the property.
// This allows properties to contain ${VAR} references that are resolved at generation time.
//
// Parameters:
//   - m: The parser.Module to get the property from
//   - name: The property name to look for
//   - eval: The evaluator for variable resolution (can be nil)
//
// Returns:
//   - The string value, resolved if eval provided, empty if not found
func GetStringPropEval(m *parser.Module, name string, eval *parser.Evaluator) string {
	if m.Map == nil {
		return ""
	}
	for _, prop := range m.Map.Properties {
		if prop.Name == name {
			if s, ok := prop.Value.(*parser.String); ok {
				if eval != nil {
					return parser.EvalToString(s, eval)
				}
				return s.Value
			}
		}
	}
	return ""
}

// getBoolProp retrieves a boolean property value from a module.
// Returns false if property not found or wrong type.
// This is used for boolean properties like "enabled" or "shared".
//
// Parameters:
//   - m: The parser.Module to get the property from
//   - name: The property name to look for
//
// Returns:
//   - The boolean value if found, false otherwise
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
// Returns nil if property not found or not a list type.
//
// Parameters:
//   - m: The parser.Module to get the property from
//   - name: The property name to look for
//
// Returns:
//   - The list of strings if found, nil otherwise
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

// GetListPropEval retrieves a list property value with variable evaluation.
// If an evaluator is provided, it evaluates any variable references in list items.
//
// Parameters:
//   - m: The parser.Module to get the property from
//   - name: The property name to look for
//   - eval: The evaluator for variable resolution (can be nil)
//
// Returns:
//   - The list of strings, resolved if eval provided, nil if not found
func GetListPropEval(m *parser.Module, name string, eval *parser.Evaluator) []string {
	if m.Map == nil {
		return nil
	}
	for _, prop := range m.Map.Properties {
		if prop.Name == name {
			if l, ok := prop.Value.(*parser.List); ok {
				if eval != nil {
					return parser.EvalToStringList(l, eval)
				}
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

// getCflags retrieves C compiler flags from a module.
// Returns space-separated string of flags from "cflags" property.
//
// This is a convenience function that joins the list property values.
// Returns empty string if no cflags property.
//
// Parameters:
//   - m: The parser.Module to get flags from
//
// Returns:
//   - Space-separated flags string, empty if none
func getCflags(m *parser.Module) string {
	return strings.Join(GetListProp(m, "cflags"), " ")
}

// getCppflags retrieves C++ compiler flags from a module.
// Returns space-separated string of flags from "cppflags" property.
//
// This is a convenience function that joins the list property values.
// Returns empty string if no cppflags property.
//
// Parameters:
//   - m: The parser.Module to get flags from
//
// Returns:
//   - Space-separated flags string, empty if none
func getCppflags(m *parser.Module) string {
	return strings.Join(GetListProp(m, "cppflags"), " ")
}

// getLdflags retrieves linker flags from a module.
// Returns space-separated string of flags from "ldflags" property.
//
// This is a convenience function that joins the list property values.
// Returns empty string if no ldflags property.
//
// Parameters:
//   - m: The parser.Module to get flags from
//
// Returns:
//   - Space-separated flags string, empty if none
func getLdflags(m *parser.Module) string {
	return strings.Join(GetListProp(m, "ldflags"), " ")
}

// getGoflags retrieves Go compiler flags from a module.
// Returns space-separated string of flags from "goflags" property.
//
// This is a convenience function that joins the list property values.
// Returns empty string if no goflags property.
//
// Parameters:
//   - m: The parser.Module to get flags from
//
// Returns:
//   - Space-separated flags string, empty if none
func getGoflags(m *parser.Module) string {
	return strings.Join(GetListProp(m, "goflags"), " ")
}

// getLto retrieves the LTO mode from a module.
// Returns the value of the "lto" property.
//
// LTO (Link-Time Optimization) modes:
//   - "full": Full LTO
//   - "thin": Thin LTO
//   - "": No LTO
//
// Parameters:
//   - m: The parser.Module to get LTO from
//
// Returns:
//   - LTO mode string, empty if not specified
func getLto(m *parser.Module) string {
	return GetStringProp(m, "lto")
}

// getLocalIncludeDirs retrieves local include directories from a module.
// Returns the list from "local_include_dirs" property.
//
// Local includes are search paths relative to the module's directory.
// They are added with -I prefix to compiler commands.
//
// Parameters:
//   - m: The parser.Module to get include dirs from
//
// Returns:
//   - List of local include directory paths, nil if none
func getLocalIncludeDirs(m *parser.Module) []string {
	return GetListProp(m, "local_include_dirs")
}

// getSystemIncludeDirs retrieves system include directories from a module.
// Returns the list from "system_include_dirs" property.
//
// System includes are search paths for system headers.
// They are added with -isystem prefix to compiler commands.
//
// Parameters:
//   - m: The parser.Module to get include dirs from
//
// Returns:
//   - List of system include directory paths, nil if none
func getSystemIncludeDirs(m *parser.Module) []string {
	return GetListProp(m, "system_include_dirs")
}

// getGoTargetVariants retrieves target variant keys from a Go module.
// Go modules can have target-specific variants (e.g., "linux_amd64", "darwin_arm64).
// This returns the keys for all target variant property maps.
//
// Parameters:
//   - m: The parser.Module representing a Go module
//
// Returns:
//   - List of target variant keys (e.g., ["linux_amd64", "darwin_arm64"])
func getGoTargetVariants(m *parser.Module) []string {
	if m.Target == nil {
		return nil
	}
	var keys []string
	for _, p := range m.Target.Properties {
		if _, ok := p.Value.(*parser.Map); !ok {
			continue
		}
		keys = append(keys, p.Name)
	}
	return keys
}

// getGoTargetProp extracts a string property from a target variant sub-map.
// Go modules can have target-specific properties nested under variant maps.
// This function extracts a specific property from a variant's property map.
//
// For example, in a go_library with a "linux_amd64" target variant,
// getGoTargetProp(m, "linux_amd64", "os") might return "linux".
//
// Parameters:
//   - m: The parser.Module representing a Go module
//   - variant: The target variant name (e.g., "linux_amd64")
//   - prop: The property name to extract from the variant
//
// Returns:
//   - The property value if found, empty string otherwise
func getGoTargetProp(m *parser.Module, variant, prop string) string {
	if m.Target == nil {
		return ""
	}
	for _, p := range m.Target.Properties {
		if p.Name != variant {
			continue
		}
		sub, ok := p.Value.(*parser.Map)
		if !ok {
			return ""
		}
		for _, sp := range sub.Properties {
			if sp.Name == prop {
				if s, ok := sp.Value.(*parser.String); ok {
					return s.Value
				}
			}
		}
	}
	return ""
}

// getJavaflags retrieves Java compiler flags from a module.
// Returns space-separated string of flags from "javaflags" property.
//
// This is a convenience function that joins the list property values.
// Returns empty string if no javaflags property.
//
// Parameters:
//   - m: The parser.Module to get flags from
//
// Returns:
//   - Space-separated flags string, empty if none
func getJavaflags(m *parser.Module) string {
	return strings.Join(GetListProp(m, "javaflags"), " ")
}

// getExportIncludeDirs retrieves exported include directories from a module.
// Returns the list from "export_include_dirs" property.
//
// Exported include directories are made available to modules that depend
// on this module. They're typically used for C/C++ header libraries.
//
// Parameters:
//   - m: The parser.Module to get include dirs from
//
// Returns:
//   - List of exported include directory paths, nil if none
func getExportIncludeDirs(m *parser.Module) []string {
	return GetListProp(m, "export_include_dirs")
}

// getExportedHeaders retrieves exported header files from a module.
// Returns the list from "exported_headers" property.
//
// Exported header files are made available to modules that depend on this module.
// They are typically installed to the export include directories.
//
// Parameters:
//   - m: The parser.Module to get headers from
//
// Returns:
//   - List of exported header file paths, nil if none
func getExportedHeaders(m *parser.Module) []string {
	return GetListProp(m, "exported_headers")
}

// getName retrieves the module name from a module.
// This is a convenience wrapper around GetStringProp.
//
// Parameters:
//   - m: The parser.Module to get name from
//
// Returns:
//   - The module name, empty if not specified
func getName(m *parser.Module) string {
	return GetStringProp(m, "name")
}

// getSrcs retrieves source file paths from a module.
// Returns the list from "srcs" property.
//
// Parameters:
//   - m: The parser.Module to get source files from
//
// Returns:
//   - List of source file paths, nil if none
func getSrcs(m *parser.Module) []string {
	return GetListProp(m, "srcs")
}

// formatSrcs combines source file paths into a single space-separated string.
// This is a convenience function for building command-line arguments.
//
// Parameters:
//   - srcs: Slice of source file paths
//
// Returns:
//   - Space-separated string of all source paths
func formatSrcs(srcs []string) string {
	return strings.Join(srcs, " ")
}

// objectOutputName generates a unique object file name for a source file.
// This ensures each source file maps to a distinct output object file,
// even when multiple source files have the same base name.
//
// The function:
//   - Extracts the base name (without extension)
//   - Replaces path separators and special chars with underscores
//   - Removes leading dots
//   - Prefixes with module name if needed for uniqueness
//
// Parameters:
//   - moduleName: The name of the module containing this source
//   - src: The source file path
//
// Returns:
//   - Unique object file name (e.g., "mylib_foo.o")
func objectOutputName(moduleName, src string) string {
	clean := filepath.Clean(src)
	clean = strings.TrimPrefix(clean, "./")
	clean = strings.TrimPrefix(clean, "../")
	srcName := strings.TrimSuffix(clean, filepath.Ext(clean))
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", " ", "_")
	srcName = replacer.Replace(srcName)
	srcName = strings.Trim(srcName, "._")
	if srcName == "" {
		srcName = "obj"
	}
	if strings.HasPrefix(srcName, moduleName) || srcName == moduleName {
		return srcName + ".o"
	}
	return moduleName + "_" + srcName + ".o"
}

// joinFlags combines multiple flag strings into a single space-separated string.
// Empty or whitespace-only parts are filtered out.
// This is used to combine compiler flags from multiple sources.
//
// Parameters:
//   - parts: Flag strings to combine
//
// Returns:
//   - Space-separated combined flags, empty if all parts empty
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

// libOutputName generates the output name for a library.
// Adds "lib" prefix if not present, and appends architecture suffix and extension.
//
// Parameters:
//   - name: Base library name
//   - archSuffix: Architecture suffix (e.g., "_arm64")
//   - ext: File extension (e.g., ".a", ".so")
//
// Returns:
//   - Full library output name (e.g., "libfoo_arm64.a")
func libOutputName(name, archSuffix, ext string) string {
	libName := name
	if !strings.HasPrefix(name, "lib") {
		libName = "lib" + name
	}
	return libName + archSuffix + ext
}

// sharedLibOutputName generates the output name for a shared library (.so).
// Convenience function that calls libOutputName with ".so" extension.
//
// Parameters:
//   - name: Base library name
//   - archSuffix: Architecture suffix
//
// Returns:
//   - Full shared library name (e.g., "libfoo_arm64.so")
func sharedLibOutputName(name string, archSuffix string) string {
	return libOutputName(name, archSuffix, ".so")
}

// staticLibOutputName generates the output name for a static library (.a).
// Convenience function that calls libOutputName with ".a" extension.
//
// Parameters:
//   - name: Base library name
//   - archSuffix: Architecture suffix
//
// Returns:
//   - Full static library name (e.g., "libfoo_arm64.a")
func staticLibOutputName(name string, archSuffix string) string {
	return libOutputName(name, archSuffix, ".a")
}

// getFirstSource retrieves the first source file from a module.
// Returns empty string if module has no sources.
//
// Parameters:
//   - m: The parser.Module to get source from
//
// Returns:
//   - First source file path, empty string if none
func getFirstSource(m *parser.Module) string {
	srcs := getSrcs(m)
	if len(srcs) == 0 {
		return ""
	}
	return srcs[0]
}

// getData retrieves data file paths from a module.
// Data files are files that need to be available at runtime,
// typically for testing. They are copied to the build output directory.
//
// Parameters:
//   - m: The parser.Module to get data files from
//
// Returns:
//   - List of data file paths, nil if none
func getData(m *parser.Module) []string {
	return GetListProp(m, "data")
}

// copyCommand returns the platform-specific copy command for ninja.
// This is used for rules that copy files during the build.
//
// The command uses $in and $out ninja variables:
//   - Unix: cp $in $out
//   - Windows: cmd /c copy $in $out
//
// Parameters: None
//
// Returns:
//   - Platform-appropriate copy command string
func copyCommand() string {
	if runtime.GOOS == "windows" {
		return "cmd /c copy $in $out"
	}
	return "cp $in $out"
}

// getTestOptionArgs retrieves test option arguments from a module.
// Returns space-separated args from the "test_options" map property.
// Test options can specify additional arguments for test execution.
//
// Parameters:
//   - m: The parser.Module to get test options from
//
// Returns:
//   - Space-separated test option arguments, empty if none
func getTestOptionArgs(m *parser.Module) string {
	return strings.Join(GetMapStringListProp(GetMapProp(m, "test_options"), "args"), " ")
}

// GetMapProp retrieves a map property value from a module.
// Map properties are nested property structures, like:
//
//   test_options {
//     args: ["-v", "-cover"],
//     env: ["FOO=bar"],
//   }
//
// Parameters:
//   - m: The parser.Module to get the property from
//   - name: The property name to look for
//
// Returns:
//   - The parser.Map if found, nil otherwise
func GetMapProp(m *parser.Module, name string) *parser.Map {
	if m.Map == nil {
		return nil
	}
	for _, prop := range m.Map.Properties {
		if prop.Name == name {
			if mp, ok := prop.Value.(*parser.Map); ok {
				return mp
			}
		}
	}
	return nil
}

// GetMapStringListProp retrieves a string list property from a map.
// This extracts a list or single string property from a nested map.
// Handles both list and single string values.
//
// Parameters:
//   - mp: The parser.Map to get the property from
//   - name: The property name to look for
//
// Returns:
//   - The list of strings if found, nil otherwise
//   - Single string values are wrapped in a single-element list
func GetMapStringListProp(mp *parser.Map, name string) []string {
	if mp == nil {
		return nil
	}
	for _, prop := range mp.Properties {
		if prop.Name == name {
			if list, ok := prop.Value.(*parser.List); ok {
				var out []string
				for _, v := range list.Values {
					if s, ok := v.(*parser.String); ok {
						out = append(out, s.Value)
					}
				}
				return out
			}
			if s, ok := prop.Value.(*parser.String); ok {
				return []string{s.Value}
			}
		}
	}
	return nil
}
