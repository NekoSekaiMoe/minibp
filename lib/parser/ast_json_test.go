package parser

import (
	"encoding/json"
	"strings"
	"testing"
	"text/scanner"
)

// TestModuleMarshalUnmarshal tests Module JSON serialization round-trip.
func TestModuleMarshalUnmarshal(t *testing.T) {
	original := &Module{
		Type:    "cc_library",
		TypePos: scanner.Position{Filename: "test.bp", Line: 1, Column: 1},
		Map: &Map{
			LBracePos: scanner.Position{Filename: "test.bp", Line: 1, Column: 12},
			RBracePos: scanner.Position{Filename: "test.bp", Line: 5, Column: 1},
			Properties: []*Property{
				{Name: "name", Value: &String{Value: "testlib"}},
				{Name: "srcs", Value: &List{Values: []Expression{&String{Value: "test.c"}}}},
			},
		},
		Override: true,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Check that type field is present (the custom "type" field)
	if !strings.Contains(string(data), `"type":"cc_library"`) {
		t.Errorf("expected type in JSON, got: %s", string(data))
	}

	var restored Module
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if restored.Type != original.Type {
		t.Errorf("Type: got %s, want %s", restored.Type, original.Type)
	}
	if restored.Override != original.Override {
		t.Errorf("Override: got %v, want %v", restored.Override, original.Override)
	}
	if restored.Map == nil {
		t.Fatal("Map is nil after unmarshal")
	}
	if len(restored.Map.Properties) != 2 {
		t.Errorf("Properties count: got %d, want 2", len(restored.Map.Properties))
	}
}

// TestSelectMarshalUnmarshal tests Select expression JSON serialization.
func TestSelectMarshalUnmarshal(t *testing.T) {
	original := &Select{
		KeywordPos: scanner.Position{Filename: "test.bp", Line: 3, Column: 5},
		Conditions: []ConfigurableCondition{
			{FunctionName: "arch", Position: scanner.Position{Filename: "test.bp", Line: 3, Column: 12}},
		},
		LBracePos: scanner.Position{Filename: "test.bp", Line: 3, Column: 20},
		RBracePos: scanner.Position{Filename: "test.bp", Line: 8, Column: 1},
		Cases: []SelectCase{
			{
				Patterns: []SelectPattern{{Value: &String{Value: "arm64"}}},
				ColonPos: scanner.Position{Filename: "test.bp", Line: 4, Column: 10},
				Value:    &List{Values: []Expression{&String{Value: "arm64_src.c"}}},
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var restored Select
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(restored.Cases) != 1 {
		t.Fatalf("Cases count: got %d, want 1", len(restored.Cases))
	}
	if len(restored.Conditions) != 1 {
		t.Fatalf("Conditions count: got %d, want 1", len(restored.Conditions))
	}
}

// TestOperatorMarshalUnmarshal tests Operator expression JSON serialization.
func TestOperatorMarshalUnmarshal(t *testing.T) {
	original := &Operator{
		Operator:    '+',
		OperatorPos: scanner.Position{Filename: "test.bp", Line: 2, Column: 10},
		Args: [2]Expression{
			&String{Value: "hello "},
			&String{Value: "world"},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var restored Operator
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if restored.Operator != '+' {
		t.Errorf("Operator: got %c, want +", restored.Operator)
	}
	if len(restored.Args) != 2 {
		t.Fatalf("Args count: got %d, want 2", len(restored.Args))
	}
}

// TestFileMarshalUnmarshal tests File JSON serialization.
func TestFileMarshalUnmarshal(t *testing.T) {
	original := &File{
		Name: "Android.bp",
		Defs: []Definition{
			&Module{
				Type: "cc_binary",
				Map: &Map{
					Properties: []*Property{
						{Name: "name", Value: &String{Value: "myapp"}},
					},
				},
			},
			&Assignment{
				Name:     "common_flags",
				Assigner: "=",
				Value:    &List{Values: []Expression{&String{Value: "-Wall"}}},
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Check that type field is present for each definition
	jsonStr := string(data)
	if !strings.Contains(jsonStr, `"type":"cc_binary"`) {
		t.Errorf("expected module type in JSON")
	}

	var restored File
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if restored.Name != original.Name {
		t.Errorf("Name: got %s, want %s", restored.Name, original.Name)
	}
	if len(restored.Defs) != 2 {
		t.Fatalf("Defs count: got %d, want 2", len(restored.Defs))
	}
}

// TestUnsetMarshalUnmarshal tests Unset expression JSON serialization.
func TestUnsetMarshalUnmarshal(t *testing.T) {
	original := &Unset{
		KeywordPos: scanner.Position{Filename: "test.bp", Line: 5, Column: 3},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var restored Unset
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if restored.KeywordPos.Line != 5 {
		t.Errorf("Line: got %d, want 5", restored.KeywordPos.Line)
	}
}

// TestVariableMarshalUnmarshal tests Variable expression JSON serialization.
func TestVariableMarshalUnmarshal(t *testing.T) {
	original := &Variable{
		Name:    "my_var",
		NamePos: scanner.Position{Filename: "test.bp", Line: 1, Column: 1},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var restored Variable
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if restored.Name != "my_var" {
		t.Errorf("Name: got %s, want my_var", restored.Name)
	}
}

// TestListMarshalUnmarshal tests List expression JSON serialization.
func TestListMarshalUnmarshal(t *testing.T) {
	original := &List{
		LBracePos: scanner.Position{Filename: "test.bp", Line: 2, Column: 8},
		RBracePos: scanner.Position{Filename: "test.bp", Line: 2, Column: 30},
		Values: []Expression{
			&String{Value: "a.c"},
			&String{Value: "b.c"},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var restored List
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(restored.Values) != 2 {
		t.Fatalf("Values count: got %d, want 2", len(restored.Values))
	}
}

// TestNestedSelectMarshalUnmarshal tests nested Select expressions.
func TestNestedSelectMarshalUnmarshal(t *testing.T) {
	original := &Select{
		Conditions: []ConfigurableCondition{
			{FunctionName: "arch"},
		},
		Cases: []SelectCase{
			{
				Patterns: []SelectPattern{{Value: &String{Value: "arm64"}}},
				Value: &Select{
					Conditions: []ConfigurableCondition{
						{FunctionName: "os"},
					},
					Cases: []SelectCase{
						{
							Patterns: []SelectPattern{{Value: &String{Value: "linux"}}},
							Value:    &String{Value: "arm64_linux.c"},
						},
					},
				},
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var restored Select
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Check nested select
	innerSelect, ok := restored.Cases[0].Value.(*Select)
	if !ok {
		t.Fatal("expected nested Select")
	}
	if len(innerSelect.Cases) != 1 {
		t.Fatalf("inner Select cases: got %d, want 1", len(innerSelect.Cases))
	}
}

// TestPosToStringAndBack tests position string conversion round-trip.
func TestPosToStringAndBack(t *testing.T) {
	original := scanner.Position{
		Filename: "test.bp",
		Line:     10,
		Column:   5,
	}

	str := posToString(original)
	if str == "" {
		t.Fatal("posToString returned empty string")
	}

	restored := stringToPos(str)
	if restored.Filename != original.Filename {
		t.Errorf("Filename: got %s, want %s", restored.Filename, original.Filename)
	}
	if restored.Line != original.Line {
		t.Errorf("Line: got %d, want %d", restored.Line, original.Line)
	}
	if restored.Column != original.Column {
		t.Errorf("Column: got %d, want %d", restored.Column, original.Column)
	}
}
