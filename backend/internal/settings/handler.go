package settings

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"

	"mypaas/internal/config"
	"mypaas/internal/db"
	"mypaas/internal/host"
	"mypaas/internal/httpx"
	"mypaas/internal/statd"
)

// settingKeys lists numeric settings that have an authoritative runtime
// consumer. Project resource defaults intentionally live in resource profiles;
// keeping a second admin-level default would create two competing sources of
// truth for new-project configuration.
var settingKeys = []string{
	"user_ram_quota_gb",
	"user_cpu_quota",
	"max_projects",
	"max_concurrent_deploys",
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
	_, _ = rand.Read(randBytes)
	newToken := "mp_" + hex.EncodeToString(randBytes)

	rawToken, _ := json.Marshal(newToken)
	err := h.queries.UpsertSetting(r.Context(), db.UpsertSettingParams{
		Key:   "mypaas_api_token",
		Value: rawToken,
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_WRITE_FAILED", "Failed to save token to database: "+err.Error(), nil)
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

	rawToken, _ := json.Marshal(req.Token)
	if err := h.queries.UpsertSetting(r.Context(), db.UpsertSettingParams{
		Key:   "cloudflare_api_token",
		Value: rawToken,
	}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_WRITE_FAILED", "Failed to save cloudflare token", nil)
		return
	}

	rawZone, _ := json.Marshal(req.ZoneID)
	if err := h.queries.UpsertSetting(r.Context(), db.UpsertSettingParams{
		Key:   "cloudflare_zone_id",
		Value: rawZone,
	}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_WRITE_FAILED", "Failed to save zone ID", nil)
		return
	}

	h.cfg.CloudflareAPIToken = req.Token
	h.cfg.CloudflareZoneID = req.ZoneID

	h.Get(w, r)
}

// Update upserts one or more platform settings and applies the supported
// runtime values to the in-memory config.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req map[string]float64
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_BODY", "Request body must be a JSON object with numeric values.", nil)
		return
	}
	if err := validateSettings(req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_SETTING", err.Error(), nil)
		return
	}

	for key, value := range req {
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

	h.applyToConfig(req)
	h.Get(w, r)
}

func validateSettings(values map[string]float64) error {
	allowed := make(map[string]struct{}, len(settingKeys))
	for _, key := range settingKeys {
		allowed[key] = struct{}{}
	}
	for key, value := range values {
		if _, ok := allowed[key]; !ok {
			return fmt.Errorf("unknown platform setting %q", key)
		}
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return fmt.Errorf("%s must be a finite number", key)
		}
		switch key {
		case "user_ram_quota_gb":
			if value <= 0 || value > 1024 {
				return errors.New("user RAM quota must be greater than 0 and at most 1024 GB")
			}
		case "user_cpu_quota":
			if value <= 0 || value > 256 {
				return errors.New("user CPU quota must be greater than 0 and at most 256 cores")
			}
		case "max_projects":
			if value < 1 || value > 10000 || value != math.Trunc(value) {
				return errors.New("maximum projects must be a whole number between 1 and 10000")
			}
		case "max_concurrent_deploys":
			if value < 1 || value > 32 || value != math.Trunc(value) {
				return errors.New("concurrent deployments must be a whole number between 1 and 32")
			}
		case "build_timeout_minutes":
			if value < 1 || value > 1440 || value != math.Trunc(value) {
				return errors.New("build timeout must be a whole number between 1 and 1440 minutes")
			}
		}
	}
	return nil
}

type hostStatsResponse struct {
	HostRAMBytes       int64                      `json:"host_ram_bytes"`
	HostCPUCores       int                        `json:"host_cpu_cores"`
	AllocatedRAMMB     int32                      `json:"allocated_ram_mb"`
	AllocatedCPU       float64                    `json:"allocated_cpu"`
	TelemetryStatus    string                     `json:"telemetry_status"`
	TelemetryErrorCode string                     `json:"telemetry_error_code,omitempty"`
	Memory             *statd.HostMemorySnapshot  `json:"memory"`
	CPU                *statd.HostCPUSnapshot     `json:"cpu"`
	Storage            *statd.HostStorageSnapshot `json:"storage"`
	Network            *statd.HostNetworkSnapshot `json:"network"`
}

func hostTelemetryErrorCode(err error) string {
	if err == nil {
		return ""
	}
	var protocolErr *statd.ProtocolError
	if errors.As(err, &protocolErr) {
		if protocolErr.Code != "" {
			return protocolErr.Code
		}
		return "PROTOCOL_ERROR"
	}
	if errors.Is(err, statd.ErrInvalidInput) {
		return "INVALID_CONFIG"
	}
	return "CONNECT_OR_IO_ERROR"
}

// HostStats returns host capacity plus optional host telemetry from mypaas-statd.
// Capacity/allocation data remains usable when host telemetry is disabled or unavailable.
// telemetry_status and telemetry_error_code make the fail-open path observable without
// exposing raw socket or filesystem errors to the dashboard.
func (h *Handler) HostStats(w http.ResponseWriter, r *http.Request) {
	cap := host.GetCapacity()

	usage, err := h.queries.GetGlobalResourceUsage(r.Context())
	var allocatedRAM int32
	var allocatedCPU float64
	if err == nil {
		allocatedRAM = usage.TotalMemoryMb
		if usage.TotalCpu.Valid && usage.TotalCpu.Int != nil {
			cpuVal, _ := usage.TotalCpu.Float64Value()
			allocatedCPU = cpuVal.Float64
		}
	}

	var memory *statd.HostMemorySnapshot
	var cpu *statd.HostCPUSnapshot
	var storage *statd.HostStorageSnapshot
	var network *statd.HostNetworkSnapshot
	telemetryStatus := "disabled"
	telemetryErrorCode := ""
	if socketPath := strings.TrimSpace(os.Getenv("STATD_SOCKET")); socketPath != "" {
		telemetryStatus = "unavailable"
		snapshot, snapshotErr := statd.NewClient(socketPath).HostSnapshot(r.Context())
		if snapshotErr == nil {
			memory = snapshot.Memory
			cpu = snapshot.CPU
			storage = snapshot.Storage
			network = snapshot.Network
			if memory != nil || cpu != nil || storage != nil || network != nil {
				telemetryStatus = "available"
			} else {
				telemetryErrorCode = "EMPTY_SNAPSHOT"
			}
		} else {
			telemetryErrorCode = hostTelemetryErrorCode(snapshotErr)
		}
	}

	httpx.JSON(w, http.StatusOK, hostStatsResponse{
		HostRAMBytes:       cap.TotalRAMBytes,
		HostCPUCores:       cap.TotalCPUCores,
		AllocatedRAMMB:     allocatedRAM,
		AllocatedCPU:       allocatedCPU,
		TelemetryStatus:    telemetryStatus,
		TelemetryErrorCode: telemetryErrorCode,
		Memory:             memory,
		CPU:                cpu,
		Storage:            storage,
		Network:            network,
	})
}

func (h *Handler) defaults() map[string]float64 {
	return map[string]float64{
		"user_ram_quota_gb":      float64(h.cfg.UserRAMQuotaMB) / 1024,
		"user_cpu_quota":         h.cfg.UserCPUQuota,
		"max_projects":           float64(h.cfg.MaxProjects),
		"max_concurrent_deploys": float64(h.cfg.MaxConcurrentDeploys),
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
