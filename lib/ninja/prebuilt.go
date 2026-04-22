package ninja

import (
	"fmt"
	"minibp/lib/parser"
	"path/filepath"
	"strings"
)

type prebuiltEtcRule struct {
	typeName string
	subdir   string
}

func (r *prebuiltEtcRule) Name() string { return r.typeName }

func (r *prebuiltEtcRule) NinjaRule(ctx RuleRenderContext) string {
	return "rule prebuilt_copy\n command = " + copyCommand() + "\n"
}

func (r *prebuiltEtcRule) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	src := getFirstSource(m)
	if src == "" {
		return nil
	}
	filename := GetStringProp(m, "filename")
	if filename == "" {
		filename = filepath.Base(src)
	}
	out := filename
	if r.subdir != "" {
		out = filepath.Join(r.subdir, filename)
	}
	return []string{filepath.ToSlash(out)}
}

func (r *prebuiltEtcRule) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string {
	src := getFirstSource(m)
	outs := r.Outputs(m, ctx)
	if src == "" || len(outs) == 0 {
		return ""
	}
	return fmt.Sprintf("build %s: prebuilt_copy %s\n", ninjaEscapePath(outs[0]), ninjaEscapePath(src))
}

func (r *prebuiltEtcRule) Desc(m *parser.Module, srcFile string) string { return "cp" }

type prebuiltBinaryRule struct {
	typeName string
}

func (r *prebuiltBinaryRule) Name() string { return r.typeName }

func (r *prebuiltBinaryRule) NinjaRule(ctx RuleRenderContext) string {
	return "rule prebuilt_binary_copy\n command = " + copyCommand() + " && chmod +x $out\n"
}

func (r *prebuiltBinaryRule) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	src := getFirstSource(m)
	if name == "" || src == "" {
		return nil
	}
	stem := GetStringProp(m, "stem")
	if stem == "" {
		stem = name
	}
	return []string{stem + ctx.ArchSuffix}
}

func (r *prebuiltBinaryRule) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string {
	src := getFirstSource(m)
	outs := r.Outputs(m, ctx)
	if src == "" || len(outs) == 0 {
		return ""
	}
	return fmt.Sprintf("build %s: prebuilt_binary_copy %s\n", ninjaEscapePath(outs[0]), ninjaEscapePath(src))
}

func (r *prebuiltBinaryRule) Desc(m *parser.Module, srcFile string) string { return "cp" }

type prebuiltLibraryRule struct {
	typeName string
	ext      string
}

func (r *prebuiltLibraryRule) Name() string { return r.typeName }

func (r *prebuiltLibraryRule) NinjaRule(ctx RuleRenderContext) string {
	return "rule prebuilt_library_copy\n command = " + copyCommand() + "\n"
}

func (r *prebuiltLibraryRule) Outputs(m *parser.Module, ctx RuleRenderContext) []string {
	name := getName(m)
	src := getFirstSource(m)
	if name == "" || src == "" {
		return nil
	}
	stem := GetStringProp(m, "stem")
	if stem == "" {
		stem = "lib" + name
	}
	if !strings.HasSuffix(stem, r.ext) {
		stem += ctx.ArchSuffix + r.ext
	}
	return []string{stem}
}

func (r *prebuiltLibraryRule) NinjaEdge(m *parser.Module, ctx RuleRenderContext) string {
	src := getFirstSource(m)
	outs := r.Outputs(m, ctx)
	if src == "" || len(outs) == 0 {
		return ""
	}
	return fmt.Sprintf("build %s: prebuilt_library_copy %s\n", ninjaEscapePath(outs[0]), ninjaEscapePath(src))
}

func (r *prebuiltLibraryRule) Desc(m *parser.Module, srcFile string) string { return "cp" }
