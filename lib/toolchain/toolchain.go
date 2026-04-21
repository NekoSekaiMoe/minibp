// Package toolchain provides cross-architecture compilation support.
// It manages toolchain configurations for different target architectures
// and operating systems.
package toolchain

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Architecture represents a target CPU architecture
type Architecture string

const (
	Arm    Architecture = "arm"
	Arm64  Architecture = "arm64"
	X86    Architecture = "x86"
	X86_64 Architecture = "x86_64"
)

// OS represents a target operating system
type OS string

const (
	Linux   OS = "linux"
	Windows OS = "windows"
	Darwin  OS = "darwin"
	Android OS = "android"
)

// Toolchain represents a complete toolchain configuration
type Toolchain struct {
	Arch    Architecture
	OS      OS
	CC      string // C compiler
	CXX     string // C++ compiler
	AR      string // Static linker
	LD      string // Linker
	Sysroot string
}

// ToolchainConfig provides toolchain detection and configuration
type ToolchainConfig struct {
	defaultCC   string
	defaultCXX  string
	defaultAR   string
	defaultLD   string
	toolchains  map[string]*Toolchain
}

// NewToolchainConfig creates a new toolchain configuration manager
func NewToolchainConfig() *ToolchainConfig {
	return &ToolchainConfig{
		defaultCC:  "gcc",
		defaultCXX: "g++",
		defaultAR:  "ar",
		defaultLD:  "ld",
		toolchains: make(map[string]*Toolchain),
	}
}

// DetectToolchain detects the appropriate toolchain for the given
// architecture and OS
func (tc *ToolchainConfig) DetectToolchain(arch Architecture, targetOS OS) (*Toolchain, error) {
	key := fmt.Sprintf("%s-%s", arch, targetOS)

	if toolchain, ok := tc.toolchains[key]; ok {
		return toolchain, nil
	}

	toolchain := &Toolchain{
		Arch: arch,
		OS:   targetOS,
		CC:   tc.defaultCC,
		CXX:  tc.defaultCXX,
		AR:   tc.defaultAR,
		LD:   tc.defaultLD,
	}

	// Detect toolchain based on OS and architecture
	toolchain.CC, toolchain.CXX, toolchain.AR = tc.detectTools(arch, targetOS)

	tc.toolchains[key] = toolchain
	return toolchain, nil
}

// detectTools detects the appropriate compiler tools for the given
// architecture and OS
func (tc *ToolchainConfig) detectTools(arch Architecture, targetOS OS) (cc, cxx, ar string) {
	// Default values
	cc = tc.defaultCC
	cxx = tc.defaultCXX
	ar = tc.defaultAR

	// Check for architecture-specific toolchain
	prefix := tc.getToolchainPrefix(arch, targetOS)
	if prefix != "" {
		cc = prefix + "-gcc"
		cxx = prefix + "-g++"
		ar = prefix + "-ar"
	}

	// Check if tools exist
	if !tc.toolExists(cc) {
		cc = tc.defaultCC
	}
	if !tc.toolExists(cxx) {
		cxx = tc.defaultCXX
	}
	if !tc.toolExists(ar) {
		ar = tc.defaultAR
	}

	return cc, cxx, ar
}

// getToolchainPrefix returns the toolchain prefix for the given
// architecture and OS
func (tc *ToolchainConfig) getToolchainPrefix(arch Architecture, targetOS OS) string {
	switch targetOS {
	case Android:
		switch arch {
		case Arm:
			return "arm-linux-androideabi"
		case Arm64:
			return "aarch64-linux-android"
		case X86:
			return "i686-linux-android"
		case X86_64:
			return "x86_64-linux-android"
		}
	case Linux:
		switch arch {
		case Arm:
			return "arm-linux-gnueabihf"
		case Arm64:
			return "aarch64-linux-gnu"
		case X86:
			return "i686-linux-gnu"
		case X86_64:
			return "x86_64-linux-gnu"
		}
	}
	return ""
}

// toolExists checks if a tool (executable) exists in PATH
func (tc *ToolchainConfig) toolExists(name string) bool {
	_, err := tc.findExecutable(name)
	return err == nil
}

// findExecutable searches for an executable in PATH
func (tc *ToolchainConfig) findExecutable(name string) (string, error) {
	return execLookup(name)
}

// execLookup wraps os/exec lookup to avoid import
func execLookup(name string) (string, error) {
	// Simple implementation: check if in PATH
	path := os.Getenv("PATH")
	for _, dir := range filepath.SplitList(path) {
		execPath := filepath.Join(dir, name)
		if info, err := os.Stat(execPath); err == nil && !info.IsDir() {
			return execPath, nil
		}
	}
	return "", fmt.Errorf("executable not found: %s", name)
}

// GetCompileFlags returns architecture-specific compile flags
func (t *Toolchain) GetCompileFlags() []string {
	flags := []string{}

	switch t.Arch {
	case Arm:
		flags = append(flags, "-march=armv7-a", "-mthumb")
	case Arm64:
		flags = append(flags, "-march=armv8-a")
	case X86:
		flags = append(flags, "-m32")
	case X86_64:
		flags = append(flags, "-m64")
	}

	if t.Sysroot != "" {
		flags = append(flags, "--sysroot="+t.Sysroot)
	}

	return flags
}

// GetLinkFlags returns architecture-specific link flags
func (t *Toolchain) GetLinkFlags() []string {
	flags := []string{}

	switch t.Arch {
	case Arm:
		flags = append(flags, "-march=armv7-a")
	case Arm64:
		flags = append(flags, "-march=armv8-a")
	case X86:
		flags = append(flags, "-m32")
	case X86_64:
		flags = append(flags, "-m64")
	}

	if t.Sysroot != "" {
		flags = append(flags, "--sysroot="+t.Sysroot)
	}

	return flags
}

// GetOutputPrefix returns the prefix for output files
func (t *Toolchain) GetOutputPrefix() string {
	return fmt.Sprintf("%s-%s", t.Arch, t.OS)
}

// Validate checks if the toolchain configuration is valid
func (t *Toolchain) Validate() error {
	if t.Arch == "" {
		return fmt.Errorf("architecture not specified")
	}
	if t.OS == "" {
		return fmt.Errorf("operating system not specified")
	}
	return nil
}

// String returns a string representation of the toolchain
func (t *Toolchain) String() string {
	return fmt.Sprintf("%s-%s (%s/%s)",
		t.Arch, t.OS, t.CC, t.CXX)
}

// ParseArchitecture parses an architecture string
func ParseArchitecture(s string) (Architecture, error) {
	s = strings.ToLower(s)
	switch s {
	case "arm":
		return Arm, nil
	case "arm64", "aarch64":
		return Arm64, nil
	case "x86", "i386", "i686":
		return X86, nil
	case "x86_64", "amd64":
		return X86_64, nil
	default:
		return "", fmt.Errorf("unknown architecture: %s", s)
	}
}

// ParseOS parses an OS string
func ParseOS(s string) (OS, error) {
	s = strings.ToLower(s)
	switch s {
	case "linux":
		return Linux, nil
	case "windows":
		return Windows, nil
	case "darwin", "macos":
		return Darwin, nil
	case "android":
		return Android, nil
	default:
		return "", fmt.Errorf("unknown operating system: %s", s)
	}
}
