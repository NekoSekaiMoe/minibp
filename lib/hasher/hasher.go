// Package hasher provides dependency hash calculation for incremental builds.
// It calculates SHA256 hashes of modules and their dependency trees to determine
// if a rebuild is necessary.
package hasher

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"minibp/lib/parser"
)

// Hasher calculates and stores hashes for modules
type Hasher struct {
	cache        map[string]string // module hash cache
	hashDir      string            // hash storage directory
	moduleHashes map[string]string // module name to hash mapping
}

// NewHasher creates a new Hasher instance
func NewHasher(buildDir string) *Hasher {
	hashDir := filepath.Join(buildDir, ".minibp", "hash")
	return &Hasher{
		cache:        make(map[string]string),
		hashDir:      hashDir,
		moduleHashes: make(map[string]string),
	}
}

// CalculateModuleHash calculates the hash of a module including all its dependencies
func (h *Hasher) CalculateModuleHash(
	module *parser.Module,
	allModules map[string]*parser.Module,
) (string, error) {
	// Check cache
	moduleName := h.getModuleName(module)
	if hash, ok := h.cache[moduleName]; ok {
		return hash, nil
	}

	hasher := sha256.New()

	// 1. Add module's own properties hash
	if err := h.hashModuleProps(module, hasher); err != nil {
		return "", err
	}

	// 2. Add dependency module hashes (recursive)
	deps := h.getModuleDeps(module, allModules)
	for _, depName := range deps {
		if dep, ok := allModules[depName]; ok {
			depHash, err := h.CalculateModuleHash(dep, allModules)
			if err != nil {
				return "", fmt.Errorf("failed to calculate hash for dependency %s: %w", depName, err)
			}
			hasher.Write([]byte(depHash))
		}
	}

	// 3. Add source file content hash
	if err := h.hashSourceFiles(module, hasher); err != nil {
		return "", err
	}

	finalHash := hex.EncodeToString(hasher.Sum(nil))
	h.cache[moduleName] = finalHash
	return finalHash, nil
}

// hashModuleProps hashes the module's properties
func (h *Hasher) hashModuleProps(module *parser.Module, w io.Writer) error {
	// Write module type
	if module.Type != "" {
		fmt.Fprintf(w, "type:%s;", module.Type)
	}

	// Write module name
	if name := h.getModuleName(module); name != "" {
		fmt.Fprintf(w, "name:%s;", name)
	}

	// Write all properties
	props := h.extractProperties(module)
	sort.Strings(props) // Ensure consistent ordering
	for _, prop := range props {
		fmt.Fprintf(w, "prop:%s;", prop)
	}

	return nil
}

// extractProperties extracts all properties from a module
func (h *Hasher) extractProperties(module *parser.Module) []string {
	var props []string

	// Add type
	if module.Type != "" {
		props = append(props, "type:"+module.Type)
	}

	// Add name
	if name := h.getModuleName(module); name != "" {
		props = append(props, "name:"+name)
	}

	// Add deps
	if deps := h.getListProp(module, "deps"); len(deps) > 0 {
		sort.Strings(deps)
		props = append(props, "deps:"+strings.Join(deps, ","))
	}

	// Add srcs
	if srcs := h.getListProp(module, "srcs"); len(srcs) > 0 {
		sort.Strings(srcs)
		props = append(props, "srcs:"+strings.Join(srcs, ","))
	}

	// Add cflags
	if cflags := h.getListProp(module, "cflags"); len(cflags) > 0 {
		sort.Strings(cflags)
		props = append(props, "cflags:"+strings.Join(cflags, ","))
	}

	return props
}

// hashSourceFiles hashes the content of source files
func (h *Hasher) hashSourceFiles(module *parser.Module, w io.Writer) error {
	srcs := h.getSourceFiles(module)

	// Sort for consistency
	sort.Strings(srcs)

	for _, src := range srcs {
		if err := h.hashFile(src, w); err != nil {
			// Ignore file not found errors
			if !os.IsNotExist(err) {
				return err
			}
		}
	}

	return nil
}

// hashFile calculates the hash of a single file
func (h *Hasher) hashFile(path string, w io.Writer) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write file path (for detecting file changes)
	fmt.Fprintf(w, "file:%s;", path)

	// Calculate file content hash
	fileHasher := sha256.New()
	if _, err := io.Copy(fileHasher, f); err != nil {
		return err
	}

	fmt.Fprintf(w, "hash:%s;", hex.EncodeToString(fileHasher.Sum(nil)))
	return nil
}

// getModuleName extracts the name from a module
func (h *Hasher) getModuleName(module *parser.Module) string {
	if module.Map != nil {
		for _, prop := range module.Map.Properties {
			if prop.Name == "name" {
				if str, ok := prop.Value.(*parser.String); ok {
					return str.Value
				}
			}
		}
	}
	return ""
}

// getModuleDeps returns all dependencies of a module
func (h *Hasher) getModuleDeps(
	module *parser.Module,
	allModules map[string]*parser.Module,
) []string {
	deps := make(map[string]bool)

	// Get direct dependencies
	for _, dep := range h.getListProp(module, "deps") {
		deps[dep] = true
	}

	// Get shared library dependencies
	for _, dep := range h.getListProp(module, "shared_libs") {
		deps[dep] = true
	}

	// Get header library dependencies
	for _, dep := range h.getListProp(module, "header_libs") {
		deps[dep] = true
	}

	// Convert to slice
	result := make([]string, 0, len(deps))
	for dep := range deps {
		result = append(result, dep)
	}
	sort.Strings(result)

	return result
}

// getSourceFiles returns all source files for a module
func (h *Hasher) getSourceFiles(module *parser.Module) []string {
	srcs := h.getListProp(module, "srcs")

	// Expand glob patterns
	var expanded []string
	for _, src := range srcs {
		// If not a glob pattern, add directly
		if !strings.Contains(src, "*") {
			expanded = append(expanded, src)
			continue
		}

		// Handle glob patterns
		matches, err := filepath.Glob(src)
		if err == nil {
			expanded = append(expanded, matches...)
		}
	}

	return expanded
}

// getListProp gets a list property from a module
func (h *Hasher) getListProp(module *parser.Module, key string) []string {
	if module.Map == nil {
		return nil
	}

	// Find property
	for _, prop := range module.Map.Properties {
		if prop.Name == key {
			// Try to convert to list
			if list, ok := prop.Value.(*parser.List); ok {
				var result []string
				for _, item := range list.Values {
					if str, ok := item.(*parser.String); ok {
						result = append(result, str.Value)
					}
				}
				return result
			}
			// Try to get string
			if str, ok := prop.Value.(*parser.String); ok {
				return []string{str.Value}
			}
		}
	}

	return nil
}

// NeedsRebuild checks if a module needs to be rebuilt
func (h *Hasher) NeedsRebuild(moduleName string) (bool, error) {
	currentHash, ok := h.moduleHashes[moduleName]
	if !ok {
		// No hash record, needs build
		return true, nil
	}

	storedHash, err := h.LoadHash(moduleName)
	if err != nil {
		// Read failed, needs build
		return true, nil
	}

	return currentHash != storedHash, nil
}

// StoreHash stores the hash for a module
func (h *Hasher) StoreHash(moduleName, hash string) error {
	// Ensure directory exists
	if err := os.MkdirAll(h.hashDir, 0755); err != nil {
		return err
	}

	// Write hash file
	hashFile := filepath.Join(h.hashDir, moduleName+".hash")
	return os.WriteFile(hashFile, []byte(hash), 0644)
}

// LoadHash loads the stored hash for a module
func (h *Hasher) LoadHash(moduleName string) (string, error) {
	hashFile := filepath.Join(h.hashDir, moduleName+".hash")

	data, err := os.ReadFile(hashFile)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

// ClearCache clears the hash cache
func (h *Hasher) ClearCache() {
	h.cache = make(map[string]string)
}

// StoreAllHashes stores all module hashes
func (h *Hasher) StoreAllHashes() error {
	for name, hash := range h.moduleHashes {
		if err := h.StoreHash(name, hash); err != nil {
			return err
		}
	}
	return nil
}

// LoadAllHashes loads all stored hashes
func (h *Hasher) LoadAllHashes(moduleNames []string) error {
	for _, name := range moduleNames {
		hash, err := h.LoadHash(name)
		if err == nil {
			h.moduleHashes[name] = hash
		}
		// Ignore non-existent files
	}
	return nil
}