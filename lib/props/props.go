package props

import "minibp/lib/parser"

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

func GetStringPropEval(m *parser.Module, name string, eval *parser.Evaluator) string {
	if m.Map == nil {
		return ""
	}
	for _, prop := range m.Map.Properties {
		if prop.Name == name {
			if s, ok := prop.Value.(*parser.String); ok {
				if eval != nil {
					val := eval.Eval(prop.Value)
					if s, ok := val.(string); ok {
						return s
					}
				} else {
					return s.Value
				}
			}
		}
	}
	return ""
}

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

func GetBoolProp(m *parser.Module, name string, eval *parser.Evaluator) bool {
	if m.Map == nil {
		return false
	}
	for _, prop := range m.Map.Properties {
		if prop.Name == name {
			if b, ok := prop.Value.(*parser.Bool); ok {
				return b.Value
			}
			if eval != nil {
				val := eval.Eval(prop.Value)
				if b, ok := val.(bool); ok {
					return b
				}
			}
		}
	}
	return false
}
