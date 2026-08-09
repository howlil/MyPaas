package repopath

import (
	"errors"
	"testing"

	"mypaas/internal/errs"
)

func TestValidateRejectsTraversalAndAbsolutePaths(t *testing.T) {
	for _, value := range []string{"../secret", "frontend/../../secret", "/etc", `frontend\\..\\secret`} {
		if err := Validate(value); !errors.Is(err, errs.ErrValidation) {
			t.Fatalf("Validate(%q) error = %v, want validation error", value, err)
		}
	}
	for _, value := range []string{"", ".", "frontend", "apps/api"} {
		if err := Validate(value); err != nil {
			t.Fatalf("Validate(%q) unexpected error: %v", value, err)
		}
	}
}
