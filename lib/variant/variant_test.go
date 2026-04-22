package variant

import (
	"reflect"
	"testing"

	"minibp/lib/parser"
)

func makeModule(props ...*parser.Property) *parser.Module {
	return &parser.Module{
		Type: "cc_library",
		Map:  &parser.Map{Properties: props},
	}
}

func TestMergeMapPropsAppendsLists(t *testing.T) {
	m := makeModule(
		&parser.Property{Name: "srcs", Value: &parser.List{Values: []parser.Expression{
			&parser.String{Value: "a.c"},
		}}},
	)
	override := &parser.Map{Properties: []*parser.Property{
		{Name: "srcs", Value: &parser.List{Values: []parser.Expression{
			&parser.String{Value: "b.c"},
		}}},
	}}

	MergeMapProps(m, override)

	list := m.Map.Properties[0].Value.(*parser.List)
	if len(list.Values) != 2 {
		t.Fatalf("expected 2 srcs after merge, got %d", len(list.Values))
	}
	got := []string{list.Values[0].(*parser.String).Value, list.Values[1].(*parser.String).Value}
	want := []string{"a.c", "b.c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("expected %v, got %v", want, got)
	}
}

func TestMergeMapPropsOverridesScalars(t *testing.T) {
	m := makeModule(
		&parser.Property{Name: "name", Value: &parser.String{Value: "original"}},
	)
	override := &parser.Map{Properties: []*parser.Property{
		{Name: "name", Value: &parser.String{Value: "overridden"}},
	}}

	MergeMapProps(m, override)

	s := m.Map.Properties[0].Value.(*parser.String).Value
	if s != "overridden" {
		t.Errorf("expected name overridden, got %q", s)
	}
}

func TestMergeMapPropsAddsNewProperty(t *testing.T) {
	m := makeModule(
		&parser.Property{Name: "name", Value: &parser.String{Value: "lib"}},
	)
	override := &parser.Map{Properties: []*parser.Property{
		{Name: "cflags", Value: &parser.List{Values: []parser.Expression{
			&parser.String{Value: "-Wall"},
		}}},
	}}

	MergeMapProps(m, override)

	if len(m.Map.Properties) != 2 {
		t.Fatalf("expected 2 properties after merge, got %d", len(m.Map.Properties))
	}
	found := false
	for _, prop := range m.Map.Properties {
		if prop.Name == "cflags" {
			found = true
		}
	}
	if !found {
		t.Error("expected cflags property to be added from override")
	}
}

func TestMergeMapPropsNilOverride(t *testing.T) {
	m := makeModule(
		&parser.Property{Name: "name", Value: &parser.String{Value: "lib"}},
	)
	MergeMapProps(m, nil)
	if len(m.Map.Properties) != 1 {
		t.Error("expected unchanged module with nil override")
	}
}

func TestIsModuleEnabledForTargetBothFalse(t *testing.T) {
	m := makeModule(
		&parser.Property{Name: "name", Value: &parser.String{Value: "lib"}},
	)
	if !IsModuleEnabledForTarget(m, false) {
		t.Error("expected module enabled when neither host/device supported set")
	}
	if !IsModuleEnabledForTarget(m, true) {
		t.Error("expected module enabled for host when neither set")
	}
}

func TestIsModuleEnabledForTargetHostOnly(t *testing.T) {
	m := makeModule(
		&parser.Property{Name: "host_supported", Value: &parser.Bool{Value: true}},
	)
	if !IsModuleEnabledForTarget(m, true) {
		t.Error("expected host build enabled")
	}
	if IsModuleEnabledForTarget(m, false) {
		t.Error("expected device build disabled for host-only module")
	}
}

func TestIsModuleEnabledForTargetDeviceOnly(t *testing.T) {
	m := makeModule(
		&parser.Property{Name: "device_supported", Value: &parser.Bool{Value: true}},
	)
	if !IsModuleEnabledForTarget(m, false) {
		t.Error("expected device build enabled")
	}
	if IsModuleEnabledForTarget(m, true) {
		t.Error("expected host build disabled for device-only module")
	}
}

func TestIsModuleEnabledForTargetBothTrue(t *testing.T) {
	m := makeModule(
		&parser.Property{Name: "host_supported", Value: &parser.Bool{Value: true}},
		&parser.Property{Name: "device_supported", Value: &parser.Bool{Value: true}},
	)
	if !IsModuleEnabledForTarget(m, true) {
		t.Error("expected host build enabled")
	}
	if !IsModuleEnabledForTarget(m, false) {
		t.Error("expected device build enabled")
	}
}

func TestIsModuleEnabledForTargetBothFalseExplicit(t *testing.T) {
	m := makeModule(
		&parser.Property{Name: "host_supported", Value: &parser.Bool{Value: false}},
		&parser.Property{Name: "device_supported", Value: &parser.Bool{Value: false}},
	)
	if !IsModuleEnabledForTarget(m, true) {
		t.Error("current behavior: both-false defaults to enabled for host")
	}
	if !IsModuleEnabledForTarget(m, false) {
		t.Error("current behavior: both-false defaults to enabled for device")
	}
}

func TestGetBoolProp(t *testing.T) {
	m := makeModule(
		&parser.Property{Name: "host_supported", Value: &parser.Bool{Value: true}},
	)
	if !GetBoolProp(m, "host_supported") {
		t.Error("expected true")
	}
	if GetBoolProp(m, "missing") {
		t.Error("expected false for missing prop")
	}
}

func TestGetBoolPropNilMap(t *testing.T) {
	m := &parser.Module{Type: "phony"}
	if GetBoolProp(m, "anything") {
		t.Error("expected false for nil map")
	}
}

func TestMergeVariantPropsArch(t *testing.T) {
	m := &parser.Module{
		Type: "cc_library",
		Map: &parser.Map{Properties: []*parser.Property{
			{Name: "name", Value: &parser.String{Value: "lib"}},
			{Name: "cflags", Value: &parser.List{Values: []parser.Expression{
				&parser.String{Value: "-Wall"},
			}}},
		}},
		Arch: map[string]*parser.Map{
			"arm64": {Properties: []*parser.Property{
				{Name: "cflags", Value: &parser.List{Values: []parser.Expression{
					&parser.String{Value: "-DARM64"},
				}}},
			}},
		},
	}

	eval := parser.NewEvaluator()
	MergeVariantProps(m, "arm64", false, eval)

	cflagsProp := m.Map.Properties[1]
	list, ok := cflagsProp.Value.(*parser.List)
	if !ok {
		t.Fatalf("expected cflags to remain a list")
	}
	if len(list.Values) != 2 {
		t.Fatalf("expected 2 cflags after arch merge, got %d", len(list.Values))
	}
	got := list.Values[1].(*parser.String).Value
	if got != "-DARM64" {
		t.Errorf("expected -DARM64 appended, got %q", got)
	}
}

func TestMergeVariantPropsHost(t *testing.T) {
	m := &parser.Module{
		Type: "cc_library",
		Map: &parser.Map{Properties: []*parser.Property{
			{Name: "name", Value: &parser.String{Value: "lib"}},
			{Name: "cflags", Value: &parser.List{Values: []parser.Expression{
				&parser.String{Value: "-Wall"},
			}}},
		}},
		Host: &parser.Map{Properties: []*parser.Property{
			{Name: "cflags", Value: &parser.List{Values: []parser.Expression{
				&parser.String{Value: "-DHOST"},
			}}},
		}},
	}

	eval := parser.NewEvaluator()
	MergeVariantProps(m, "", true, eval)

	cflagsProp := m.Map.Properties[1]
	list := cflagsProp.Value.(*parser.List)
	if len(list.Values) != 2 {
		t.Fatalf("expected 2 cflags after host merge, got %d", len(list.Values))
	}
	got := list.Values[1].(*parser.String).Value
	if got != "-DHOST" {
		t.Errorf("expected -DHOST appended, got %q", got)
	}
}

func TestMergeVariantPropsTarget(t *testing.T) {
	m := &parser.Module{
		Type: "cc_library",
		Map: &parser.Map{Properties: []*parser.Property{
			{Name: "name", Value: &parser.String{Value: "lib"}},
			{Name: "cflags", Value: &parser.List{Values: []parser.Expression{
				&parser.String{Value: "-Wall"},
			}}},
		}},
		Target: &parser.Map{Properties: []*parser.Property{
			{Name: "cflags", Value: &parser.List{Values: []parser.Expression{
				&parser.String{Value: "-DDEVICE"},
			}}},
		}},
	}

	eval := parser.NewEvaluator()
	MergeVariantProps(m, "", false, eval)

	cflagsProp := m.Map.Properties[1]
	list := cflagsProp.Value.(*parser.List)
	if len(list.Values) != 2 {
		t.Fatalf("expected 2 cflags after target merge, got %d", len(list.Values))
	}
	got := list.Values[1].(*parser.String).Value
	if got != "-DDEVICE" {
		t.Errorf("expected -DDEVICE appended, got %q", got)
	}
}

func TestMergeVariantPropsMultilibLib64(t *testing.T) {
	m := &parser.Module{
		Type: "cc_library",
		Map: &parser.Map{Properties: []*parser.Property{
			{Name: "name", Value: &parser.String{Value: "lib"}},
		}},
		Multilib: map[string]*parser.Map{
			"lib64": {Properties: []*parser.Property{
				{Name: "cflags", Value: &parser.List{Values: []parser.Expression{
					&parser.String{Value: "-D64"},
				}}},
			}},
		},
	}

	eval := parser.NewEvaluator()
	MergeVariantProps(m, "arm64", false, eval)

	found := false
	for _, prop := range m.Map.Properties {
		if prop.Name == "cflags" {
			found = true
			list := prop.Value.(*parser.List)
			if len(list.Values) != 1 {
				t.Errorf("expected 1 cflag from multilib, got %d", len(list.Values))
			}
			if list.Values[0].(*parser.String).Value != "-D64" {
				t.Errorf("expected -D64, got %v", list.Values[0])
			}
		}
	}
	if !found {
		t.Error("expected cflags from multilib lib64 to be merged for arm64")
	}
}

func TestMergeVariantPropsMultilibLib32(t *testing.T) {
	m := &parser.Module{
		Type: "cc_library",
		Map: &parser.Map{Properties: []*parser.Property{
			{Name: "name", Value: &parser.String{Value: "lib"}},
		}},
		Multilib: map[string]*parser.Map{
			"lib32": {Properties: []*parser.Property{
				{Name: "cflags", Value: &parser.List{Values: []parser.Expression{
					&parser.String{Value: "-D32"},
				}}},
			}},
		},
	}

	eval := parser.NewEvaluator()
	MergeVariantProps(m, "arm", false, eval)

	found := false
	for _, prop := range m.Map.Properties {
		if prop.Name == "cflags" {
			found = true
			list := prop.Value.(*parser.List)
			if list.Values[0].(*parser.String).Value != "-D32" {
				t.Errorf("expected -D32, got %v", list.Values[0])
			}
		}
	}
	if !found {
		t.Error("expected cflags from multilib lib32 to be merged for arm")
	}
}

func TestMergeVariantPropsMultilibNoMatch(t *testing.T) {
	m := &parser.Module{
		Type: "cc_library",
		Map: &parser.Map{Properties: []*parser.Property{
			{Name: "name", Value: &parser.String{Value: "lib"}},
		}},
		Multilib: map[string]*parser.Map{
			"lib64": {Properties: []*parser.Property{
				{Name: "cflags", Value: &parser.List{Values: []parser.Expression{
					&parser.String{Value: "-D64"},
				}}},
			}},
		},
	}

	eval := parser.NewEvaluator()
	MergeVariantProps(m, "arm", false, eval)

	for _, prop := range m.Map.Properties {
		if prop.Name == "cflags" {
			t.Error("expected no cflags merge for lib64 on arm arch")
		}
	}
}

func TestMergeVariantPropsNoOverrides(t *testing.T) {
	m := makeModule(
		&parser.Property{Name: "name", Value: &parser.String{Value: "lib"}},
	)
	eval := parser.NewEvaluator()
	MergeVariantProps(m, "", false, eval)
	if len(m.Map.Properties) != 1 {
		t.Error("expected unchanged module with no overrides")
	}
}
