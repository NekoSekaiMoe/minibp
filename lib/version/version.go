// Package version provides version information for minibp.
// This package encapsulates all build-time and runtime version metadata,
// allowing the binary to self-report its version without external configuration files.
//
// Version information is injected at build time using Go's -ldflags mechanism
// via the -X flag, which sets package-level variables. This approach ensures
// version information is embedded directly in the binary, making it portable
// and not requiring runtime file access.
//
// The Get() function returns version info combining both build-time injected values
// (git metadata, build date) and runtime-detected values (Go version, compiler, platform).
//
// Typical build command with version injection:
//   go build -ldflags=" \
//     -X 'minibp/lib/version.gitTag=$(git describe --tags --abbrev=0)' \
//     -X 'minibp/lib/version.gitBranch=$(git branch --show-current)' \
//     -X 'minibp/lib/version.gitCommit=$(git rev-parse HEAD)' \
//     -X 'minibp/lib/version.gitTreeState=$(test -z "$(git status --porcelain)" && echo clean || echo dirty)' \
//     -X 'minibp/lib/version.buildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)' \
//     -X 'minibp/lib/version.minibpVer=0.001'" \
//     -o minibp cmd/minibp/main.go
package version

import (
	"fmt"
	"runtime"
)

// Info contains comprehensive version information about the minibp build.
// All fields are exported to support JSON serialization for programmatic access,
// such as in CI/CD pipelines or version reporting tools.
//
// This struct is designed to be backwards compatible: missing build-time
// injection values default to "unknown" rather than empty strings.
//
// JSON serialization example:
//   data, _ := json.Marshal(version.Get())
//   fmt.Println(string(data))
//
// Fields are grouped by source:
//   - Build-time: GitTag, GitBranch, GitCommit, GitTreeState, BuildDate, MinibpVer
//   - Runtime: GoVersion, Compiler, Platform
type Info struct {
	// GitTag is the Git tag from the most recent commit, e.g., "v1.2.3".
	// This is typically set by git describe --tags at build time.
	// Empty if no tag exists or not injected.
	// Returns: Tag string or "unknown" if not set
	GitTag string `json:"gitTag"`

	// GitBranch is the current Git branch name, e.g., "main", "feature/my-branch".
	// This is typically set by git branch --show-current at build time.
	// Returns: Branch name or "unknown" if not set
	GitBranch string `json:"gitBranch"`

	// GitCommit is the full Git commit hash (40 characters), e.g., "abc123def456...".
	// This is set by git rev-parse HEAD at build time.
	// Returns: Full 40-character commit SHA or "unknown" if not set
	GitCommit string `json:"gitCommit"`

	// GitTreeState describes the state of the Git working tree at build time.
	// Values: "clean" (no uncommitted changes) or "dirty" (uncommitted changes exist).
	// This is determined by checking git status --porcelain at build time.
	// Returns: "clean" or "dirty" or "unknown" if not set
	GitTreeState string `json:"gitTreeState"`

	// BuildDate is the date and time of the build in ISO 8601 format, e.g., "2024-01-15T10:30:00Z".
	// Uses UTC timezone to ensure consistent cross-timezone builds.
	// Format: YYYY-MM-DDTHH:MM:SSZ
	// Returns: ISO 8601 timestamp or "unknown" if not set
	BuildDate string `json:"buildDate"`

	// MinibpVer is the semantic version of minibp itself, e.g., "0.001".
	// This differs from GitTag as it tracks the project's own versioning scheme.
	// Defaults to "0.001" before the first official release.
	// Returns: Semantic version string, defaults to "0.001"
	MinibpVer string `json:"minibpVersion"`

	// GoVersion is the Go runtime version, e.g., "go1.21.0".
	// Detected at runtime via runtime.Version().
	// Returns: Go version string from runtime
	GoVersion string `json:"goVersion"`

	// Compiler is the Go compiler used, either "gc" (standard compiler) or "gccgo".
	// Detected at runtime via runtime.Compiler.
	// Returns: "gc" or "gccgo"
	Compiler string `json:"compiler"`

	// Platform is the target platform in OS/arch format, e.g., "linux/amd64", "darwin/arm64".
	// Detected at runtime via runtime.GOOS and runtime.GOARCH.
	// Returns: OS/arch tuple, e.g., "linux/amd64", "darwin/arm64"
	Platform string `json:"platform"`
}

// String returns the Git tag as the string representation of the version info.
// This implements the fmt.Stringer interface for convenient printing,
// allowing direct use in fmt.Printf with %s or fmt.Sprintf.
//
// Returns:
//   - The GitTag value if non-empty and not "unknown"
//   - "unknown" if the tag was not set at build time
//
// Note: This method returns only the GitTag, not the full version info.
// For complete version output, use Get() and access desired fields directly.
//
// Edge cases:
//   - Empty GitTag returns empty string (not "unknown") since String() accesses the field directly
//   - "unknown" GitTag returns "unknown" string
//
// Example output: "v1.2.3" or "unknown" or ""
func (info Info) String() string {
	return info.GitTag
}

// Get returns the complete version Info for this binary.
//
// This function combines build-time injected values (from -ldflags) with
// runtime-detected values. Build-time injection ensures version consistency
// across deployments, while runtime detection provides accurate environment info.
//
// Build-time injected values (may be "unknown" if not set):
//   - gitTag: Git tag from most recent commit, e.g., "v1.2.3"
//   - gitBranch: Current branch name, e.g., "main"
//   - gitCommit: Full commit hash, e.g., "abc123..."
//   - gitTreeState: "clean" or "dirty" indicating uncommitted changes
//   - buildDate: ISO 8601 timestamp of build, e.g., "2024-01-15T10:30:00Z"
//   - minibpVer: Project version, e.g., "0.001"
//
// Runtime-detected values (always accurate for the current execution):
//   - GoVersion: Go runtime version, e.g., "go1.21.0"
//   - Compiler: Go compiler used, "gc" or "gccgo"
//   - Platform: OS/arch tuple, e.g., "linux/amd64"
//
// Returns:
//   - Info struct populated with all version fields
//
// Note: The returned struct shares no pointers with internal state,
// making it safe to modify without affecting future Get() calls.
// The caller may freely modify the returned struct.
//
// Edge cases:
//   - If build-time injection failed, fields will have default "unknown" values
//   - Runtime values are always populated from the running process
//   - Platform format uses runtime.GOOS/runtime.GOARCH, not build target
func Get() Info {
	return Info{
		GitTag:       gitTag,
		GitBranch:    gitBranch,
		GitCommit:    gitCommit,
		GitTreeState: gitTreeState,
		BuildDate:    buildDate,
		MinibpVer:    minibpVer,
		GoVersion:    runtime.Version(),
		Compiler:     runtime.Compiler,
		Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// Package-level variables for build-time version injection.
// These are marked as private to prevent direct access from other packages,
// enforcing the use of the Get() function which provides the complete Info struct.
//
// Build tools inject values using the -X flag with the full import path.
//
// Default values ensure the binary remains functional even without injection,
// though version reporting will show "unknown" for missing fields.
//
// Example ldflags injection:
//   -X 'minibp/lib/version.gitTag=v1.0.0'
//   -X 'minibp/lib/version.gitCommit=$(git rev-parse HEAD)'
//   -X 'minibp/lib/version.gitTreeState=$(test -z "$(git status --porcelain)" && echo clean || echo dirty)'
//   -X 'minibp/lib/version.buildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)'
//   -X 'minibp/lib/version.minibpVer=0.001'
var (
	// gitTag is the Git tag from the most recent commit.
	// Set by: git describe --tags --abbrev=0
	// Injected via: -X 'minibp/lib/version.gitTag=$(git describe --tags --abbrev=0)'
	gitTag = "unknown"

	// gitBranch is the current Git branch name.
	// Examples: "main", "feature/my-branch", "release/v2.0"
	// Set by: git branch --show-current
	// Injected via: -X 'minibp/lib/version.gitBranch=$(git branch --show-current)'
	gitBranch = "unknown"

	// gitCommit is the full Git commit hash (40 hex characters).
	// Example: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
	// Set by: git rev-parse HEAD
	// Injected via: -X 'minibp/lib/version.gitCommit=$(git rev-parse HEAD)'
	gitCommit = "unknown"

	// gitTreeState indicates whether the working tree had uncommitted changes.
	// Values: "clean" (no uncommitted changes) or "dirty" (uncommitted changes exist)
	// Set by: git status --porcelain | wc -l | xargs test 0 -eq 1 && echo dirty || echo clean
	// Injected via: -X 'minibp/lib/version.gitTreeState=$(test -z "$(git status --porcelain)" && echo clean || echo dirty)'
	gitTreeState = "unknown"

	// buildDate is the build timestamp in ISO 8601 format (UTC).
	// Example: "2024-01-15T10:30:00Z"
	// Format: YYYY-MM-DDTHH:MM:SSZ
	// Set by: date -u +%Y-%m-%dT%H:%M:%SZ
	// Injected via: -X 'minibp/lib/version.buildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)'
	buildDate = "unknown"

	// minibpVer is the project's semantic version string.
	// Example: "0.001", "1.0.0", "2.3.4-beta"
	// Starts at "0.001" before official release tagging begins.
	// Injected via: -X 'minibp/lib/version.minibpVer=0.001'
	minibpVer = "0.001"
)
