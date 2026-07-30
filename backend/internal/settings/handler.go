package settings

import (
	"encoding/json"
	"net/http"

	"mypaas/internal/config"
	"mypaas/internal/db"
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
	for key, raw := range overrides {
		if _, ok := merged[key]; ok {
			var v float64
			if json.Unmarshal(raw, &v) == nil {
				merged[key] = v
			}
		}
	}

	httpx.JSON(w, http.StatusOK, merged)
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

func (h *Handler) defaults() map[string]float64 {
	return map[string]float64{
		"user_ram_quota_gb":      float64(h.cfg.UserRAMQuotaMB) / 1024,
		"user_cpu_quota":         h.cfg.UserCPUQuota,
		"max_projects":           float64(h.cfg.MaxProjects),
		"max_concurrent_deploys": float64(h.cfg.MaxConcurrentDeploys),
		"project_default_ram_mb": 512,
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
