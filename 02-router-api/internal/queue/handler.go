package queue

import (
	"encoding/base64"
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/rapid-saas/router-api/internal/model"
)

type Handler struct {
	processor    *Processor
	logger       *slog.Logger
	sharedSecret string
}

type consumeRequest struct {
	Events []json.RawMessage `json:"events"`
}

type sharedEnvelope struct {
	EventID       string          `json:"event_id"`
	TenantID      string          `json:"tenant_id"`
	Type          string          `json:"type"`
	SchemaVersion int             `json:"schema_version"`
	OccurredAt    time.Time       `json:"occurred_at"`
	Payload       json.RawMessage `json:"payload"`
}

type webhookReceivedPayload struct {
	EndpointID  string            `json:"endpointId"`
	TenantID    string            `json:"tenantId"`
	Headers     map[string]string `json:"headers"`
	BodyBase64  string            `json:"bodyBase64"`
	BodySha256  string            `json:"bodySha256"`
	ReceivedAt  string            `json:"receivedAt"`
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
	events := make([]model.QueueEvent, 0, len(req.Events))
	for _, raw := range req.Events {
		event, err := decodeQueueEvent(raw)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		events = append(events, event)
	}
	results := h.processor.ProcessBatch(r.Context(), events)
	writeJSON(w, http.StatusAccepted, map[string]any{"accepted": len(events), "results": results})
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

func decodeQueueEvent(raw json.RawMessage) (model.QueueEvent, error) {
	var envelope sharedEnvelope
	if err := json.Unmarshal(raw, &envelope); err == nil && envelope.Type == "security.webhook.received" {
		return decodeWebhookEnvelope(envelope)
	}

	var legacy model.QueueEvent
	if err := json.Unmarshal(raw, &legacy); err != nil {
		return model.QueueEvent{}, err
	}
	return legacy, nil
}

func decodeWebhookEnvelope(envelope sharedEnvelope) (model.QueueEvent, error) {
	var payload webhookReceivedPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return model.QueueEvent{}, err
	}

	body, err := base64.StdEncoding.DecodeString(payload.BodyBase64)
	if err != nil {
		return model.QueueEvent{}, err
	}
	bodyJSON := json.RawMessage(body)
	if !json.Valid(bodyJSON) {
		encoded, err := json.Marshal(string(body))
		if err != nil {
			return model.QueueEvent{}, err
		}
		bodyJSON = encoded
	}

	receivedAt := envelope.OccurredAt.UnixMilli()
	if payload.ReceivedAt != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, payload.ReceivedAt); err == nil {
			receivedAt = parsed.UnixMilli()
		}
	}

	tenantID := payload.TenantID
	if tenantID == "" {
		tenantID = envelope.TenantID
	}

	return model.QueueEvent{
		ID:          envelope.EventID,
		TenantID:    tenantID,
		EndpointID:  payload.EndpointID,
		Headers:     payload.Headers,
		Body:        bodyJSON,
		ReceivedAt:  receivedAt,
		RequestHash: payload.BodySha256,
	}, nil
}
