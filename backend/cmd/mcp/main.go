package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var (
	apiURL     string
	apiToken   string
	httpClient = &http.Client{Timeout: 30 * time.Second}
)

type apiErrorBody struct {
	Error struct {
		Code    string          `json:"code"`
		Message string          `json:"message"`
		Details json.RawMessage `json:"details,omitempty"`
	} `json:"error"`
}

func main() {
	apiURL = strings.TrimRight(os.Getenv("MYPAAS_URL"), "/")
	if apiURL == "" {
		apiURL = "http://localhost:8080/api"
	}
	apiToken = os.Getenv("MYPAAS_API_TOKEN")

	s := server.NewMCPServer(
		"MyPaas",
		"1.1.0",
		server.WithToolCapabilities(true),
	)

	registerProjectTools(s)
	registerDeploymentTools(s)
	registerEnvTools(s)
	registerAdminTools(s)

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
		os.Exit(1)
	}
}

func registerProjectTools(s *server.MCPServer) {
	s.AddTool(mcp.NewTool("list_projects",
		mcp.WithDescription("List all projects deployed on MyPaas."),
	), listProjectsHandler)

	s.AddTool(mcp.NewTool("get_project",
		mcp.WithDescription("Get one project by UUID."),
		mcp.WithString("project_id", mcp.Required(), mcp.Description("Project UUID.")),
	), getProjectHandler)

	for _, action := range []struct {
		name        string
		description string
		path        string
		success     string
	}{
		{"deploy_project", "Trigger a deployment for a project.", "deploy", "Deployment triggered"},
		{"start_project", "Start a stopped project.", "start", "Project started"},
		{"stop_project", "Stop a running project.", "stop", "Project stopped"},
		{"restart_project", "Restart a running project.", "restart", "Project restarted"},
	} {
		action := action
		s.AddTool(mcp.NewTool(action.name,
			mcp.WithDescription(action.description),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Project UUID.")),
		), projectActionHandler(action.path, action.success))
	}

	s.AddTool(mcp.NewTool("create_project",
		mcp.WithDescription("Create a MyPaas project. For Git sources, inspect the repository first and pass deployMode/resource fields from detection. For public images, use sourceType=registry, deployMode=image, and imageRef."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Project slug, 3-30 lowercase letters, numbers, and dashes.")),
		mcp.WithString("sourceType", mcp.Description("git or registry. Defaults to git.")),
		mcp.WithString("repoUrl", mcp.Description("Git repository URL for sourceType=git.")),
		mcp.WithString("imageRef", mcp.Description("Public OCI image reference for sourceType=registry.")),
		mcp.WithString("branch", mcp.Description("Git branch. Leave empty to let MyPaas resolve the default branch.")),
		mcp.WithString("deployMode", mcp.Description("dockerfile, compose, static, or image.")),
		mcp.WithString("resourceProfile", mcp.Description("static, go-small, node-python, compose-main, or custom.")),
		mcp.WithNumber("appPort", mcp.Description("Internal app port. Not required for static projects.")),
		mcp.WithNumber("memoryLimitMb", mcp.Description("Main service memory limit in MB.")),
		mcp.WithNumber("cpuLimit", mcp.Description("Main service CPU limit.")),
		mcp.WithBoolean("sharedPostgres", mcp.Description("Provision shared Postgres only when the app explicitly needs it. Defaults to false.")),
		mcp.WithString("mainService", mcp.Description("Compose service receiving public traffic.")),
		mcp.WithString("baseDirectory", mcp.Description("Repo-relative source directory for Git projects.")),
		mcp.WithString("staticFrontendPath", mcp.Description("Repo-relative static frontend directory served by Caddy.")),
		mcp.WithString("composeFilePath", mcp.Description("Repo-relative compose file path.")),
		mcp.WithString("composeOverridePaths", mcp.Description("Comma-separated repo-relative compose override paths.")),
		mcp.WithString("composeProfiles", mcp.Description("Comma-separated compose profile names.")),
		mcp.WithString("composeWorkdir", mcp.Description("Repo-relative compose working directory override.")),
	), createProjectHandler)

	s.AddTool(mcp.NewTool("update_project_settings",
		mcp.WithDescription("Patch mutable project settings. Project name/subdomain are immutable after creation."),
		mcp.WithString("project_id", mcp.Required(), mcp.Description("Project UUID.")),
		mcp.WithString("branch", mcp.Description("Git branch.")),
		mcp.WithString("imageRef", mcp.Description("Registry image reference.")),
		mcp.WithString("resourceProfile", mcp.Description("static, go-small, node-python, compose-main, or custom.")),
		mcp.WithNumber("appPort", mcp.Description("Internal app port.")),
		mcp.WithNumber("memoryLimitMb", mcp.Description("Main service memory limit in MB.")),
		mcp.WithNumber("cpuLimit", mcp.Description("Main service CPU limit.")),
		mcp.WithString("mainService", mcp.Description("Compose public service.")),
		mcp.WithString("baseDirectory", mcp.Description("Repo-relative source directory.")),
		mcp.WithString("staticFrontendPath", mcp.Description("Repo-relative static frontend path, or empty to clear.")),
		mcp.WithString("composeFilePath", mcp.Description("Repo-relative compose file path, or empty to auto-detect.")),
		mcp.WithString("composeOverridePaths", mcp.Description("Comma-separated repo-relative compose override paths.")),
		mcp.WithString("composeProfiles", mcp.Description("Comma-separated compose profile names.")),
		mcp.WithString("composeWorkdir", mcp.Description("Repo-relative compose workdir, or empty to auto.")),
	), updateProjectSettingsHandler)

	s.AddTool(mcp.NewTool("inspect_repository",
		mcp.WithDescription("Inspect a Git repository branch/tree before create or settings update."),
		mcp.WithString("repoUrl", mcp.Required(), mcp.Description("Git repository URL.")),
		mcp.WithString("branch", mcp.Description("Branch to inspect. Empty uses default branch.")),
		mcp.WithString("baseDirectory", mcp.Description("Repo-relative directory to inspect.")),
	), inspectRepositoryHandler)

	s.AddTool(mcp.NewTool("detect_compose",
		mcp.WithDescription("Detect and analyze compose candidates for a Git repository."),
		mcp.WithString("repoUrl", mcp.Required(), mcp.Description("Git repository URL.")),
		mcp.WithString("branch", mcp.Description("Branch to inspect.")),
		mcp.WithString("baseDirectory", mcp.Description("Repo-relative base directory.")),
	), detectComposeHandler)
}

func registerDeploymentTools(s *server.MCPServer) {
	s.AddTool(mcp.NewTool("list_deployments",
		mcp.WithDescription("List deployments for a project."),
		mcp.WithString("project_id", mcp.Required(), mcp.Description("Project UUID.")),
		mcp.WithNumber("limit", mcp.Description("Maximum rows. Default 20.")),
		mcp.WithNumber("offset", mcp.Description("Offset. Default 0.")),
	), listDeploymentsHandler)

	s.AddTool(mcp.NewTool("get_deployment",
		mcp.WithDescription("Get one deployment by UUID."),
		mcp.WithString("deployment_id", mcp.Required(), mcp.Description("Deployment UUID.")),
	), getDeploymentHandler)

	s.AddTool(mcp.NewTool("rollback_deployment",
		mcp.WithDescription("Rollback to a successful deployment."),
		mcp.WithString("deployment_id", mcp.Required(), mcp.Description("Deployment UUID.")),
	), rollbackDeploymentHandler)

	s.AddTool(mcp.NewTool("get_logs",
		mcp.WithDescription("Get recent project logs."),
		mcp.WithString("project_id", mcp.Required(), mcp.Description("Project UUID.")),
		mcp.WithNumber("tail", mcp.Description("Number of log lines. Default 500.")),
	), getLogsHandler)

	s.AddTool(mcp.NewTool("get_metrics_snapshot",
		mcp.WithDescription("Get current project container metrics snapshot."),
		mcp.WithString("project_id", mcp.Required(), mcp.Description("Project UUID.")),
	), getMetricsSnapshotHandler)
}

func registerEnvTools(s *server.MCPServer) {
	s.AddTool(mcp.NewTool("list_env_vars",
		mcp.WithDescription("List environment variable keys for a project. Values are not revealed."),
		mcp.WithString("project_id", mcp.Required(), mcp.Description("Project UUID.")),
	), listEnvVarsHandler)

	s.AddTool(mcp.NewTool("set_env_vars",
		mcp.WithDescription("Bulk set environment variables. vars_json must be a JSON array accepted by the MyPaas env API, for example [{\"key\":\"DATABASE_URL\",\"value\":\"...\"}]."),
		mcp.WithString("project_id", mcp.Required(), mcp.Description("Project UUID.")),
		mcp.WithString("vars_json", mcp.Required(), mcp.Description("JSON array of env var objects.")),
	), setEnvVarsHandler)

	s.AddTool(mcp.NewTool("delete_env_var",
		mcp.WithDescription("Delete one environment variable. confirm_key must exactly match key."),
		mcp.WithString("project_id", mcp.Required(), mcp.Description("Project UUID.")),
		mcp.WithString("key", mcp.Required(), mcp.Description("Environment variable key.")),
		mcp.WithString("confirm_key", mcp.Required(), mcp.Description("Must exactly match key.")),
	), deleteEnvVarHandler)
}

func registerAdminTools(s *server.MCPServer) {
	s.AddTool(mcp.NewTool("get_quota",
		mcp.WithDescription("Get current user's quota usage."),
	), getQuotaHandler)

	s.AddTool(mcp.NewTool("get_host_stats",
		mcp.WithDescription("Get host resource capacity and allocation stats. Requires owner/admin access."),
	), getHostStatsHandler)
}

func doRequest(ctx context.Context, method, path string, body []byte) (string, error) {
	req, err := http.NewRequestWithContext(ctx, method, apiURL+path, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	if apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+apiToken)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode >= 400 {
		return "", formatAPIError(resp.StatusCode, respBody)
	}

	return string(respBody), nil
}

func formatAPIError(statusCode int, respBody []byte) error {
	var apiErr apiErrorBody
	if err := json.Unmarshal(respBody, &apiErr); err == nil && apiErr.Error.Message != "" {
		if apiErr.Error.Code != "" {
			return fmt.Errorf("API error (%d %s): %s", statusCode, apiErr.Error.Code, apiErr.Error.Message)
		}
		return fmt.Errorf("API error (%d): %s", statusCode, apiErr.Error.Message)
	}
	if len(respBody) == 0 {
		return fmt.Errorf("API error (%d)", statusCode)
	}
	return fmt.Errorf("API error (%d): %s", statusCode, string(respBody))
}

func toolArgs(request mcp.CallToolRequest) (map[string]interface{}, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments")
	}
	return args, nil
}

func requiredString(args map[string]interface{}, key string) (string, error) {
	value, ok := args[key].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s must be a non-empty string", key)
	}
	return value, nil
}

func optionalString(args map[string]interface{}, key string) (string, bool, error) {
	value, ok := args[key]
	if !ok {
		return "", false, nil
	}
	text, ok := value.(string)
	if !ok {
		return "", false, fmt.Errorf("%s must be a string", key)
	}
	return strings.TrimSpace(text), true, nil
}

func optionalNumber(args map[string]interface{}, key string) (float64, bool, error) {
	value, ok := args[key]
	if !ok {
		return 0, false, nil
	}
	number, ok := value.(float64)
	if !ok {
		return 0, false, fmt.Errorf("%s must be a number", key)
	}
	return number, true, nil
}

func optionalBool(args map[string]interface{}, key string) (bool, bool, error) {
	value, ok := args[key]
	if !ok {
		return false, false, nil
	}
	flag, ok := value.(bool)
	if !ok {
		return false, false, fmt.Errorf("%s must be a boolean", key)
	}
	return flag, true, nil
}

func commaList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func pathID(id string) string {
	return url.PathEscape(id)
}

func resultFromRequest(ctx context.Context, method, path string, body []byte, prefix string) (*mcp.CallToolResult, error) {
	resp, err := doRequest(ctx, method, path, body)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if prefix == "" {
		return mcp.NewToolResultText(resp), nil
	}
	return mcp.NewToolResultText(prefix + ": " + resp), nil
}

func listProjectsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return resultFromRequest(ctx, http.MethodGet, "/projects", nil, "")
}

func getProjectHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, err := toolArgs(request)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	projectID, err := requiredString(args, "project_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return resultFromRequest(ctx, http.MethodGet, "/projects/"+pathID(projectID), nil, "")
}

func projectActionHandler(action, success string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := toolArgs(request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		projectID, err := requiredString(args, "project_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return resultFromRequest(ctx, http.MethodPost, "/projects/"+pathID(projectID)+"/"+action, nil, success)
	}
}

func createProjectHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, err := toolArgs(request)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	name, err := requiredString(args, "name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	payload := map[string]interface{}{"name": name}
	sourceType, _, err := addOptionalString(payload, args, "sourceType")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if sourceType == "" {
		sourceType = "git"
		payload["sourceType"] = sourceType
	}

	for _, key := range []string{
		"repoUrl", "imageRef", "branch", "deployMode", "resourceProfile", "mainService",
		"baseDirectory", "staticFrontendPath", "composeFilePath", "composeWorkdir",
	} {
		if _, _, err := addOptionalString(payload, args, key); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
	}
	if value, ok, err := optionalString(args, "composeOverridePaths"); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	} else if ok {
		payload["composeOverridePaths"] = commaList(value)
	}
	if value, ok, err := optionalString(args, "composeProfiles"); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	} else if ok {
		payload["composeProfiles"] = commaList(value)
	}
	for _, key := range []string{"appPort", "memoryLimitMb", "cpuLimit"} {
		if err := addOptionalNumber(payload, args, key); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
	}
	if sharedPostgres, ok, err := optionalBool(args, "sharedPostgres"); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	} else if ok {
		payload["sharedPostgres"] = sharedPostgres
	} else {
		payload["sharedPostgres"] = false
	}

	if sourceType == "git" {
		if repoURL, _ := payload["repoUrl"].(string); strings.TrimSpace(repoURL) == "" {
			return mcp.NewToolResultError("repoUrl is required for sourceType=git"), nil
		}
	}
	if sourceType == "registry" {
		if imageRef, _ := payload["imageRef"].(string); strings.TrimSpace(imageRef) == "" {
			return mcp.NewToolResultError("imageRef is required for sourceType=registry"), nil
		}
		if _, ok := payload["deployMode"]; !ok {
			payload["deployMode"] = "image"
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return resultFromRequest(ctx, http.MethodPost, "/projects", body, "Project created")
}

func updateProjectSettingsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, err := toolArgs(request)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	projectID, err := requiredString(args, "project_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	payload := map[string]interface{}{}
	for _, key := range []string{
		"branch", "imageRef", "resourceProfile", "mainService", "baseDirectory",
		"staticFrontendPath", "composeFilePath", "composeWorkdir",
	} {
		if _, _, err := addOptionalString(payload, args, key); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
	}
	if value, ok, err := optionalString(args, "composeOverridePaths"); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	} else if ok {
		payload["composeOverridePaths"] = commaList(value)
	}
	if value, ok, err := optionalString(args, "composeProfiles"); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	} else if ok {
		payload["composeProfiles"] = commaList(value)
	}
	for _, key := range []string{"appPort", "memoryLimitMb", "cpuLimit"} {
		if err := addOptionalNumber(payload, args, key); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
	}
	if len(payload) == 0 {
		return mcp.NewToolResultError("at least one setting field is required"), nil
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return resultFromRequest(ctx, http.MethodPatch, "/projects/"+pathID(projectID), body, "Project updated")
}

func inspectRepositoryHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, err := toolArgs(request)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	payload, err := sourceDetectionPayload(args)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	payload["inspectOnly"] = true
	body, err := json.Marshal(payload)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return resultFromRequest(ctx, http.MethodPost, "/projects/detect-mode", body, "")
}

func detectComposeHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, err := toolArgs(request)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	payload, err := sourceDetectionPayload(args)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return resultFromRequest(ctx, http.MethodPost, "/projects/detect-compose", body, "")
}

func sourceDetectionPayload(args map[string]interface{}) (map[string]interface{}, error) {
	repoURL, err := requiredString(args, "repoUrl")
	if err != nil {
		return nil, err
	}
	payload := map[string]interface{}{"repoUrl": repoURL}
	for _, key := range []string{"branch", "baseDirectory"} {
		if _, _, err := addOptionalString(payload, args, key); err != nil {
			return nil, err
		}
	}
	return payload, nil
}

func listDeploymentsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, err := toolArgs(request)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	projectID, err := requiredString(args, "project_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	limit := 20
	offset := 0
	if value, ok, err := optionalNumber(args, "limit"); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	} else if ok {
		limit = int(value)
	}
	if value, ok, err := optionalNumber(args, "offset"); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	} else if ok {
		offset = int(value)
	}
	query := url.Values{}
	query.Set("limit", fmt.Sprintf("%d", limit))
	query.Set("offset", fmt.Sprintf("%d", offset))
	return resultFromRequest(ctx, http.MethodGet, "/projects/"+pathID(projectID)+"/deployments?"+query.Encode(), nil, "")
}

func getDeploymentHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return deploymentByID(ctx, request, http.MethodGet, "", "")
}

func rollbackDeploymentHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return deploymentByID(ctx, request, http.MethodPost, "rollback", "Rollback triggered")
}

func deploymentByID(ctx context.Context, request mcp.CallToolRequest, method, suffix, prefix string) (*mcp.CallToolResult, error) {
	args, err := toolArgs(request)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	deploymentID, err := requiredString(args, "deployment_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	path := "/deployments/" + pathID(deploymentID)
	if suffix != "" {
		path += "/" + suffix
	}
	return resultFromRequest(ctx, method, path, nil, prefix)
}

func getLogsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, err := toolArgs(request)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	projectID, err := requiredString(args, "project_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	tail := 500
	if value, ok, err := optionalNumber(args, "tail"); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	} else if ok {
		tail = int(value)
	}
	query := url.Values{}
	query.Set("tail", fmt.Sprintf("%d", tail))
	return resultFromRequest(ctx, http.MethodGet, "/projects/"+pathID(projectID)+"/logs?"+query.Encode(), nil, "")
}

func getMetricsSnapshotHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, err := toolArgs(request)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	projectID, err := requiredString(args, "project_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return resultFromRequest(ctx, http.MethodGet, "/projects/"+pathID(projectID)+"/metrics", nil, "")
}

func listEnvVarsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, err := toolArgs(request)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	projectID, err := requiredString(args, "project_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return resultFromRequest(ctx, http.MethodGet, "/projects/"+pathID(projectID)+"/env", nil, "")
}

func setEnvVarsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, err := toolArgs(request)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	projectID, err := requiredString(args, "project_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	varsJSON, err := requiredString(args, "vars_json")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	var vars json.RawMessage
	if err := json.Unmarshal([]byte(varsJSON), &vars); err != nil {
		return mcp.NewToolResultError("vars_json must be valid JSON"), nil
	}
	body, err := json.Marshal(map[string]json.RawMessage{"vars": vars})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return resultFromRequest(ctx, http.MethodPut, "/projects/"+pathID(projectID)+"/env", body, "Environment updated")
}

func deleteEnvVarHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, err := toolArgs(request)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	projectID, err := requiredString(args, "project_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	key, err := requiredString(args, "key")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	confirmKey, err := requiredString(args, "confirm_key")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if confirmKey != key {
		return mcp.NewToolResultError("confirm_key must exactly match key"), nil
	}
	return resultFromRequest(ctx, http.MethodDelete, "/projects/"+pathID(projectID)+"/env/"+pathID(key), nil, "Environment variable deleted")
}

func getQuotaHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return resultFromRequest(ctx, http.MethodGet, "/me/quota", nil, "")
}

func getHostStatsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return resultFromRequest(ctx, http.MethodGet, "/admin/host-stats", nil, "")
}

func addOptionalString(payload map[string]interface{}, args map[string]interface{}, key string) (string, bool, error) {
	value, ok, err := optionalString(args, key)
	if err != nil || !ok {
		return value, ok, err
	}
	if value == "" {
		payload[key] = nil
	} else {
		payload[key] = value
	}
	return value, true, nil
}

func addOptionalNumber(payload map[string]interface{}, args map[string]interface{}, key string) error {
	value, ok, err := optionalNumber(args, key)
	if err != nil || !ok {
		return err
	}
	if key == "appPort" || key == "memoryLimitMb" {
		payload[key] = int(value)
		return nil
	}
	payload[key] = value
	return nil
}
