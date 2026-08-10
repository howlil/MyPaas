package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"mypaas/internal/db"
	"mypaas/internal/errs"
)

type fakeResourceAccessStore struct {
	projects    map[uuid.UUID]db.Project
	deployments map[uuid.UUID]db.Deployment
}

func (s *fakeResourceAccessStore) GetProjectByID(_ context.Context, id uuid.UUID) (db.Project, error) {
	project, ok := s.projects[id]
	if !ok {
		return db.Project{}, pgx.ErrNoRows
	}
	return project, nil
}

func (s *fakeResourceAccessStore) GetDeploymentByID(_ context.Context, id uuid.UUID) (db.Deployment, error) {
	deployment, ok := s.deployments[id]
	if !ok {
		return db.Deployment{}, pgx.ErrNoRows
	}
	return deployment, nil
}

func TestAuthorizeResourceRequestProjectOwnership(t *testing.T) {
	ownerID := uuid.New()
	otherID := uuid.New()
	projectID := uuid.New()
	store := &fakeResourceAccessStore{
		projects: map[uuid.UUID]db.Project{
			projectID: {ID: projectID, UserID: ownerID},
		},
		deployments: map[uuid.UUID]db.Deployment{},
	}

	if err := AuthorizeResourceRequest(context.Background(), "/api/projects/"+projectID.String()+"/env/SECRET/reveal", store, User{ID: ownerID, Role: "member"}); err != nil {
		t.Fatalf("owner access rejected: %v", err)
	}
	if err := AuthorizeResourceRequest(context.Background(), "/projects/"+projectID.String()+"/logs", store, User{ID: otherID, Role: "member"}); !errors.Is(err, errs.ErrNotFound) {
		t.Fatalf("cross-user access error = %v, want ErrNotFound", err)
	}
}

func TestAuthorizeResourceRequestDeploymentOwnership(t *testing.T) {
	ownerID := uuid.New()
	otherID := uuid.New()
	projectID := uuid.New()
	deploymentID := uuid.New()
	store := &fakeResourceAccessStore{
		projects: map[uuid.UUID]db.Project{
			projectID: {ID: projectID, UserID: ownerID},
		},
		deployments: map[uuid.UUID]db.Deployment{
			deploymentID: {ID: deploymentID, ProjectID: projectID},
		},
	}

	if err := AuthorizeResourceRequest(context.Background(), "/api/deployments/"+deploymentID.String(), store, User{ID: otherID, Role: "member"}); !errors.Is(err, errs.ErrNotFound) {
		t.Fatalf("cross-user deployment access error = %v, want ErrNotFound", err)
	}
	if err := AuthorizeResourceRequest(context.Background(), "/deployments/"+deploymentID.String()+"/rollback", store, User{ID: ownerID, Role: "member"}); err != nil {
		t.Fatalf("deployment owner access rejected: %v", err)
	}
}

func TestAuthorizeResourceRequestOwnerBypassAndCollectionEndpoints(t *testing.T) {
	store := &fakeResourceAccessStore{
		projects:    map[uuid.UUID]db.Project{},
		deployments: map[uuid.UUID]db.Deployment{},
	}

	if err := AuthorizeResourceRequest(context.Background(), "/api/projects/detect-mode", store, User{ID: uuid.New(), Role: "member"}); err != nil {
		t.Fatalf("collection endpoint rejected: %v", err)
	}
	if err := AuthorizeResourceRequest(context.Background(), "/api/projects/"+uuid.NewString()+"/env", store, User{ID: uuid.New(), Role: "owner"}); err != nil {
		t.Fatalf("platform owner bypass rejected: %v", err)
	}
}
