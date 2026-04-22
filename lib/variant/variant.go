package variant

import "minibp/lib/parser"

func MergeVariantProps(m *parser.Module, arch string, host bool, eval *parser.Evaluator) {
	if arch != "" && m.Arch != nil {
		MergeMapProps(m, m.Arch[arch])
	}
	if host && m.Host != nil {
		MergeMapProps(m, m.Host)
	}
	if !host && m.Target != nil {
		MergeMapProps(m, m.Target)
	}
	if len(m.Multilib) > 0 {
		mergeMultilib(m, arch)
	}
}

func mergeMultilib(m *parser.Module, arch string) {
	for abi, mlMap := range m.Multilib {
		switch {
		case abi == "lib32" && (arch == "x86" || arch == "arm"):
			MergeMapProps(m, mlMap)
		case abi == "lib64" && (arch == "x86_64" || arch == "arm64"):
			MergeMapProps(m, mlMap)
		case abi == arch:
			MergeMapProps(m, mlMap)
		}
	}
}

func MergeMapProps(m *parser.Module, override *parser.Map) {
	if override == nil {
		return
	}
	for _, prop := range override.Properties {
		switch prop.Value.(type) {
		case *parser.List:
			merged := false
			for _, baseProp := range m.Map.Properties {
				if baseProp.Name == prop.Name {
					if baseList, ok := baseProp.Value.(*parser.List); ok {
						if archList, ok := prop.Value.(*parser.List); ok {
							baseList.Values = append(baseList.Values, archList.Values...)
						}
					}
					merged = true
					break
				}
			}
			if !merged {
				m.Map.Properties = append(m.Map.Properties, prop)
			}
		default:
			found := false
			for i, baseProp := range m.Map.Properties {
				if baseProp.Name == prop.Name {
					m.Map.Properties[i].Value = prop.Value
					found = true
					break
				}
			}
			if !found {
				m.Map.Properties = append(m.Map.Properties, prop)
			}
		}
	}
}

func IsModuleEnabledForTarget(m *parser.Module, hostBuild bool) bool {
	hs := GetBoolProp(m, "host_supported")
	ds := GetBoolProp(m, "device_supported")
	if !hs && !ds {
		return true
	}
	if hostBuild {
		return hs
	}
	return ds
}

func GetBoolProp(m *parser.Module, name string) bool {
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
