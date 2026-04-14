// module/module.go - Base module interface and struct
package module

// Module is the interface that all module types must implement.
// It provides a unified API for accessing module properties regardless
// of the specific module type (cc_library, go_binary, java_library, etc.).
// The build system uses this interface to work with modules generically.
type Module interface {
	// Name returns the unique name of this module within its package
	Name() string
	// Type returns the module type string (e.g., "cc_library", "go_binary")
	Type() string
	// Srcs returns the list of source files for this module
	Srcs() []string
	// Deps returns the list of dependency module names
	Deps() []string
	// Props returns all properties as a map
	Props() map[string]interface{}
	// GetProp retrieves a specific property by key
	GetProp(key string) interface{}
}

// BaseModule provides a common implementation of the Module interface.
// It embeds the core fields that all module types share.
// Module types embed BaseModule and add their specific fields.
type BaseModule struct {
	Name_  string                 // Module name (unique within package)
	Type_  string                 // Module type string
	Srcs_  []string               // List of source files
	Deps_  []string               // List of dependency module names
	Props_ map[string]interface{} // Additional properties as key-value pairs
}

// Name returns the module name
func (m *BaseModule) Name() string { return m.Name_ }

// Type returns the module type
func (m *BaseModule) Type() string { return m.Type_ }

// Srcs returns the list of source files
func (m *BaseModule) Srcs() []string { return m.Srcs_ }

// Deps returns the list of dependencies
func (m *BaseModule) Deps() []string { return m.Deps_ }

// Props returns all properties as a map
func (m *BaseModule) Props() map[string]interface{} { return m.Props_ }

// GetProp retrieves a specific property by key, returns nil if not found
func (m *BaseModule) GetProp(key string) interface{} { return m.Props_[key] }
