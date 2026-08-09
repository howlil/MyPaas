package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"mypaas/internal/db"
	"mypaas/internal/errs"
)

type resourceAccessStore interface {
	GetProjectByID(context.Context, uuid.UUID) (db.Project, error)
	GetDeploymentByID(context.Context, uuid.UUID) (db.Deployment, error)
}

// AuthorizeResourceRequest applies project ownership to every authenticated
// project/deployment URL before the request reaches its handler. This keeps
// nested project resources such as env vars, logs, metrics and lifecycle
// actions behind the same access boundary instead of relying on each handler
// to remember an ownership check.
//
// Platform owners retain administrative access. For non-owners, resources
// owned by another user intentionally resolve as not found to avoid resource
// enumeration.
func AuthorizeResourceRequest(ctx context.Context, requestPath string, store resourceAccessStore, user User) error {
	if user.Role == "owner" {
		return nil
	}

	parts := strings.Split(strings.Trim(requestPath, "/"), "/")
	if len(parts) > 0 && parts[0] == "api" {
		parts = parts[1:]
	}
	if len(parts) < 2 {
		return nil
	}

	id, err := uuid.Parse(parts[1])
	if err != nil {
		// Collection endpoints such as /projects/detect-mode are not scoped to
		// an existing project and continue to their own validation.
		return nil
	}

	switch parts[0] {
	case "projects":
		return authorizeProject(ctx, store, user, id)
	case "deployments":
		deployment, err := store.GetDeploymentByID(ctx, id)
		if err != nil {
			return mapResourceLookupError(err)
		}
		return authorizeProject(ctx, store, user, deployment.ProjectID)
	default:
		return nil
	}
}

func authorizeProject(ctx context.Context, store resourceAccessStore, user User, projectID uuid.UUID) error {
	project, err := store.GetProjectByID(ctx, projectID)
	if err != nil {
		return mapResourceLookupError(err)
	}
	if project.UserID != user.ID {
		return errs.ErrNotFound
	}
	return nil
}

func mapResourceLookupError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return errs.ErrNotFound
	}
	return err
}
