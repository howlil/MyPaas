package project

import (
	"fmt"
	"strings"
	"unicode"

	"mypaas/internal/errs"
)

const (
	SourceTypeGit      = "git"
	SourceTypeRegistry = "registry"
)

func normalizeSourceType(sourceType, deployMode string) (string, error) {
	sourceType = strings.ToLower(strings.TrimSpace(sourceType))
	if sourceType == "" {
		if strings.TrimSpace(deployMode) == "image" {
			return SourceTypeRegistry, nil
		}
		return SourceTypeGit, nil
	}

	switch sourceType {
	case SourceTypeGit, SourceTypeRegistry:
		return sourceType, nil
	default:
		return "", fmt.Errorf("%w: source type must be git or registry", errs.ErrValidation)
	}
}

func normalizeImageRef(value *string) (*string, error) {
	if value == nil {
		return nil, fmt.Errorf("%w: container image reference is required", errs.ErrValidation)
	}

	image := strings.TrimSpace(*value)
	if image == "" {
		return nil, fmt.Errorf("%w: container image reference is required", errs.ErrValidation)
	}
	if strings.ContainsAny(image, "\r\n\x00") {
		return nil, fmt.Errorf("%w: invalid container image reference", errs.ErrValidation)
	}

	// Registry UIs commonly expose a copyable `docker pull <image>` command.
	// Accept that exact single-image form while keeping the stored value and
	// Docker invocation as an image reference rather than arbitrary shell text.
	fields := strings.Fields(image)
	if len(fields) == 3 && strings.EqualFold(fields[1], "pull") &&
		(strings.EqualFold(fields[0], "docker") || strings.EqualFold(fields[0], "podman")) {
		image = fields[2]
	}

	if len(image) > 512 {
		return nil, fmt.Errorf("%w: container image reference is too long", errs.ErrValidation)
	}
	if strings.HasPrefix(image, "-") || strings.Contains(image, "://") {
		return nil, fmt.Errorf("%w: invalid container image reference", errs.ErrValidation)
	}
	if strings.ContainsFunc(image, unicode.IsSpace) {
		return nil, fmt.Errorf("%w: container image reference cannot contain whitespace", errs.ErrValidation)
	}

	return &image, nil
}

func projectSourceType(deployMode string) string {
	if deployMode == "image" {
		return SourceTypeRegistry
	}
	return SourceTypeGit
}
