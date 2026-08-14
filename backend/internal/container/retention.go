package container

import (
	"context"
	"strings"
)

const defaultRetentionUntil = "168h"

// CleanupBuildCache removes BuildKit cache records older than the configured
// retention window. Docker itself keeps cache records that are still required
// by active builds; this is intentionally age-scoped instead of an unbounded
// `docker builder prune -a`.
func (d *DockerCLI) CleanupBuildCache(ctx context.Context, until string) error {
	until = normalizeRetentionUntil(until)
	return runSimple(ctx, "docker", "builder", "prune", "-f", "--filter", "until="+until)
}

func normalizeRetentionUntil(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultRetentionUntil
	}
	return value
}
