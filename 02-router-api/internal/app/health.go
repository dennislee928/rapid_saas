package app

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/rapid-saas/router-api/internal/store"
)

type HealthHandler struct {
	logger     *slog.Logger
	repository store.HealthRepository
}

func NewHealthHandler(logger *slog.Logger, repository store.HealthRepository) *HealthHandler {
	return &HealthHandler{logger: logger, repository: repository}
}

func (h *HealthHandler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if h.repository != nil {
		if err := h.repository.Ping(r.Context()); err != nil {
			h.logger.Warn("readiness check failed", slog.Any("error", err))
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
