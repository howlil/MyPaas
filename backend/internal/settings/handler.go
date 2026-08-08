package settings

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"mypaas/internal/config"
	"mypaas/internal/db"
	"mypaas/internal/host"
	"mypaas/internal/httpx"
)

// settingKeys lists the keys that can be overridden via the API.
var settingKeys = []string{
	"user_ram_quota_gb",
	"user_cpu_quota",
	"max_projects",
	"max_concurrent_deploys",
	"project_default_ram_mb",
	"project_default_cpu",
	"build_timeout_minutes",
}

type Handler struct {
	queries *db.Queries
	cfg     *config.Config
}

func NewHandler(queries *db.Queries, cfg *config.Config) *Handler {
	return &Handler{queries: queries, cfg: cfg}
}

// Get returns the current platform settings merged from env defaults and DB
// overrides. Only whitelisted keys are exposed.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	rows, err := h.queries.GetAllSettings(r.Context())
	if err != nil {
		httpx.DomainError(w, err)
		return
	}

	overrides := make(map[string]json.RawMessage, len(rows))
	for _, row := range rows {
		overrides[row.Key] = row.Value
	}

	merged := h.defaults()
	res := make(map[string]interface{})
	for k, v := range merged {
		res[k] = v
	}

	for key, raw := range overrides {
		if _, ok := merged[key]; ok {
			var v float64
			if json.Unmarshal(raw, &v) == nil {
				res[key] = v
			}
		}
	}

	res["mcp_api_token"] = h.cfg.ApiToken
	res["cloudflare_configured"] = h.cfg.CloudflareAPIToken != "" && h.cfg.CloudflareZoneID != ""

	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) RegenerateMCPToken(w http.ResponseWriter, r *http.Request) {
	randBytes := make([]byte, 24)
	rand.Read(randBytes)
	newToken := "mp_" + hex.EncodeToString(randBytes)

	err := updateEnvFile("MYPAAS_API_TOKEN", newToken)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "ENV_WRITE_FAILED", "Failed to write .env file: "+err.Error(), nil)
		return
	}

	h.cfg.ApiToken = newToken
	h.Get(w, r)
}

type cloudflareReq struct {
	Token  string `json:"token"`
	ZoneID string `json:"zone_id"`
}

func (h *Handler) UpdateCloudflareConfig(w http.ResponseWriter, r *http.Request) {
	var req cloudflareReq
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body", nil)
		return
	}

	if err := updateEnvFile("CLOUDFLARE_API_TOKEN", req.Token); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "ENV_WRITE_FAILED", "Failed to save token", nil)
		return
	}
	if err := updateEnvFile("CLOUDFLARE_ZONE_ID", req.ZoneID); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "ENV_WRITE_FAILED", "Failed to save zone ID", nil)
		return
	}

	h.cfg.CloudflareAPIToken = req.Token
	h.cfg.CloudflareZoneID = req.ZoneID

	h.Get(w, r)
}

func updateEnvFile(key, value string) error {
	var envPath string
	if configDir := os.Getenv("MYPAAS_CONFIG_DIR"); configDir != "" {
		envPath = filepath.Join(configDir, ".env")
	} else {
		for _, p := range []string{".env", "../.env", "../../.env", "../../../.env"} {
			if _, err := os.Stat(p); err == nil {
				envPath = p
				break
			}
		}
	}
	if envPath == "" {
		envPath = ".env" // Fallback to current directory
	}

	content, err := os.ReadFile(envPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	var lines []string
	if len(content) > 0 {
		lines = strings.Split(string(content), "\n")
	}
	found := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), key+"=") {
			lines[i] = fmt.Sprintf("%s=%q", key, value)
			found = true
			break
		}
	}
	if !found {
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			lines = append(lines, "")
		}
		lines = append(lines, fmt.Sprintf("%s=%q", key, value))
	}

	return os.WriteFile(envPath, []byte(strings.Join(lines, "\n")), 0600)
}

// Update upserts one or more platform settings and applies them to the
// running config so changes take effect immediately.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req map[string]float64
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_BODY", "Request body must be a JSON object with numeric values.", nil)
		return
	}

	allowed := make(map[string]struct{}, len(settingKeys))
	for _, k := range settingKeys {
		allowed[k] = struct{}{}
	}

	for key, value := range req {
		if _, ok := allowed[key]; !ok {
			continue
		}
		raw, err := json.Marshal(value)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "INVALID_VALUE", "Cannot encode value for "+key, nil)
			return
		}
		if err := h.queries.UpsertSetting(r.Context(), db.UpsertSettingParams{
			Key:   key,
			Value: raw,
		}); err != nil {
			httpx.DomainError(w, err)
			return
		}
	}

	// Apply overrides to the in-memory config so they take effect immediately.
	h.applyToConfig(req)

	// Return the merged view so the frontend can confirm.
	h.Get(w, r)
}

// HostStats returns the physical capacity of the VM and how much is allocated.
func (h *Handler) HostStats(w http.ResponseWriter, r *http.Request) {
	cap := host.GetCapacity()

	// Calculate total allocated RAM across ALL projects
	usage, err := h.queries.GetGlobalResourceUsage(r.Context())
	var allocatedRAM int32
	var allocatedCPU float64
	if err == nil {
		allocatedRAM = usage.TotalMemoryMb
		if usage.TotalCpu.Valid && usage.TotalCpu.Int != nil {
			// quick numeric conversion
			cpuVal, _ := usage.TotalCpu.Float64Value()
			allocatedCPU = cpuVal.Float64
		}
	}

	res := map[string]interface{}{
		"host_ram_bytes": cap.TotalRAMBytes,
		"host_cpu_cores": cap.TotalCPUCores,
		"allocated_ram_mb": allocatedRAM,
		"allocated_cpu": allocatedCPU,
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) defaults() map[string]float64 {
	cap := host.GetCapacity()
	
	// Smart defaults based on physical capacity
	defaultUserRAM := float64(h.cfg.UserRAMQuotaMB) / 1024
	defaultProjectRAM := float64(512)

	// If VM has <= 2GB RAM (approx 2048 * 1024 * 1024 = 2147483648)
	// Give max 1.5GB to user, default 256MB per project.
	if cap.TotalRAMBytes > 0 && cap.TotalRAMBytes <= (2500*1024*1024) {
		defaultUserRAM = 1.5
		defaultProjectRAM = 256
	}

	return map[string]float64{
		"user_ram_quota_gb":      defaultUserRAM,
		"user_cpu_quota":         h.cfg.UserCPUQuota,
		"max_projects":           float64(h.cfg.MaxProjects),
		"max_concurrent_deploys": float64(h.cfg.MaxConcurrentDeploys),
		"project_default_ram_mb": defaultProjectRAM,
		"project_default_cpu":    0.5,
		"build_timeout_minutes":  float64(h.cfg.BuildTimeoutMinutes),
	}
}

func (h *Handler) applyToConfig(values map[string]float64) {
	if v, ok := values["user_ram_quota_gb"]; ok {
		h.cfg.UserRAMQuotaMB = int32(v * 1024)
	}
	if v, ok := values["user_cpu_quota"]; ok {
		h.cfg.UserCPUQuota = v
	}
	if v, ok := values["max_projects"]; ok {
		h.cfg.MaxProjects = int32(v)
	}
	if v, ok := values["max_concurrent_deploys"]; ok {
		h.cfg.MaxConcurrentDeploys = int(v)
	}
	if v, ok := values["build_timeout_minutes"]; ok {
		h.cfg.BuildTimeoutMinutes = int(v)
	}
}
