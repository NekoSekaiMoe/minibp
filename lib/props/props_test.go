package props

import (
	"reflect"
	"testing"

	"minibp/lib/parser"
)

func makeModule(props ...*parser.Property) *parser.Module {
	return &parser.Module{
		Type: "cc_binary",
		Map:  &parser.Map{Properties: props},
	}
}

func TestGetStringProp(t *testing.T) {
	m := makeModule(
		&parser.Property{Name: "name", Value: &parser.String{Value: "app"}},
		&parser.Property{Name: "srcs", Value: &parser.List{Values: []parser.Expression{&parser.String{Value: "main.c"}}}},
	)
	if got := GetStringProp(m, "name"); got != "app" {
		t.Errorf("GetStringProp(name) = %q, want %q", got, "app")
	}
	if got := GetStringProp(m, "missing"); got != "" {
		t.Errorf("GetStringProp(missing) = %q, want empty", got)
	}
	if got := GetStringProp(m, "srcs"); got != "" {
		t.Errorf("GetStringProp(srcs) = %q, want empty for non-string prop", got)
	}
}

func TestGetStringPropNilMap(t *testing.T) {
	m := &parser.Module{Type: "phony"}
	if got := GetStringProp(m, "name"); got != "" {
		t.Errorf("expected empty for nil map, got %q", got)
	}
}

func TestGetStringPropEval(t *testing.T) {
	eval := parser.NewEvaluator()
	eval.SetVar("myname", "resolved")
	m := makeModule(
		&parser.Property{Name: "name", Value: &parser.String{Value: "app"}},
	)
	if got := GetStringPropEval(m, "name", eval); got != "app" {
		t.Errorf("GetStringPropEval(name) = %q, want %q", got, "app")
	}
}

func TestGetStringPropEvalWithVariable(t *testing.T) {
	eval := parser.NewEvaluator()
	eval.SetVar("myname", "resolved")
	m := makeModule(
		&parser.Property{Name: "name", Value: &parser.Variable{Name: "myname"}},
	)
	// GetStringPropEval only handles *parser.String directly; Variable values
	// are not resolved by this function (they require EvalModule first).
	// Verify the known behavior: non-String value returns empty.
	if got := GetStringPropEval(m, "name", eval); got != "" {
		t.Errorf("GetStringPropEval(name) = %q, want empty for Variable prop", got)
	}
}

func TestGetStringPropEvalNil(t *testing.T) {
	m := makeModule(
		&parser.Property{Name: "name", Value: &parser.String{Value: "app"}},
	)
	if got := GetStringPropEval(m, "name", nil); got != "app" {
		t.Errorf("GetStringPropEval(name, nil) = %q, want %q", got, "app")
	}
}

func TestGetListProp(t *testing.T) {
	m := makeModule(
		&parser.Property{Name: "srcs", Value: &parser.List{Values: []parser.Expression{
			&parser.String{Value: "a.c"},
			&parser.String{Value: "b.c"},
		}}},
	)
	got := GetListProp(m, "srcs")
	want := []string{"a.c", "b.c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetListProp(srcs) = %v, want %v", got, want)
	}
}

func TestGetListPropMissing(t *testing.T) {
	m := makeModule(
		&parser.Property{Name: "name", Value: &parser.String{Value: "app"}},
	)
	if got := GetListProp(m, "srcs"); got != nil {
		t.Errorf("GetListProp(srcs) = %v, want nil", got)
	}
}

func TestGetListPropNilMap(t *testing.T) {
	m := &parser.Module{Type: "phony"}
	if got := GetListProp(m, "srcs"); got != nil {
		t.Errorf("expected nil for nil map, got %v", got)
	}
}

func TestGetListPropEval(t *testing.T) {
	eval := parser.NewEvaluator()
	eval.SetVar("extra", "extra.c")
	m := makeModule(
		&parser.Property{Name: "srcs", Value: &parser.List{Values: []parser.Expression{
			&parser.String{Value: "a.c"},
			&parser.Variable{Name: "extra"},
		}}},
	)
	got := GetListPropEval(m, "srcs", eval)
	want := []string{"a.c", "extra.c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetListPropEval(srcs) = %v, want %v", got, want)
	}
}

func TestGetListPropEvalNil(t *testing.T) {
	m := makeModule(
		&parser.Property{Name: "srcs", Value: &parser.List{Values: []parser.Expression{
			&parser.String{Value: "a.c"},
		}}},
	)
	got := GetListPropEval(m, "srcs", nil)
	want := []string{"a.c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetListPropEval(srcs, nil) = %v, want %v", got, want)
	}
}

func TestGetBoolProp(t *testing.T) {
	m := makeModule(
		&parser.Property{Name: "host_supported", Value: &parser.Bool{Value: true}},
		&parser.Property{Name: "device_supported", Value: &parser.Bool{Value: false}},
	)
	if !GetBoolProp(m, "host_supported", nil) {
		t.Error("expected host_supported true")
	}
	if GetBoolProp(m, "device_supported", nil) {
		t.Error("expected device_supported false")
	}
	if GetBoolProp(m, "missing", nil) {
		t.Error("expected missing prop to return false")
	}
}

func TestGetBoolPropNilMap(t *testing.T) {
	m := &parser.Module{Type: "phony"}
	if GetBoolProp(m, "host_supported", nil) {
		t.Error("expected false for nil map")
	}
}

func TestGetBoolPropNonBoolValue(t *testing.T) {
	m := makeModule(
		&parser.Property{Name: "host_supported", Value: &parser.String{Value: "true"}},
	)
	if GetBoolProp(m, "host_supported", nil) {
		t.Error("expected false for non-bool value without eval")
	}
}

func TestGetBoolPropWithEval(t *testing.T) {
	eval := parser.NewEvaluator()
	eval.SetVar("enabled", true)
	m := makeModule(
		&parser.Property{Name: "host_supported", Value: &parser.Variable{Name: "enabled"}},
	)
	if !GetBoolProp(m, "host_supported", eval) {
		t.Error("expected host_supported true via eval")
	}
}
