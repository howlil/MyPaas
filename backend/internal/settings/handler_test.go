package settings

import (
	"errors"
	"testing"

	"mypaas/internal/config"
	"mypaas/internal/statd"
)

func TestHostTelemetryErrorCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "nil", err: nil, want: ""},
		{name: "protocol", err: &statd.ProtocolError{Code: "INVALID_REQUEST"}, want: "INVALID_REQUEST"},
		{name: "protocol without code", err: &statd.ProtocolError{}, want: "PROTOCOL_ERROR"},
		{name: "invalid config", err: statd.ErrInvalidInput, want: "INVALID_CONFIG"},
		{name: "wrapped invalid config", err: errors.Join(errors.New("outer"), statd.ErrInvalidInput), want: "INVALID_CONFIG"},
		{name: "io", err: errors.New("dial unix: permission denied"), want: "CONNECT_OR_IO_ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hostTelemetryErrorCode(tt.err); got != tt.want {
				t.Fatalf("hostTelemetryErrorCode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		values  map[string]float64
		wantErr bool
	}{
		{name: "valid", values: map[string]float64{"user_ram_quota_gb": 4, "user_cpu_quota": 2, "max_projects": 20, "max_concurrent_deploys": 2, "build_timeout_minutes": 15}},
		{name: "unknown key", values: map[string]float64{"project_default_ram_mb": 512}, wantErr: true},
		{name: "zero quota", values: map[string]float64{"user_ram_quota_gb": 0}, wantErr: true},
		{name: "fractional projects", values: map[string]float64{"max_projects": 2.5}, wantErr: true},
		{name: "excessive concurrency", values: map[string]float64{"max_concurrent_deploys": 33}, wantErr: true},
		{name: "zero timeout", values: map[string]float64{"build_timeout_minutes": 0}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateSettings(tt.values)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateSettings() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSettingsDefaultsOnlyExposeRuntimeBackedValues(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		UserRAMQuotaMB:       4096,
		UserCPUQuota:         2,
		MaxProjects:          12,
		MaxConcurrentDeploys: 3,
		BuildTimeoutMinutes:  20,
	}
	h := &Handler{cfg: cfg}
	defaults := h.defaults()

	if _, ok := defaults["project_default_ram_mb"]; ok {
		t.Fatal("project_default_ram_mb must not be exposed without an authoritative consumer")
	}
	if _, ok := defaults["project_default_cpu"]; ok {
		t.Fatal("project_default_cpu must not be exposed without an authoritative consumer")
	}
	if got := defaults["user_ram_quota_gb"]; got != 4 {
		t.Fatalf("user_ram_quota_gb = %v, want 4", got)
	}
}

func TestApplyToConfig(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	h := &Handler{cfg: cfg}
	h.applyToConfig(map[string]float64{
		"user_ram_quota_gb":      3.5,
		"user_cpu_quota":         1.5,
		"max_projects":           15,
		"max_concurrent_deploys": 2,
		"build_timeout_minutes":  25,
	})

	if cfg.UserRAMQuotaMB != 3584 {
		t.Fatalf("UserRAMQuotaMB = %d, want 3584", cfg.UserRAMQuotaMB)
	}
	if cfg.UserCPUQuota != 1.5 || cfg.MaxProjects != 15 || cfg.MaxConcurrentDeploys != 2 || cfg.BuildTimeoutMinutes != 25 {
		t.Fatalf("runtime config not updated: %+v", cfg)
	}
}
