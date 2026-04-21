// module/registry.go - Factory registry for module creation
package module

import (
	"fmt"
	"sync"

	"minibp/lib/parser"
)

// Factory is the interface for creating Module instances from AST nodes.
// Each module type (cc_library, go_binary, etc.) has a corresponding factory
// that knows how to parse the AST and create the appropriate Module struct.
type Factory interface {
	Create(ast *parser.Module, eval *parser.Evaluator) (Module, error)
}

// registryMu is a mutex for thread-safe access to the registry map.
// Using RWMutex allows multiple concurrent readers while blocking writers.
var (
	registryMu sync.RWMutex
	registry   = make(map[string]Factory)
)

// Register adds a factory to the registry for a specific module type name.
// This is called during initialization to register all built-in module types.
// Parameters:
//   - name: The module type string (e.g., "cc_library", "go_binary")
//   - factory: The Factory implementation for creating that module type
func Register(name string, factory Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()

	registry[name] = factory
}

// Lookup retrieves a factory from the registry by module type name.
// Returns nil if no factory is registered for the given type.
// Parameters:
//   - name: The module type string to look up
//
// Returns:
//
//	The Factory for that module type, or nil if not found
func Lookup(name string) Factory {
	registryMu.RLock()
	defer registryMu.RUnlock()

	return registry[name]
}

// registrySnapshot creates a shallow copy of the current registry state.
// This is useful for testing to preserve and restore the registry.
// Returns:
//
//	A map containing copies of all registered factories
func registrySnapshot() map[string]Factory {
	registryMu.RLock()
	defer registryMu.RUnlock()

	snapshot := make(map[string]Factory, len(registry))
	for name, factory := range registry {
		snapshot[name] = factory
	}

	return snapshot
}

// restoreRegistry replaces the current registry with a snapshot.

// This is used in tests to restore the registry to a previous state.

// Parameters:

// - snapshot: A map of module type names to factories

func restoreRegistry(snapshot map[string]Factory) {

	registryMu.Lock()

	defer registryMu.Unlock()

	if snapshot == nil {

		registry = make(map[string]Factory)

		return

	}

	registry = make(map[string]Factory, len(snapshot))

	for name, factory := range snapshot {

		registry[name] = factory

	}

}

// resetRegistry clears all registered factories from the registry.
// This is used in tests to start with a clean registry state.
func resetRegistry() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = make(map[string]Factory)
}

// registryLen returns the number of registered module types.
// This is primarily used in tests to verify registration state.
// Returns:
//
//	The count of registered factories in the registry
func registryLen() int {
	registryMu.RLock()
	defer registryMu.RUnlock()

	return len(registry)
}

// Create builds a Module from an AST node using the appropriate factory.
// It looks up the factory by the module type and delegates creation to it.
// Parameters:
//   - ast: The parsed AST module node
//   - eval: Optional evaluator for expression evaluation
//
// Returns:
//
//	A Module instance and any error that occurred
//	Returns an error if the module type is not registered
func Create(ast *parser.Module, eval *parser.Evaluator) (Module, error) {
	factory := Lookup(ast.Type)
	if factory == nil {
		return nil, fmt.Errorf("unknown module type: %s", ast.Type)
	}
	return factory.Create(ast, eval)
}
