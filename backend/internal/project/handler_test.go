package project

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type handlerErrorResponse struct {
	Error struct {
		Code string `json:"code"`
	} `json:"error"`
}

func TestDetectModeAcceptsBaseDirectory(t *testing.T) {
	h := &Handler{service: &Service{}}
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/projects/detect-mode",
		strings.NewReader(`{"repoUrl":"","branch":"main","inspectOnly":true,"baseDirectory":"docs"}`),
	)
	rec := httptest.NewRecorder()

	h.DetectMode(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("DetectMode() status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var payload handlerErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Error.Code == "INVALID_JSON" {
		t.Fatalf("DetectMode() rejected baseDirectory as invalid JSON: %s", rec.Body.String())
	}
	if payload.Error.Code != "VALIDATION_FAILED" {
		t.Fatalf("DetectMode() error code = %q, want VALIDATION_FAILED", payload.Error.Code)
	}
}

func TestDetectComposeAcceptsBaseDirectory(t *testing.T) {
	h := &Handler{service: &Service{}}
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/projects/detect-compose",
		strings.NewReader(`{"repoUrl":"","branch":"main","baseDirectory":"docs"}`),
	)
	rec := httptest.NewRecorder()

	h.DetectCompose(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("DetectCompose() status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var payload handlerErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Error.Code == "INVALID_JSON" {
		t.Fatalf("DetectCompose() rejected baseDirectory as invalid JSON: %s", rec.Body.String())
	}
	if payload.Error.Code != "VALIDATION_FAILED" {
		t.Fatalf("DetectCompose() error code = %q, want VALIDATION_FAILED", payload.Error.Code)
	}
}
