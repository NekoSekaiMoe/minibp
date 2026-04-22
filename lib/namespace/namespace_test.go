package namespace

import (
	"reflect"
	"testing"

	"minibp/lib/parser"
)

func TestBuildMap(t *testing.T) {
	modules := map[string]*parser.Module{
		"vendor": {
			Type: "soong_namespace",
			Map: &parser.Map{Properties: []*parser.Property{
				{Name: "name", Value: &parser.String{Value: "vendor"}},
				{Name: "imports", Value: &parser.List{Values: []parser.Expression{
					&parser.String{Value: "core"},
					&parser.String{Value: "common"},
				}}},
			}},
		},
		"app": {
			Type: "cc_binary",
			Map: &parser.Map{Properties: []*parser.Property{
				{Name: "name", Value: &parser.String{Value: "app"}},
			}},
		},
	}

	result := BuildMap(modules, func(m *parser.Module, key string) string {
		for _, prop := range m.Map.Properties {
			if prop.Name == key {
				if s, ok := prop.Value.(*parser.String); ok {
					return s.Value
				}
			}
		}
		return ""
	})

	if len(result) != 1 {
		t.Fatalf("expected 1 namespace, got %d", len(result))
	}
	ns := result["vendor"]
	if ns == nil {
		t.Fatal("expected vendor namespace")
	}
	want := []string{"core", "common"}
	if !reflect.DeepEqual(ns.Imports, want) {
		t.Errorf("expected imports %v, got %v", want, ns.Imports)
	}
}

func TestBuildMapSkipsNonNamespaceModules(t *testing.T) {
	modules := map[string]*parser.Module{
		"app": {
			Type: "cc_binary",
			Map: &parser.Map{Properties: []*parser.Property{
				{Name: "name", Value: &parser.String{Value: "app"}},
			}},
		},
	}

	result := BuildMap(modules, func(m *parser.Module, key string) string {
		for _, prop := range m.Map.Properties {
			if prop.Name == key {
				if s, ok := prop.Value.(*parser.String); ok {
					return s.Value
				}
			}
		}
		return ""
	})

	if len(result) != 0 {
		t.Errorf("expected 0 namespaces for non-namespace modules, got %d", len(result))
	}
}

func TestBuildMapSkipsUnnamedNamespace(t *testing.T) {
	modules := map[string]*parser.Module{
		"anon": {
			Type: "soong_namespace",
			Map:  &parser.Map{Properties: []*parser.Property{}},
		},
	}

	result := BuildMap(modules, func(m *parser.Module, key string) string {
		return ""
	})

	if len(result) != 0 {
		t.Errorf("expected 0 namespaces for unnamed namespace, got %d", len(result))
	}
}

func TestResolveModuleRefSimpleColon(t *testing.T) {
	namespaces := map[string]*Info{}
	got := ResolveModuleRef(":libfoo", namespaces)
	if got != "libfoo" {
		t.Errorf("ResolveModuleRef(:libfoo) = %q, want %q", got, "libfoo")
	}
}

func TestResolveModuleRefNamespaceRef(t *testing.T) {
	namespaces := map[string]*Info{
		"vendor": {Imports: []string{"core"}},
	}
	got := ResolveModuleRef("//vendor:libfoo", namespaces)
	if got != "libfoo" {
		t.Errorf("ResolveModuleRef(//vendor:libfoo) = %q, want %q", got, "libfoo")
	}
}

func TestResolveModuleRefUnknownNamespace(t *testing.T) {
	namespaces := map[string]*Info{}
	got := ResolveModuleRef("//unknown:libfoo", namespaces)
	if got != "//unknown:libfoo" {
		t.Errorf("ResolveModuleRef(//unknown:libfoo) = %q, want %q", got, "//unknown:libfoo")
	}
}

func TestResolveModuleRefNoPrefix(t *testing.T) {
	namespaces := map[string]*Info{}
	got := ResolveModuleRef("libfoo", namespaces)
	if got != "libfoo" {
		t.Errorf("ResolveModuleRef(libfoo) = %q, want %q", got, "libfoo")
	}
}

func TestApplyOverrides(t *testing.T) {
	base := &parser.Module{
		Type: "cc_library",
		Map: &parser.Map{Properties: []*parser.Property{
			{Name: "name", Value: &parser.String{Value: "lib"}},
			{Name: "cflags", Value: &parser.List{Values: []parser.Expression{
				&parser.String{Value: "-Wall"},
			}}},
		}},
	}
	ovr := &parser.Module{
		Type:     "cc_library",
		Override: true,
		Map: &parser.Map{Properties: []*parser.Property{
			{Name: "name", Value: &parser.String{Value: "lib"}},
			{Name: "cflags", Value: &parser.List{Values: []parser.Expression{
				&parser.String{Value: "-O2"},
			}}},
		}},
	}

	modules := map[string]*parser.Module{"lib": base, "lib_override": ovr}
	ApplyOverrides(modules)

	mod := modules["lib"]
	if mod != base {
		t.Error("expected base module to remain after ApplyOverrides")
	}
}

func TestApplyOverridesNoOverrideModules(t *testing.T) {
	modules := map[string]*parser.Module{
		"app": {
			Type: "cc_binary",
			Map: &parser.Map{Properties: []*parser.Property{
				{Name: "name", Value: &parser.String{Value: "app"}},
			}},
		},
	}
	ApplyOverrides(modules)
	if len(modules) != 1 {
		t.Errorf("expected unchanged module count, got %d", len(modules))
	}
}

func TestApplySoongConfigModuleTypes(t *testing.T) {
	eval := parser.NewEvaluator()
	modules := map[string]*parser.Module{
		"acme_cc": {
			Type: "soong_config_module_type",
			Map: &parser.Map{Properties: []*parser.Property{
				{Name: "name", Value: &parser.String{Value: "custom_acme_cc"}},
				{Name: "module_type", Value: &parser.String{Value: "custom"}},
				{Name: "config_namespace", Value: &parser.String{Value: "acme"}},
				{Name: "vars", Value: &parser.Map{Properties: []*parser.Property{
					{Name: "board", Value: &parser.String{Value: "soc_a"}},
				}}},
			}},
		},
	}

	getStr := func(m *parser.Module, key string) string {
		if m.Map == nil {
			return ""
		}
		for _, prop := range m.Map.Properties {
			if prop.Name == key {
				if s, ok := prop.Value.(*parser.String); ok {
					return s.Value
				}
			}
		}
		return ""
	}

	ApplySoongConfigModuleTypes(modules, getStr, eval)

	sel := &parser.Select{
		Conditions: []parser.ConfigurableCondition{
			{FunctionName: "soong_config_variable",
				Args: []parser.Expression{&parser.String{Value: "acme"}, &parser.String{Value: "board"}}},
		},
		Cases: []parser.SelectCase{
			{
				Patterns: []parser.SelectPattern{{Value: &parser.String{Value: "soc_a"}}},
				Value:    &parser.List{Values: []parser.Expression{&parser.String{Value: "soc_a.c"}}},
			},
			{
				Patterns: []parser.SelectPattern{{Value: &parser.Variable{Name: "default"}}},
				Value:    &parser.List{Values: []parser.Expression{&parser.String{Value: "generic.c"}}},
			},
		},
	}
	result := eval.Eval(sel)
	list, ok := result.([]interface{})
	if !ok || len(list) != 1 {
		t.Fatalf("expected []interface{} with 1 item, got %v", result)
	}
	if list[0] != "soc_a.c" {
		t.Errorf("expected soong_config_variable to match soc_a via ApplySoongConfigModuleTypes, got %v", list[0])
	}
}
