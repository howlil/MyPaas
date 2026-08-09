package deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/google/uuid"
	
	"mypaas/internal/db"
)

func needsStaticBuild(workspace string) bool {
	b, err := os.ReadFile(filepath.Join(workspace, "package.json"))
	if err != nil {
		return false
	}
	
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(b, &pkg); err == nil {
		if _, ok := pkg.Scripts["build"]; ok {
			return true
		}
	}
	return false
}

// buildStaticSPA spins up a temporary node container to build the static SPA.
func (s *Service) buildStaticSPA(ctx context.Context, project db.Project, deploymentID uuid.UUID, workspace string, log func(string)) error {
	log("No pre-built static files found. Starting ephemeral static builder...")
	
	// We run an ephemeral container with node:20-alpine
	// Mount the workspace to /app
	// Run npm install && npm run build
	
	cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
		"-v", fmt.Sprintf("%s:/app", workspace),
		"-w", "/app",
		"node:20-alpine",
		"sh", "-c", "npm install && npm run build",
	)
	
	log("Running npm install && npm run build in node:20-alpine container...")
	
	out, err := cmd.CombinedOutput()
	if err != nil {
		log(string(out))
		return fmt.Errorf("static build failed: %w", err)
	}
	
	log(string(out))
	log("Static build completed successfully.")
	return nil
}
