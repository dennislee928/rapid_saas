package admin

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/rapid-saas/router-api/internal/model"
	"github.com/rapid-saas/router-api/internal/store"
)

type Handler struct {
	repo   store.AdminRepository
	logger *slog.Logger
}

func NewHandler(repo store.AdminRepository, logger *slog.Logger) *Handler {
	return &Handler{repo: repo, logger: logger}
}

func (h *Handler) ListEndpoints(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantID(r.Context())
	items, err := h.repo.ListEndpoints(r.Context(), tenantID)
	writeResult(w, items, err)
}

func (h *Handler) CreateEndpoint(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantID(r.Context())
	var endpoint model.Endpoint
	if !decodeJSON(w, r, &endpoint) {
		return
	}
	endpoint.TenantID = tenantID
	writeResult(w, endpoint, h.repo.CreateEndpoint(r.Context(), &endpoint))
}

func (h *Handler) GetEndpoint(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantID(r.Context())
	item, err := h.repo.GetEndpoint(r.Context(), tenantID, chi.URLParam(r, "endpointID"))
	writeResult(w, item, err)
}

func (h *Handler) UpdateEndpoint(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantID(r.Context())
	var endpoint model.Endpoint
	if !decodeJSON(w, r, &endpoint) {
		return
	}
	endpoint.ID = chi.URLParam(r, "endpointID")
	endpoint.TenantID = tenantID
	writeResult(w, endpoint, h.repo.UpdateEndpoint(r.Context(), &endpoint))
}

func (h *Handler) DeleteEndpoint(w http.ResponseWriter, r *http.Request) {
	err := h.repo.DeleteEndpoint(r.Context(), TenantID(r.Context()), chi.URLParam(r, "endpointID"))
	writeResult(w, map[string]bool{"deleted": err == nil}, err)
}

func (h *Handler) ListDestinations(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ListDestinations(r.Context(), TenantID(r.Context()))
	writeResult(w, items, err)
}

func (h *Handler) CreateDestination(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantID(r.Context())
	var destination model.Destination
	if !decodeJSON(w, r, &destination) {
		return
	}
	destination.TenantID = tenantID
	writeResult(w, destination, h.repo.CreateDestination(r.Context(), &destination))
}

func (h *Handler) GetDestination(w http.ResponseWriter, r *http.Request) {
	item, err := h.repo.GetDestination(r.Context(), TenantID(r.Context()), chi.URLParam(r, "destinationID"))
	writeResult(w, item, err)
}

func (h *Handler) UpdateDestination(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantID(r.Context())
	var destination model.Destination
	if !decodeJSON(w, r, &destination) {
		return
	}
	destination.ID = chi.URLParam(r, "destinationID")
	destination.TenantID = tenantID
	writeResult(w, destination, h.repo.UpdateDestination(r.Context(), &destination))
}

func (h *Handler) DeleteDestination(w http.ResponseWriter, r *http.Request) {
	err := h.repo.DeleteDestination(r.Context(), TenantID(r.Context()), chi.URLParam(r, "destinationID"))
	writeResult(w, map[string]bool{"deleted": err == nil}, err)
}

func (h *Handler) ListRules(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ListRules(r.Context(), TenantID(r.Context()), r.URL.Query().Get("endpoint_id"))
	writeResult(w, items, err)
}

func (h *Handler) CreateRule(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantID(r.Context())
	var rule model.Rule
	if !decodeJSON(w, r, &rule) {
		return
	}
	rule.TenantID = tenantID
	writeResult(w, rule, h.repo.CreateRule(r.Context(), &rule))
}

func (h *Handler) GetRule(w http.ResponseWriter, r *http.Request) {
	item, err := h.repo.GetRule(r.Context(), TenantID(r.Context()), chi.URLParam(r, "ruleID"))
	writeResult(w, item, err)
}

func (h *Handler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantID(r.Context())
	var rule model.Rule
	if !decodeJSON(w, r, &rule) {
		return
	}
	rule.ID = chi.URLParam(r, "ruleID")
	rule.TenantID = tenantID
	writeResult(w, rule, h.repo.UpdateRule(r.Context(), &rule))
}

func (h *Handler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	err := h.repo.DeleteRule(r.Context(), TenantID(r.Context()), chi.URLParam(r, "ruleID"))
	writeResult(w, map[string]bool{"deleted": err == nil}, err)
}

func (h *Handler) ListDeliveryLogs(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ListDeliveryLogs(r.Context(), TenantID(r.Context()), intQuery(r, "limit", 50))
	writeResult(w, items, err)
}

func (h *Handler) ListDLQ(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ListDLQ(r.Context(), TenantID(r.Context()), intQuery(r, "limit", 50))
	writeResult(w, items, err)
}

func (h *Handler) ReplayDLQ(w http.ResponseWriter, r *http.Request) {
	event, err := h.repo.ReplayDLQ(r.Context(), TenantID(r.Context()), chi.URLParam(r, "dlqID"))
	writeResult(w, map[string]any{"replayed": err == nil, "event": event}, err)
}

func (h *Handler) UsageSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.repo.UsageSummary(r.Context(), TenantID(r.Context()))
	writeResult(w, summary, err)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

func intQuery(r *http.Request, key string, fallback int) int {
	value, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func writeResult(w http.ResponseWriter, payload any, err error) {
	if err == nil {
		writeJSON(w, http.StatusOK, payload)
		return
	}
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, store.ErrConflict) {
		writeError(w, http.StatusConflict, "conflict")
		return
	}
	writeError(w, http.StatusInternalServerError, "internal error")
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
