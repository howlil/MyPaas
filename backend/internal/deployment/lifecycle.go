package deployment

import (
	"context"
	"log/slog"

	"mypaas/internal/db"
)

// ReconcileMissingRuntimes is the startup recovery path for container-backed
// projects. Static projects intentionally have no Docker stack and are restored
// by route reconciliation instead. Container/Compose projects use the same
// immutable creation-time project name as the rest of the deployment engine.
func (s *Service) ReconcileMissingRuntimes(ctx context.Context) error {
	projects, err := s.queries.ListRoutableProjects(ctx)
	if err != nil {
		return err
	}
	for _, project := range projects {
		if !projectHasContainerRuntime(project) {
			continue
		}
		name := runtimeStackName(project)
		if s.docker.StackExists(ctx, name, project.DeployMode) {
			continue
		}
		slog.Info("reconciler: runtime missing for running project, triggering deployment",
			"project", project.Name,
			"id", project.ID,
			"mode", project.DeployMode,
			"runtime", name,
		)
		deployment, err := s.queries.CreateDeployment(ctx, db.CreateDeploymentParams{
			ProjectID:   project.ID,
			TriggeredBy: "manual",
		})
		if err != nil {
			slog.Error("reconciler: failed to create recovery deployment", "project", project.Name, "error", err)
			continue
		}
		go s.runDeployment(project.ID, deployment.ID)
	}
	return nil
}

func projectHasContainerRuntime(project db.Project) bool {
	return project.DeployMode == "dockerfile" || project.DeployMode == "compose" || project.DeployMode == "image"
}

func runtimeStackName(project db.Project) string {
	return "mypaas-" + project.Name
}
