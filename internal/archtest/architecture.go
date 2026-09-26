package archtest

import (
	"path/filepath"
	"strings"
)

// moduleRelativePath returns a stable slash-separated path for diagnostics.
func moduleRelativePath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return strings.ReplaceAll(path, `\`, "/")
	}
	return filepath.ToSlash(rel)
}
