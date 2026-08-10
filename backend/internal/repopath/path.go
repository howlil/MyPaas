package repopath

import (
	"fmt"
	"path/filepath"
	"strings"

	"mypaas/internal/errs"
)

// Validate rejects user-controlled repository paths that are absolute,
// platform-specific, contain traversal segments, or otherwise escape the
// repository-relative path contract.
func Validate(value string) error {
	value = strings.TrimSpace(value)
	if value == "" || value == "." {
		return nil
	}
	if strings.ContainsRune(value, '\x00') {
		return fmt.Errorf("%w: repository path contains a NUL byte", errs.ErrValidation)
	}
	if strings.HasPrefix(value, "/") || filepath.IsAbs(value) {
		return fmt.Errorf("%w: repository path %q must be relative", errs.ErrValidation, value)
	}
	if strings.Contains(value, "\\") {
		return fmt.Errorf("%w: repository path %q must use forward slashes", errs.ErrValidation, value)
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == ".." {
			return fmt.Errorf("%w: repository path %q contains parent-directory segments", errs.ErrValidation, value)
		}
	}
	cleaned := filepath.Clean(filepath.FromSlash(value))
	rel, err := filepath.Rel(".", cleaned)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%w: repository path %q escapes the repository root", errs.ErrValidation, value)
	}
	return nil
}
