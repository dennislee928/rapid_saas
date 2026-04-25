package queue

import (
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/rapid-saas/router-api/internal/model"
)

type Handler struct {
	processor    *Processor
	logger       *slog.Logger
	sharedSecret string
}

type consumeRequest struct {
	Events []model.QueueEvent `json:"events"`
}

func NewHandler(processor *Processor, logger *slog.Logger, sharedSecret string) *Handler {
	return &Handler{processor: processor, logger: logger, sharedSecret: sharedSecret}
}

func (h *Handler) Consume(w http.ResponseWriter, r *http.Request) {
	if h.sharedSecret != "" && !validBearer(r.Header.Get("Authorization"), h.sharedSecret) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	defer r.Body.Close()
	var req consumeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	results := h.processor.ProcessBatch(r.Context(), req.Events)
	writeJSON(w, http.StatusAccepted, map[string]any{"accepted": len(req.Events), "results": results})
}

func validBearer(header, secret string) bool {
	token := strings.TrimPrefix(header, "Bearer ")
	return subtle.ConstantTimeCompare([]byte(token), []byte(secret)) == 1
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
