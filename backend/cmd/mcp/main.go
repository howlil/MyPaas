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
	projectID, ok := request.Params.Arguments["project_id"].(string)
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
	projectID, ok := request.Params.Arguments["project_id"].(string)
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
	projectID, ok := request.Params.Arguments["project_id"].(string)
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
	projectID, ok := request.Params.Arguments["project_id"].(string)
	if !ok {
		return mcp.NewToolResultError("project_id must be a string"), nil
	}

	resp, err := doRequest("POST", "/projects/"+projectID+"/start", nil)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText("Project started successfully: " + resp), nil
}
