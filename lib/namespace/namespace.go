package namespace

import (
	"fmt"
	"minibp/lib/module"
	"minibp/lib/parser"
	"minibp/lib/variant"
	"strings"
)

type Info struct {
	Imports []string
}

func BuildMap(modules map[string]*parser.Module, getStringProp func(*parser.Module, string) string) map[string]*Info {
	result := make(map[string]*Info)
	for _, mod := range modules {
		if mod.Type != "soong_namespace" || mod.Map == nil {
			continue
		}
		name := getStringProp(mod, "name")
		if name == "" {
			continue
		}
		ns := &Info{}
		for _, prop := range mod.Map.Properties {
			if prop.Name == "imports" {
				if l, ok := prop.Value.(*parser.List); ok {
					for _, v := range l.Values {
						if s, ok := v.(*parser.String); ok {
							ns.Imports = append(ns.Imports, s.Value)
						}
					}
				}
			}
		}
		result[name] = ns
	}
	return result
}

func ResolveModuleRef(ref string, namespaces map[string]*Info) string {
	ref = strings.TrimPrefix(ref, ":")
	if strings.HasPrefix(ref, "//") {
		sepIdx := strings.Index(ref, ":")
		if sepIdx >= 0 {
			nsName := ref[2:sepIdx]
			modName := ref[sepIdx+1:]
			if _, ok := namespaces[nsName]; ok {
				return modName
			}
		}
	}
	return ref
}

func ApplyOverrides(modules map[string]*parser.Module) {
	for name, ovr := range modules {
		if !ovr.Override {
			continue
		}
		base, ok := modules[name]
		if !ok || base == ovr {
			continue
		}
		if base.Map != nil && ovr.Map != nil {
			variant.MergeMapProps(base, ovr.Map)
		}
		modules[name] = base
	}
}

func ApplySoongConfigModuleTypes(modules map[string]*parser.Module, getStringProp func(*parser.Module, string) string, eval *parser.Evaluator) {
	for _, ct := range modules {
		if ct.Type != "soong_config_module_type" {
			continue
		}
		baseType := getStringProp(ct, "module_type")
		ns := getStringProp(ct, "config_namespace")
		typeName := getStringProp(ct, "name")
		if baseType == "" || typeName == "" {
			continue
		}
		if ct.Map != nil {
			for _, prop := range ct.Map.Properties {
				if prop.Name == "vars" {
					if mp, ok := prop.Value.(*parser.Map); ok {
						for _, p := range mp.Properties {
							if s, ok := p.Value.(*parser.String); ok {
								key := fmt.Sprintf("%s.%s", ns, p.Name)
								eval.SetConfig(key, s.Value)
							}
						}
					}
				}
			}
		}
		if !module.Has(typeName) {
			module.RegisterAlias(typeName, baseType)
		}
	}
}
