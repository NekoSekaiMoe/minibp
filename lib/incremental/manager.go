// Package incremental implements incremental build functionality for caching and reusing
// parsed Blueprint file results.
//
// This package provides the Manager type to manage incremental parsing state:
//   - Tracks which .bp files have changed (by comparing file hashes)
//   - Caches parsed ASTs as JSON files (stored in .minibp/json/ directory)
//   - Maintains dependency file .minibp/dep.json to record file hashes
//
// Main workflow:
//  1. Create Manager and load existing dependency hashes (if any)
//  2. Check each .bp file for reparsing needs (NeedsReparse)
//  3. If file unchanged, load from JSON cache (LoadJSON)
//  4. If file changed or first seen, reparse and save cache (SaveJSON)
//  5. Finally save updated dependency hashes (SaveDepFile)
//
// Example:
//  mgr, err := incremental.NewManager("/path/to/project")
//  if err != nil { return err }
//  needsReparse, _ := mgr.NeedsReparse("foo.bp")
//  if needsReparse {
//      // Parse file and cache
//      mgr.SaveJSON("foo.bp", parsedFile)
//  } else {
//      // Load from cache
//      cached, _ := mgr.LoadJSON("foo.bp")
//  }
//  mgr.SaveDepFile()
package incremental

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"minibp/lib/parser"
)

// DepFile represents the data structure of the .minibp/dep.json dependency file.
//
// This file records the SHA256 hash of each .bp file to determine if the file has changed
// during the next build, thus deciding whether reparsing is needed.
//
// Example JSON format:
//
//	{
//	  "version": 1,
//	  "hashes": {
//	    "src/foo/Android.bp": "a1b2c3d4...",
//	    "src/bar/Android.bp": "e5f6g7h8..."
//	  }
//	}
//
// Fields:
//   - Version: Dependency file format version, currently 1
//   - Hashes: Map from .bp file path to its SHA256 hash value
type DepFile struct {
	Version int                `json:"version"`
	Hashes  map[string]string `json:"hashes"` // bpFilePath -> sha256hex
}

// Manager manages incremental parsing state.
//
// Manager is responsible for tracking which .bp files have changed and caching
// parsed ASTs as JSON files. It determines if a file needs reparsing by
// comparing the file's SHA256 hash.
//
// Main responsibilities:
//   - Manage cache files under .minibp/ directory
//   - Record and maintain hash values for each .bp file
//   - Provide methods to check if files need reparsing
//   - Provide methods to save and load JSON cache
//
// Workflow:
//   1. Create necessary directories on initialization (.minibp/ and .minibp/json/)
//   2. Attempt to load existing dep.json file to restore previous hash records
//   3. Check each file for reparsing needs during build process
//   4. Save updated dependency information after build completes
type Manager struct {
	workDir string            // project root (where .minibp lives)
	jsonDir string            // .minibp/json/
	depFile string            // .minibp/dep.json
	hashes  map[string]string // in-memory copy of dep.json hashes
}

// NewManager creates a new incremental build manager.
//
// This function initializes a Manager instance and sets up necessary cache directories.
// If .minibp/ or .minibp/json/ directories don't exist, they will be created automatically.
// It also attempts to load an existing dep.json file to restore previous file hash records.
//
// Parameters:
//   - workDir: Project root directory path where .minibp directory will be created
//
// Returns:
//   - *Manager: Initialized Manager instance
//   - error: Returns error if directory creation fails; loading dep.json failure won't return error (starts fresh)
//
// Edge cases:
//   - If .minibp/dep.json doesn't exist or has invalid format, starts fresh (clears hash records)
//   - Directory creation failure returns error immediately
//   - Even if dep.json fails to load, a usable Manager instance is returned
//
// Example:
//
//	mgr, err := incremental.NewManager("/home/user/myproject")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// mgr is now ready to use, will manage cache under /home/user/myproject/.minibp/
func NewManager(workDir string) (*Manager, error) {
	// Construct cache directory and dependency file paths
	jsonDir := filepath.Join(workDir, ".minibp", "json")
	depFile := filepath.Join(workDir, ".minibp", "dep.json")

	// Initialize Manager instance with work directory and cache paths
	m := &Manager{
		workDir: workDir,
		jsonDir: jsonDir,
		depFile: depFile,
		hashes:  make(map[string]string),
	}

	// Create .minibp/json/ directory if it doesn't exist
	// MkdirAll creates all directories in the path that don't exist
	if err := os.MkdirAll(jsonDir, 0755); err != nil {
		return nil, fmt.Errorf("create json dir: %w", err)
	}

	// Attempt to load existing dependency file
	// If file doesn't exist or is corrupted, don't return error; start fresh (clear hashes)
	if err := m.loadDepFile(); err != nil {
		// If dep.json doesn't exist or has invalid format, start fresh
		// This ensures build won't fail even if cache is corrupted
		m.hashes = make(map[string]string)
	}

	return m, nil
}

// loadDepFile loads the .minibp/dep.json dependency file.
//
// This function reads the dep.json file from disk and parses its JSON content,
// loading the file hash values recorded in the file into memory (m.hashes).
// Returns error if file doesn't exist or format is invalid.
//
// Returns:
//   - error: Returns error if file read fails or JSON parsing fails
//     - File not found: Returns os.PathError
//     - Invalid JSON format: Returns json.UnmarshalError
//     - Success: Returns nil
//
// Edge cases:
//   - If dep.json exists but Hashes field is null, initializes to empty map
//   - If dep.json exists but format version doesn't match, still attempts to load (only uses Hashes field)
//   - Caller should handle returned error, typically choosing to start fresh rather than fail
//
// Example dep.json content:
//
//	{
//	  "version": 1,
//	  "hashes": {
//	    "src/foo.bp": "a1b2c3d4e5f6..."
//	  }
//	}
func (m *Manager) loadDepFile() error {
	// Read raw content of dep.json file
	data, err := os.ReadFile(m.depFile)
	if err != nil {
		return err
	}

	// Parse JSON content into DepFile struct
	var dep DepFile
	if err := json.Unmarshal(data, &dep); err != nil {
		return err
	}

	// Copy loaded hash values into memory
	m.hashes = dep.Hashes
	// Handle case where Hashes is null (JSON "hashes": null)
	if m.hashes == nil {
		m.hashes = make(map[string]string)
	}
	return nil
}

// SaveDepFile saves current dependency hashes to .minibp/dep.json file.
//
// This function serializes the in-memory file hash records (m.hashes) to JSON format
// and writes to the dep.json file. Uses MarshalIndent to generate formatted JSON
// for human readability and version control.
//
// Returns:
//   - error: Returns error if JSON serialization fails or file write fails
//     - Serialization failure: Returns json.MarshalError
//     - Write failure: Returns os.PathError or permission error
//     - Success: Returns nil
//
// Edge cases:
//   - If m.hashes is empty map, still writes a valid JSON file (hashes as empty object)
//   - File permission set to 0644 (owner read/write, group and others read-only)
//   - Write failure doesn't rollback previous state
//
// Example dep.json output:
//
//	{
//	  "version": 1,
//	  "hashes": {
//	    "src/foo.bp": "a1b2c3d4...",
//	    "src/bar.bp": "e5f6g7h8..."
//	  }
//	}
func (m *Manager) SaveDepFile() error {
	// 构造 DepFile 结构体，准备序列化
	dep := DepFile{
		Version: 1,
		Hashes:  m.hashes,
	}
	// 序列化为格式化的 JSON（使用两个空格缩进）
	data, err := json.MarshalIndent(dep, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal dep file: %w", err)
	}
	// 写入 dep.json 文件，权限 0644
	return os.WriteFile(m.depFile, data, 0644)
}

// hashFile computes the SHA256 hash of the specified file's content.
//
// This function opens the file, reads its entire content, and computes the SHA256 hash.
// The hash value is returned as a hexadecimal string (64 characters long).
//
// Parameters:
//   - path: File path to compute hash for
//
// Returns:
//   - string: SHA256 hash of file content (hex lowercase string)
//   - error: Returns error if file open or read fails
//     - File not found: Returns os.PathError
//     - Read failure: Returns io.ReadError
//     - Success: Returns nil
//
// Edge cases:
//   - Empty file hash: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
//   - Large files are read in streaming fashion, not loaded entirely into memory
//   - File open failure returns error immediately
//
// Example:
//
//	hash, err := mgr.hashFile("src/foo.bp")
//	if err != nil {
//	    return err
//	}
//	fmt.Println(hash) // Output similar to: a1b2c3d4e5f6...
func (m *Manager) hashFile(path string) (string, error) {
	// 打开文件用于读取
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file for hash: %w", err)
	}
	// 确保文件在函数返回前关闭
	defer f.Close()

	// 初始化 SHA256 哈希计算器
	h := sha256.New()
	// 流式复制文件内容到哈希计算器
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash file content: %w", err)
	}
	// 计算最终的哈希值并格式化为十六进制字符串
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// NeedsReparse checks if the specified .bp file needs reparsing.
//
// This function computes the current SHA256 hash of the file and compares it
// with the stored hash value in memory. If the file is first seen or content
// has changed, reparsing is needed.
//
// Parameters:
//   - bpFile: Path to the .bp file to check
//
// Returns:
//   - bool: true means reparsing needed, false means file unchanged and cache can be used
//   - error: Returns error if file hash computation fails (returns true, error in this case)
//
// Cases requiring reparsing:
//   - First build: No stored hash value for this file
//   - File modified: Current hash doesn't match stored hash
//   - Hash computation failed: Returns error (caller decides how to handle)
//
// Edge cases:
//   - If file hash computation fails, returns true (conservative strategy: reparse rather than use potentially stale cache)
//   - First-seen file automatically updates hash value in memory
//   - File content change updates hash in memory (but not persisted, need to call SaveDepFile)
//   - Even if file is read-only or empty, hash is computed normally
//
// Example:
//
//	needsReparse, err := mgr.NeedsReparse("src/foo/Android.bp")
//	if err != nil {
//	    // Handle error (file may not exist)
//	    return err
//	}
//	if needsReparse {
//	    // Need to reparse file
//	    parsedFile, _ := parser.ParseFile(...)
//	    mgr.SaveJSON("src/foo/Android.bp", parsedFile)
//	} else {
//	    // Can use cache
//	    cached, _ := mgr.LoadJSON("src/foo/Android.bp")
//	}
func (m *Manager) NeedsReparse(bpFile string) (bool, error) {
	// 计算文件的当前哈希值
	currentHash, err := m.hashFile(bpFile)
	if err != nil {
		// 哈希计算失败，保守地认为需要重新解析
		return true, fmt.Errorf("hash %s: %w", bpFile, err)
	}

	// 检查是否有存储的哈希值
	storedHash, exists := m.hashes[bpFile]
	if !exists {
		// 首次看到这个文件，记录哈希值并返回需要解析
		m.hashes[bpFile] = currentHash
		return true, nil
	}

	// 比较哈希值是否相同
	if storedHash != currentHash {
		// 文件已改变，更新哈希值并返回需要解析
		m.hashes[bpFile] = currentHash
		return true, nil
	}

	// 文件未改变，可以使用缓存
	return false, nil
}

// jsonFilePath returns the JSON cache file path for the specified .bp file.
//
// This function converts .bp file path to cache file path using these rules:
//  1. First try to get relative path from work directory (ensures path stability)
//  2. Replace path separators (/ or \) with __ to avoid directory traversal issues
//  3. Use sanitizeName to further clean special characters in filename
//  4. Final file is placed under .minibp/json/ directory with .json extension
//
// Parameters:
//   - bpFile: Path to .bp file (can be absolute or relative)
//
// Returns:
//   - string: Full path to the JSON cache file
//
// Edge cases:
//   - If relative path cannot be computed, falls back to original path
//   - Special characters in path (like :*?"<>|) are replaced with _
//   - Path separators on different OS are handled correctly
//
// Example:
//
//	mgr.workDir = "/home/user/project"
//	mgr.jsonFilePath("/home/user/project/src/foo/Android.bp")
//	// Returns: /home/user/project/.minibp/json/src__foo__Android.bp.json
//
//	mgr.jsonFilePath("src/bar.bp")
//	// Returns: /home/user/project/.minibp/json/src__bar.bp.json
func (m *Manager) jsonFilePath(bpFile string) string {
	// 尝试获取相对于工作目录的路径，保证缓存键的稳定性
	// 这样即使工作目录改变，只要相对路径相同就能命中缓存
	rel, err := filepath.Rel(m.workDir, bpFile)
	if err != nil {
		// 如果无法计算相对路径（如在不同驱动器上），使用原始路径
		rel = bpFile
	}
	// 将路径分隔符替换为 __，避免目录遍历问题
	sanitized := strings.ReplaceAll(rel, string(filepath.Separator), "__")
	sanitized = strings.ReplaceAll(sanitized, "/", "__")
	// 拼接最终路径：.minibp/json/清理后的文件名.json
	return filepath.Join(m.jsonDir, sanitizeName(sanitized)+".json")
}

// sanitizeName ensures filename is safe and contains no characters that could cause
// path traversal or filesystem issues.
//
// This function iterates through each character in the input string and replaces
// the following special characters with underscore (_):
//   - Path separators: / and \
//   - Windows forbidden characters: : * ? " < > |
//
// This prevents security issues (like path traversal attacks) from malicious or
// accidental file paths, or problems on filesystems that don't support certain characters.
//
// Parameters:
//   - name: Filename or path string to sanitize
//
// Returns:
//   - string: Sanitized safe filename
//
// Edge cases:
//   - Empty string returns empty string
//   - If no special characters, returns original string
//   - All matching special characters are replaced, not just the first
//   - Unicode characters not in special character list are preserved
//
// Example:
//
//	sanitizeName("src/foo:bar.bp")
//	// Returns: src_foo_bar.bp
//
//	sanitizeName("C:\\Users\\test\\file.bp")
//	// Returns: C__Users_test__file.bp
//
//	sanitizeName("normal_file.bp")
//	// Returns: normal_file.bp (no change)
func sanitizeName(name string) string {
	// 使用 strings.Map 遍历每个字符并进行替换
	result := strings.Map(func(r rune) rune {
		// 检查是否为需要替换的特殊字符
		if r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|' {
			return '_'
		}
		return r
	}, name)
	return result
}

// SaveJSON saves the parsed File AST to a JSON cache file.
//
// This function serializes the parser.File struct to formatted JSON and writes
// to the cache directory. Cache file path is determined by jsonFilePath method,
// typically located under .minibp/json/ directory.
//
// Parameters:
//   - bpFile: Original .bp file path, used to generate cache filename
//   - file: Parsed AST (Abstract Syntax Tree) to cache
//
// Returns:
//   - error: Returns error if JSON serialization fails or file write fails
//     - Serialization failure: Returns json.MarshalError
//     - Write failure: Returns os.PathError or permission error
//     - Success: Returns nil
//
// Edge cases:
//   - If target directory doesn't exist, returns error (should be created in NewManager)
//   - File permission set to 0644 (owner read/write, group and others read-only)
//   - If cache file already exists, it will be overwritten
//   - Serialization uses indented format for debugging and version control
//
// Example:
//
//	parsedFile, _ := parser.ParseFile(reader, "src/foo.bp", source)
//	err := mgr.SaveJSON("src/foo.bp", parsedFile)
//	if err != nil {
//	    // Cache failure shouldn't block build, can log warning
//	    fmt.Fprintf(os.Stderr, "warning: failed to cache: %v\n", err)
//	}
func (m *Manager) SaveJSON(bpFile string, file *parser.File) error {
	// 获取缓存文件的路径
	jsonPath := m.jsonFilePath(bpFile)

	// 将 AST 序列化为格式化的 JSON（使用两个空格缩进）
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal ast to json: %w", err)
	}

	// 写入缓存文件
	return os.WriteFile(jsonPath, data, 0644)
}

// LoadJSON loads a previously parsed File AST from JSON cache file.
//
// This function attempts to read the JSON cache for the specified .bp file
// and deserializes it into a parser.File struct. If the cache file doesn't
// exist or is corrupted, returns nil (not an error, just cache miss).
//
// Parameters:
//   - bpFile: Original .bp file path, used to locate cache file
//
// Returns:
//   - *parser.File: Parsed AST if cache exists and is valid; otherwise nil
//   - error: Returns error if JSON deserialization fails; cache miss returns nil
//
// Edge cases:
//   - Cache file doesn't exist: Returns nil, nil (cache miss, not an error)
//   - Cache file corrupted or invalid JSON format: Returns error
//   - Cache file exists but content is empty: Returns error (empty JSON can't be deserialized)
//   - If dep.json and cache are inconsistent (e.g., cache manually deleted), triggers reparse
//
// Example:
//
//	cached, err := mgr.LoadJSON("src/foo.bp")
//	if err != nil {
//	    // Cache corrupted, need to handle error
//	    return err
//	}
//	if cached == nil {
//	    // Cache miss, need to reparse
//	    parsedFile, _ := parser.ParseFile(...)
//	    mgr.SaveJSON("src/foo.bp", parsedFile)
//	} else {
//	    // Use cached AST
//	    processFile(cached)
//	}
func (m *Manager) LoadJSON(bpFile string) (*parser.File, error) {
	// 获取缓存文件的路径
	jsonPath := m.jsonFilePath(bpFile)

	// 尝试读取缓存文件
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		// 缓存未命中（文件不存在），这不是错误
		// 调用方应该重新解析该文件
		return nil, nil
	}

	// 反序列化 JSON 到 parser.File 结构体
	var file parser.File
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("unmarshal cached ast: %w", err)
	}

	return &file, nil
}

// UpdateHash updates the stored hash value for the specified .bp file.
//
// This function computes the current SHA256 hash of the file and updates
// the in-memory hash table. Note: This function only updates the in-memory
// hash value and does not automatically persist to dep.json.
// To persist changes, call SaveDepFile().
//
// Parameters:
//   - bpFile: Path to the .bp file to update hash for
//
// Returns:
//   - error: Returns error if file hash computation fails
//     - File not found: Returns os.PathError
//     - Read failure: Returns io.ReadError
//     - Success: Returns nil
//
// Edge cases:
//   - If file is first seen, adds to hash table
//   - If file already has hash value, overwrites with new value
//   - Hash is recomputed and updated even if unchanged
//   - Caller must manually call SaveDepFile() to persist
//
// When to use:
//   - When manually modifying .bp files and wanting to update hash without reparsing
//   - Usually not needed after NeedsReparse already updated memory hash
//   - Mainly for external scenarios requiring manual hash updates
//
// Example:
//
//	err := mgr.UpdateHash("src/foo.bp")
//	if err != nil {
//	    return err
//	}
//	// Remember to persist
//	mgr.SaveDepFile()
func (m *Manager) UpdateHash(bpFile string) error {
	// 计算文件的当前哈希值
	hash, err := m.hashFile(bpFile)
	if err != nil {
		return err
	}
	// 更新内存中的哈希值
	m.hashes[bpFile] = hash
	return nil
}
