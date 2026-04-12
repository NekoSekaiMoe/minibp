// ninja/rules.go - Ninja rule definitions for minibp
package ninja

import (
	"fmt"
	"strings"

	"minibp/parser"
)

// BuildRule is the interface for all ninja rule implementations
type BuildRule interface {
	Name() string
	NinjaRule() string                 // returns ninja rule definition
	NinjaEdge(m *parser.Module) string // returns build edge for a module
	Outputs(m *parser.Module) []string
}

// getStringProp extracts a string property from a module
func getStringProp(m *parser.Module, name string) string {
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

// getListProp extracts a list of strings from a module
func getListProp(m *parser.Module, name string) []string {
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

// getCflags extracts cflags from a module
func getCflags(m *parser.Module) string {
	return formatFlags(getListProp(m, "cflags"))
}

// getCppflags extracts cppflags from a module
func getCppflags(m *parser.Module) string {
	return formatFlags(getListProp(m, "cppflags"))
}

// getLdflags extracts ldflags from a module
func getLdflags(m *parser.Module) string {
	return formatFlags(getListProp(m, "ldflags"))
}

// getGoflags extracts goflags from a module
func getGoflags(m *parser.Module) string {
	return formatFlags(getListProp(m, "goflags"))
}

// getJavaflags extracts javaflags from a module
func getJavaflags(m *parser.Module) string {
	return formatFlags(getListProp(m, "javaflags"))
}

// formatFlags formats a list of flags as a space-separated string
func formatFlags(flags []string) string {
	return strings.Join(flags, " ")
}

// getName returns the 'name' property of a module
func getName(m *parser.Module) string {
	return getStringProp(m, "name")
}

// getSrcs returns the 'srcs' property of a module
func getSrcs(m *parser.Module) []string {
	return getListProp(m, "srcs")
}

// formatSrcs formats source files as a space-separated string
func formatSrcs(srcs []string) string {
	return strings.Join(srcs, " ")
}

// ccLibrary implements the cc_library rule
type ccLibrary struct{}

func (r *ccLibrary) Name() string {
	return "cc_library"
}

func (r *ccLibrary) NinjaRule() string {
	return `rule cc_compile
    command = gcc -c $in -o $out $flags

rule cc_archive
    command = ar rcs $out $in
`
}

func (r *ccLibrary) Outputs(m *parser.Module) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	return []string{fmt.Sprintf("lib%s.a", name)}
}

func (r *ccLibrary) NinjaEdge(m *parser.Module) string {
	name := getName(m)
	srcs := getSrcs(m)
	if name == "" || len(srcs) == 0 {
		return ""
	}

	var edges strings.Builder
	objFiles := make([]string, 0, len(srcs))

	// Extract flags
	cflags := getCflags(m)

	// Generate compile edges for each source file
	for _, src := range srcs {
		obj := strings.TrimSuffix(src, ".c")
		obj = strings.TrimSuffix(obj, ".cc")
		obj += ".o"
		objFiles = append(objFiles, obj)
		edges.WriteString(fmt.Sprintf("build %s: cc_compile %s\n flags = %s\n", obj, src, cflags))
	}

	// Generate archive edge
	out := r.Outputs(m)[0]
	edges.WriteString(fmt.Sprintf("build %s: cc_archive %s\n", out, strings.Join(objFiles, " ")))

	return edges.String()
}

// ccBinary implements the cc_binary rule
type ccBinary struct{}

func (r *ccBinary) Name() string {
	return "cc_binary"
}

func (r *ccBinary) NinjaRule() string {
	return `rule cc_link
    command = gcc -o $out $in $flags
`
}

func (r *ccBinary) Outputs(m *parser.Module) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	return []string{name}
}

func (r *ccBinary) NinjaEdge(m *parser.Module) string {
	name := getName(m)
	srcs := getSrcs(m)
	if name == "" || len(srcs) == 0 {
		return ""
	}

	cflags := getCflags(m)
	ldflags := getLdflags(m)
	allFlags := cflags
	if ldflags != "" {
		if allFlags != "" {
			allFlags += " "
		}
		allFlags += ldflags
	}

	var edges strings.Builder
	objFiles := make([]string, 0, len(srcs))

	// Generate compile edges for each source file
	for _, src := range srcs {
		obj := strings.TrimSuffix(src, ".c")
		obj += ".o"
		objFiles = append(objFiles, obj)
		edges.WriteString(fmt.Sprintf("build %s: cc_compile %s\n flags = %s\n", obj, src, cflags))
	}

	// Generate link edge
	out := r.Outputs(m)[0]
	edges.WriteString(fmt.Sprintf("build %s: cc_link %s\n flags = %s\n", out, strings.Join(objFiles, " "), allFlags))

	return edges.String()
}

// cppLibrary implements the cpp_library rule
// Same as cc_library but uses g++ compiler

type cppLibrary struct{}

func (r *cppLibrary) Name() string {
	return "cpp_library"
}

func (r *cppLibrary) NinjaRule() string {
	return `rule cpp_compile
 command = g++ -c $in -o $out $flags

rule cpp_archive
 command = ar rcs $out $in
`
}

func (r *cppLibrary) Outputs(m *parser.Module) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	return []string{fmt.Sprintf("lib%s.a", name)}
}

func (r *cppLibrary) NinjaEdge(m *parser.Module) string {
	name := getName(m)
	srcs := getSrcs(m)
	if name == "" || len(srcs) == 0 {
		return ""
	}

	var edges strings.Builder
	objFiles := make([]string, 0, len(srcs))

	// Extract flags
	cflags := getCflags(m)
	cppflags := getCppflags(m)
	allFlags := cflags
	if cppflags != "" {
		if allFlags != "" {
			allFlags += " "
		}
		allFlags += cppflags
	}

	// Generate compile edges for each source file
	for _, src := range srcs {
		obj := strings.TrimSuffix(src, ".cpp")
		obj = strings.TrimSuffix(obj, ".cc")
		obj = strings.TrimSuffix(obj, ".cxx")
		obj += ".o"
		objFiles = append(objFiles, obj)
		edges.WriteString(fmt.Sprintf("build %s: cpp_compile %s\n flags = %s\n", obj, src, allFlags))
	}

	// Generate archive edge
	out := r.Outputs(m)[0]
	edges.WriteString(fmt.Sprintf("build %s: cpp_archive %s\n", out, strings.Join(objFiles, " ")))

	return edges.String()
}

// cppBinary implements the cpp_binary rule
// Same as cc_binary but uses g++ compiler

type cppBinary struct{}

func (r *cppBinary) Name() string {
	return "cpp_binary"
}

func (r *cppBinary) NinjaRule() string {
	return `rule cpp_link
 command = g++ -o $out $in $flags
`
}

func (r *cppBinary) Outputs(m *parser.Module) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	return []string{name}
}

func (r *cppBinary) NinjaEdge(m *parser.Module) string {
	name := getName(m)
	srcs := getSrcs(m)
	if name == "" || len(srcs) == 0 {
		return ""
	}

	cflags := getCflags(m)
	cppflags := getCppflags(m)
	ldflags := getLdflags(m)
	allFlags := cflags
	if cppflags != "" {
		if allFlags != "" {
			allFlags += " "
		}
		allFlags += cppflags
	}
	if ldflags != "" {
		if allFlags != "" {
			allFlags += " "
		}
		allFlags += ldflags
	}

	var edges strings.Builder
	objFiles := make([]string, 0, len(srcs))

	// Generate compile edges for each source file
	for _, src := range srcs {
		obj := strings.TrimSuffix(src, ".cpp")
		obj = strings.TrimSuffix(obj, ".cc")
		obj = strings.TrimSuffix(obj, ".cxx")
		obj += ".o"
		objFiles = append(objFiles, obj)
		edges.WriteString(fmt.Sprintf("build %s: cpp_compile %s\n flags = %s\n", obj, src, allFlags))
	}

	// Generate link edge
	out := r.Outputs(m)[0]
	edges.WriteString(fmt.Sprintf("build %s: cpp_link %s\n flags = %s\n", out, strings.Join(objFiles, " "), ldflags))

	return edges.String()
}

// goLibrary implements the go_library rule
type goLibrary struct{}

func (r *goLibrary) Name() string {
	return "go_library"
}

func (r *goLibrary) NinjaRule() string {
	return `rule go_build_archive
    command = go build -buildmode=archive -o $out $in
`
}

func (r *goLibrary) Outputs(m *parser.Module) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	return []string{fmt.Sprintf("%s.a", name)}
}

func (r *goLibrary) NinjaEdge(m *parser.Module) string {
	name := getName(m)
	srcs := getSrcs(m)
	if name == "" || len(srcs) == 0 {
		return ""
	}

	goflags := getGoflags(m)
	out := r.Outputs(m)[0]
	return fmt.Sprintf("build %s: go_build_archive %s\n flags = %s\n", out, formatSrcs(srcs), goflags)
}

// goBinary implements the go_binary rule
type goBinary struct{}

func (r *goBinary) Name() string {
	return "go_binary"
}

func (r *goBinary) NinjaRule() string {
	return `rule go_build
    command = go build -o $out $in
`
}

func (r *goBinary) Outputs(m *parser.Module) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	return []string{name}
}

func (r *goBinary) NinjaEdge(m *parser.Module) string {
	name := getName(m)
	srcs := getSrcs(m)
	if name == "" || len(srcs) == 0 {
		return ""
	}

	goflags := getGoflags(m)
	out := r.Outputs(m)[0]
	return fmt.Sprintf("build %s: go_build %s\n flags = %s\n", out, formatSrcs(srcs), goflags)
}

// javaLibrary implements the java_library rule
type javaLibrary struct{}

func (r *javaLibrary) Name() string {
	return "java_library"
}

func (r *javaLibrary) NinjaRule() string {
	return `rule javac_lib
 command = javac -d $outdir $in $flags

rule jar_create
 command = jar cf $out -C $outdir .
`
}

func (r *javaLibrary) Outputs(m *parser.Module) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	return []string{fmt.Sprintf("%s.jar", name)}
}

func (r *javaLibrary) NinjaEdge(m *parser.Module) string {
	name := getName(m)
	srcs := getSrcs(m)
	if name == "" || len(srcs) == 0 {
		return ""
	}

	javaflags := getJavaflags(m)
	out := r.Outputs(m)[0]
	outdir := name + "_classes"

	var edges strings.Builder

	// Generate compile edges for each source file (creates .class files in outdir)
	edges.WriteString(fmt.Sprintf("build %s.stamp: javac_lib %s\n outdir = %s\n flags = %s\n",
		name, formatSrcs(srcs), outdir, javaflags))

	// Generate jar edge (depends on compile output)
	edges.WriteString(fmt.Sprintf("build %s: jar_create %s.stamp\n outdir = %s\n",
		out, name, outdir))

	return edges.String()
}

// javaBinary implements the java_binary rule
type javaBinary struct{}

func (r *javaBinary) Name() string {
	return "java_binary"
}

func (r *javaBinary) NinjaRule() string {
	return `rule javac_bin
 command = javac -d $outdir $in $flags

rule jar_create_executable
 command = jar cfe $out $main_class -C $outdir .
`
}

func (r *javaBinary) Outputs(m *parser.Module) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	return []string{fmt.Sprintf("%s.jar", name)}
}

func (r *javaBinary) NinjaEdge(m *parser.Module) string {
	name := getName(m)
	srcs := getSrcs(m)
	mainClass := getStringProp(m, "main_class")
	if name == "" || len(srcs) == 0 || mainClass == "" {
		return ""
	}

	javaflags := getJavaflags(m)
	out := r.Outputs(m)[0]
	outdir := name + "_classes"

	var edges strings.Builder

	// Generate compile edges for each source file
	edges.WriteString(fmt.Sprintf("build %s.stamp: javac_bin %s\n outdir = %s\n flags = %s\n",
		name, formatSrcs(srcs), outdir, javaflags))

	// Generate jar edge (depends on compile output)
	edges.WriteString(fmt.Sprintf("build %s: jar_create_executable %s.stamp\n outdir = %s\n main_class = %s\n",
		out, name, outdir, mainClass))

	return edges.String()
}

// customRule implements the custom rule
type customRule struct{}

func (r *customRule) Name() string {
	return "custom"
}

func (r *customRule) NinjaRule() string {
	return `rule custom_command
    command = $cmd
`
}

func (r *customRule) Outputs(m *parser.Module) []string {
	outs := getListProp(m, "outs")
	return outs
}

func (r *customRule) NinjaEdge(m *parser.Module) string {
	srcs := getListProp(m, "srcs")
	outs := getListProp(m, "outs")
	cmd := getStringProp(m, "cmd")

	if len(outs) == 0 || cmd == "" {
		return ""
	}

	// Substitute $in and $out in command
	cmd = strings.ReplaceAll(cmd, "$in", "$in")
	cmd = strings.ReplaceAll(cmd, "$out", "$out")

	outStr := strings.Join(outs, " ")
	srcStr := formatSrcs(srcs)

	if srcStr == "" {
		return fmt.Sprintf("build %s: custom_command\n    cmd = %s\n", outStr, cmd)
	}

	return fmt.Sprintf("build %s: custom_command %s\n    cmd = %s\n", outStr, srcStr, cmd)
}

// GetAllRules returns all available rule implementations
func GetAllRules() []BuildRule {
	return []BuildRule{
		&ccLibrary{},
		&ccBinary{},
		&cppLibrary{},
		&cppBinary{},
		&goLibrary{},
		&goBinary{},
		&javaLibrary{},
		&javaBinary{},
		&customRule{},
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
