package container

import "testing"

func TestNormalizeRetentionUntilDefaultsToSevenDays(t *testing.T) {
	if got := normalizeRetentionUntil(""); got != "168h" {
		t.Fatalf("normalizeRetentionUntil() = %q, want 168h", got)
	}
}

func TestNormalizeRetentionUntilTrimsConfiguredValue(t *testing.T) {
	if got := normalizeRetentionUntil(" 336h "); got != "336h" {
		t.Fatalf("normalizeRetentionUntil() = %q, want 336h", got)
	}
}
