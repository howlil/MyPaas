package migration

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"

	"mypaas/internal/httpx"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Prepare starts a migration export.
func (h *Handler) Prepare(w http.ResponseWriter, r *http.Request) {
	m, err := h.service.Prepare(r.Context())
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, m)
}

// Status returns the current migration state.
func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	m := h.service.Status(r.Context(), id)
	if m == nil {
		httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "Migration not found.", nil)
		return
	}
	httpx.JSON(w, http.StatusOK, m)
}

// Download streams the migration archive. Uses a query-string token so the
// browser can download directly without Authorization headers.
func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	token := r.URL.Query().Get("token")

	archivePath, err := h.service.ArchivePath(id, token)
	if err != nil {
		httpx.Error(w, http.StatusForbidden, "DOWNLOAD_FAILED", err.Error(), nil)
		return
	}

	f, err := os.Open(archivePath)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Cannot open archive.", nil)
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Cannot stat archive.", nil)
		return
	}

	filename := filepath.Base(archivePath)
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", fmt.Sprint(info.Size()))
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, f)
}
