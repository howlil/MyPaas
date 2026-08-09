package nixpacks

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

type Plan struct {
	Providers  []string            `json:"providers"`
	BuildImage string              `json:"buildImage"`
	Variables  map[string]string   `json:"variables"`
	Phases     map[string]Phase    `json:"phases"`
	Start      *Start              `json:"start"`
}

type Phase struct {
	NixPkgs []string `json:"nixPkgs"`
	AptPkgs []string `json:"aptPkgs"`
	Cmds    []string `json:"cmds"`
}

type Start struct {
	Cmd string `json:"cmd"`
}

// PlanWorkspace runs `nixpacks plan <workspace> -o json` and parses the result.
func PlanWorkspace(ctx context.Context, workspace string) (*Plan, error) {
	cmd := exec.CommandContext(ctx, "nixpacks", "plan", workspace, "-o", "json")
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("nixpacks plan failed: %w, stderr: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("execute nixpacks plan: %w", err)
	}

	var plan Plan
	if err := json.Unmarshal(out, &plan); err != nil {
		return nil, fmt.Errorf("decode nixpacks plan json: %w", err)
	}

	return &plan, nil
}
