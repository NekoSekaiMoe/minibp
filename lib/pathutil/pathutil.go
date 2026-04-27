// Package pathutil provides path manipulation utilities.
package pathutil

import "strings"

// SanitizePath removes '..' from a path to prevent directory traversal.
// It repeatedly replaces "../" and "..\" with an empty string until no
// more occurrences are found. This is a simple but effective way to
// mitigate path traversal vulnerabilities.
func SanitizePath(path string) string {
	for {
		cleaned := strings.ReplaceAll(path, "../", "")
		cleaned = strings.ReplaceAll(cleaned, "..\\", "")
		if cleaned == path {
			return cleaned
		}
		path = cleaned
	}
}
