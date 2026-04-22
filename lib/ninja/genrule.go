package ninja

import (
	"fmt"
	"minibp/lib/parser"
	"strings"
)

type genrule struct{}

func (r *genrule) Name() string { return "genrule" }

func (r *genrule) NinjaRule(ctx RuleRenderContext) string {
	return `rule genrule_command
 command = $cmd
 description = Genrule $out
 restat = 1
`
}

func (r *genrule) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	if name == "" {
		return nil
	}
	outs := GetListProp(m, "outs")
	if len(outs) > 0 {
		return outs
	}
	return []string{name + ".out"}
}

func (r *genrule) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string {
	name := getName(m)
	srcs := getSrcs(m)
	cmd := GetStringProp(m, "cmd")
	if name == "" || cmd == "" {
		return ""
	}

	outs := r.Outputs(m, ctx)
	if len(outs) == 0 {
		return ""
	}

	toolFiles := GetListProp(m, "tool_files")
	deps := GetListProp(m, "deps")
	data := getData(m)

	var allDeps []string
	allDeps = append(allDeps, deps...)
	allDeps = append(allDeps, toolFiles...)
	allDeps = append(allDeps, data...)

	var edges strings.Builder
	escapedOuts := make([]string, 0, len(outs))
	for _, out := range outs {
		escapedOuts = append(escapedOuts, ninjaEscapePath(out))
	}
	edges.WriteString(fmt.Sprintf("build %s: genrule_command %s", strings.Join(escapedOuts, " "), strings.Join(srcs, " ")))
	if len(allDeps) > 0 {
		edges.WriteString(fmt.Sprintf(" | %s", strings.Join(allDeps, " ")))
	}
	edges.WriteString("\n")
	edges.WriteString(fmt.Sprintf(" cmd = %s\n", cmd))
	return edges.String()
}

func (r *genrule) Desc(m *parser.Module, srcFile string) string {
	return "genrule"
}
