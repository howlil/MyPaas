package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var (
	apiURL   string
	apiToken string
)

func main() {
	apiURL = os.Getenv("MYPAAS_URL")
	if apiURL == "" {
		// Try to fallback to localhost for development
		apiURL = "http://localhost:8080/api"
	}
	apiToken = os.Getenv("MYPAAS_API_TOKEN")

	s := server.NewMCPServer(
		"MyPaas",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Tool: list_projects
	listProjectsTool := mcp.NewTool("list_projects",
		mcp.WithDescription("List all projects deployed on MyPaas"),
	)
	s.AddTool(listProjectsTool, listProjectsHandler)

	// Tool: deploy_project
	deployProjectTool := mcp.NewTool("deploy_project",
		mcp.WithDescription("Trigger a deployment for a specific project"),
		mcp.WithString("project_id",
			mcp.Required(),
			mcp.Description("The UUID of the project to deploy"),
		),
	)
	s.AddTool(deployProjectTool, deployProjectHandler)

	// Tool: get_project
	getProjectTool := mcp.NewTool("get_project",
		mcp.WithDescription("Get details of a specific project"),
		mcp.WithString("project_id",
			mcp.Required(),
			mcp.Description("The UUID of the project"),
		),
	)
	s.AddTool(getProjectTool, getProjectHandler)

	// Tool: stop_project
	stopProjectTool := mcp.NewTool("stop_project",
		mcp.WithDescription("Stop a running project"),
		mcp.WithString("project_id",
			mcp.Required(),
			mcp.Description("The UUID of the project to stop"),
		),
	)
	s.AddTool(stopProjectTool, stopProjectHandler)

	// Tool: start_project
	startProjectTool := mcp.NewTool("start_project",
		mcp.WithDescription("Start a stopped project"),
		mcp.WithString("project_id",
			mcp.Required(),
			mcp.Description("The UUID of the project to start"),
		),
	)
	s.AddTool(startProjectTool, startProjectHandler)

	// Tool: create_project
	createProjectTool := mcp.NewTool("create_project",
		mcp.WithDescription("Create a new project in MyPaas from a Git repository. AGENT INSTRUCTION: Before calling, analyze the repo to infer parameters:\n1. deployMode: 'compose' (if docker-compose.yml exists), 'dockerfile' (if Dockerfile exists), or 'static' (plain HTML/JS).\n2. resourceProfile: 'compose-main' (for compose), 'node-python' (Node/Python/PHP), 'go-small' (Go), or 'static'.\n3. sharedPostgres: true ONLY if the app explicitly requires a PostgreSQL database (e.g., uses DATABASE_URL).\n4. appPort: The port the app listens on internally (check EXPOSE in Dockerfile or server code)."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the project (e.g., my-awesome-app)")),
		mcp.WithString("repoUrl", mcp.Required(), mcp.Description("GitHub Repository URL (e.g., https://github.com/user/repo)")),
		mcp.WithString("branch", mcp.Required(), mcp.Description("Git branch to deploy (e.g., main)")),
		mcp.WithString("deployMode", mcp.Required(), mcp.Description("Deploy mode: 'dockerfile', 'compose', or 'static'")),
		mcp.WithString("resourceProfile", mcp.Description("Resource profile: 'static', 'go-small', 'node-python', 'compose-main', 'custom' (default: 'go-small')")),
		mcp.WithNumber("appPort", mcp.Description("Port the app listens on (default: 3000, 80 for static)")),
		mcp.WithBoolean("sharedPostgres", mcp.Description("Whether to provision a shared Postgres database (default: true)")),
	)
	s.AddTool(createProjectTool, createProjectHandler)

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
		os.Exit(1)
	}
}

func doRequest(method, path string, body []byte) (string, error) {
	req, err := http.NewRequest(method, apiURL+path, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	if apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+apiToken)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	return string(respBody), nil
}

func listProjectsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := doRequest("GET", "/projects", nil)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(resp), nil
}

func deployProjectHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid arguments"), nil
	}
	projectID, ok := args["project_id"].(string)
	if !ok {
		return mcp.NewToolResultError("project_id must be a string"), nil
	}

	resp, err := doRequest("POST", "/projects/"+projectID+"/deploy", nil)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText("Deployment triggered successfully: " + resp), nil
}

func getProjectHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid arguments"), nil
	}
	projectID, ok := args["project_id"].(string)
	if !ok {
		return mcp.NewToolResultError("project_id must be a string"), nil
	}

	resp, err := doRequest("GET", "/projects/"+projectID, nil)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(resp), nil
}

func stopProjectHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid arguments"), nil
	}
	projectID, ok := args["project_id"].(string)
	if !ok {
		return mcp.NewToolResultError("project_id must be a string"), nil
	}

	resp, err := doRequest("POST", "/projects/"+projectID+"/stop", nil)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText("Project stopped successfully: " + resp), nil
}

func startProjectHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid arguments"), nil
	}
	projectID, ok := args["project_id"].(string)
	if !ok {
		return mcp.NewToolResultError("project_id must be a string"), nil
	}

	resp, err := doRequest("POST", "/projects/"+projectID+"/start", nil)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText("Project started successfully: " + resp), nil
}

func createProjectHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid arguments"), nil
	}
	name, _ := args["name"].(string)
	repoUrl, _ := args["repoUrl"].(string)
	branch, _ := args["branch"].(string)
	deployMode, _ := args["deployMode"].(string)

	resourceProfile, _ := args["resourceProfile"].(string)

	appPort := 3000
	if p, ok := args["appPort"].(float64); ok {
		appPort = int(p)
	} else if deployMode == "static" {
		appPort = 80
	}

	sharedPostgres, ok := args["sharedPostgres"].(bool)
	if !ok {
		sharedPostgres = true // default to true
	}

	payload := map[string]interface{}{
		"name":            name,
		"repoUrl":         repoUrl,
		"branch":          branch,
		"deployMode":      deployMode,
		"resourceProfile": resourceProfile,
		"appPort":         appPort,
		"sharedPostgres":  sharedPostgres,
	}

	body, _ := json.Marshal(payload)
	resp, err := doRequest("POST", "/projects", body)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText("Project created successfully: " + resp), nil
}

