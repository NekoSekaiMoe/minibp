package toolchain

import (
	"strings"
	"testing"
)

func TestParseArchitecture(t *testing.T) {
	tests := []struct {
		input    string
		expected Architecture
		hasError bool
	}{
		{"arm", Arm, false},
		{"arm64", Arm64, false},
		{"aarch64", Arm64, false},
		{"x86", X86, false},
		{"i386", X86, false},
		{"i686", X86, false},
		{"x86_64", X86_64, false},
		{"amd64", X86_64, false},
		{"invalid", "", true},
	}

	for _, test := range tests {
		result, err := ParseArchitecture(test.input)
		if test.hasError {
			if err == nil {
				t.Errorf("Expected error for %s, got nil", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for %s: %v", test.input, err)
			}
			if result != test.expected {
				t.Errorf("Expected %s, got %s", test.expected, result)
			}
		}
	}
}

func TestParseOS(t *testing.T) {
	tests := []struct {
		input    string
		expected OS
		hasError bool
	}{
		{"linux", Linux, false},
		{"windows", Windows, false},
		{"darwin", Darwin, false},
		{"macos", Darwin, false},
		{"android", Android, false},
		{"invalid", "", true},
	}

	for _, test := range tests {
		result, err := ParseOS(test.input)
		if test.hasError {
			if err == nil {
				t.Errorf("Expected error for %s, got nil", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for %s: %v", test.input, err)
			}
			if result != test.expected {
				t.Errorf("Expected %s, got %s", test.expected, result)
			}
		}
	}
}

func TestDetectToolchain(t *testing.T) {
	tc := NewToolchainConfig()

	toolchain, err := tc.DetectToolchain(X86_64, Linux)
	if err != nil {
		t.Fatalf("Failed to detect toolchain: %v", err)
	}

	if toolchain.Arch != X86_64 {
		t.Errorf("Expected arch x86_64, got %s", toolchain.Arch)
	}
	if toolchain.OS != Linux {
		t.Errorf("Expected OS linux, got %s", toolchain.OS)
	}
}

func TestToolchainValidation(t *testing.T) {
	tc := &Toolchain{}

	// Empty toolchain should fail validation
	if err := tc.Validate(); err == nil {
		t.Error("Expected validation error for empty toolchain")
	}

	// Valid toolchain
	tc.Arch = X86_64
	tc.OS = Linux
	if err := tc.Validate(); err != nil {
		t.Errorf("Unexpected validation error: %v", err)
	}
}

func TestGetCompileFlags(t *testing.T) {
	tc := &Toolchain{
		Arch: X86_64,
		OS:   Linux,
	}

	flags := tc.GetCompileFlags()

	// x86_64 should have -m64 flag
	found := false
	for _, flag := range flags {
		if strings.Contains(flag, "m64") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected -m64 flag for x86_64")
	}
}

func TestGetCompileFlagsArm(t *testing.T) {
	tc := &Toolchain{
		Arch: Arm64,
		OS:   Linux,
	}

	flags := tc.GetCompileFlags()

	// arm64 should have armv8-a flag
	found := false
	for _, flag := range flags {
		if strings.Contains(flag, "armv8-a") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected armv8-a flag for arm64")
	}
}

func TestGetLinkFlags(t *testing.T) {
	tc := &Toolchain{
		Arch: X86_64,
		OS:   Linux,
	}

	flags := tc.GetLinkFlags()

	// x86_64 should have -m64 flag
	found := false
	for _, flag := range flags {
		if strings.Contains(flag, "m64") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected -m64 flag for x86_64 link flags")
	}
}

func TestGetOutputPrefix(t *testing.T) {
	tc := &Toolchain{
		Arch: X86_64,
		OS:   Linux,
	}

	prefix := tc.GetOutputPrefix()
	expected := "x86_64-linux"

	if prefix != expected {
		t.Errorf("Expected prefix %s, got %s", expected, prefix)
	}
}

func TestToolchainString(t *testing.T) {
	tc := &Toolchain{
		Arch: X86_64,
		OS:   Linux,
		CC:   "gcc",
		CXX:  "g++",
	}

	str := tc.String()

	if !strings.Contains(str, "x86_64") {
		t.Error("Expected string to contain architecture")
	}
	if !strings.Contains(str, "linux") {
		t.Error("Expected string to contain OS")
	}
}

func TestToolchainCaching(t *testing.T) {
	tc := NewToolchainConfig()

	// First detection
	toolchain1, err := tc.DetectToolchain(X86_64, Linux)
	if err != nil {
		t.Fatalf("Failed to detect toolchain: %v", err)
	}

	// Second detection should return cached result
	toolchain2, err := tc.DetectToolchain(X86_64, Linux)
	if err != nil {
		t.Fatalf("Failed to detect cached toolchain: %v", err)
	}

	if toolchain1 != toolchain2 {
		t.Error("Expected cached toolchain to be same instance")
	}
}

func TestSysroot(t *testing.T) {
	sysroot := "/path/to/sysroot"
	tc := &Toolchain{
		Arch:    Arm64,
		OS:      Linux,
		Sysroot: sysroot,
	}

	flags := tc.GetCompileFlags()

	// Should have sysroot flag
	found := false
	for _, flag := range flags {
		if strings.Contains(flag, sysroot) {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected sysroot flag")
	}
}
