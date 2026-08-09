package quota

import (
	"context"

	"github.com/google/uuid"

	"mypaas/internal/db"
)

// WithinCreateQuota serializes a user's quota check and the caller's project
// insert in one database transaction. Without this boundary two concurrent
// creates can both observe the same old usage and independently exceed quota.
func (s *Service) WithinCreateQuota(
	ctx context.Context,
	userID uuid.UUID,
	memoryMb int32,
	cpu float64,
	fn func(*db.Queries) error,
) error {
	return s.withUserQuotaLock(ctx, userID, func(txService *Service, queries *db.Queries) error {
		if err := txService.CheckCreate(ctx, userID, memoryMb, cpu); err != nil {
			return err
		}
		return fn(queries)
	})
}

// WithinUpdateQuota uses the same per-user lock as project creation. This is
// required because a create racing with a resource-increasing update can
// otherwise both validate against the same pre-change usage.
func (s *Service) WithinUpdateQuota(
	ctx context.Context,
	project db.Project,
	memoryMb int32,
	cpu float64,
	fn func(*db.Queries) error,
) error {
	return s.withUserQuotaLock(ctx, project.UserID, func(txService *Service, queries *db.Queries) error {
		if err := txService.CheckUpdate(ctx, project, memoryMb, cpu); err != nil {
			return err
		}
		return fn(queries)
	})
}

func (s *Service) withUserQuotaLock(
	ctx context.Context,
	userID uuid.UUID,
	fn func(*Service, *db.Queries) error,
) error {
	return s.queries.InTx(ctx, func(queries *db.Queries) error {
		if err := queries.LockUserQuota(ctx, userID); err != nil {
			return err
		}

		txService := *s
		txService.queries = queries
		return fn(&txService, queries)
	})
}
