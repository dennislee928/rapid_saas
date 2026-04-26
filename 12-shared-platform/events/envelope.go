package events

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type Envelope struct {
	EventID        string          `json:"event_id"`
	TenantID       string          `json:"tenant_id"`
	Type           string          `json:"type"`
	SchemaVersion  int             `json:"schema_version"`
	OccurredAt     time.Time       `json:"occurred_at"`
	IdempotencyKey string          `json:"idempotency_key"`
	Payload        json.RawMessage `json:"payload"`
	TraceID        string          `json:"trace_id,omitempty"`
}

type NewEnvelopeInput struct {
	EventID        string
	TenantID       string
	Type           string
	SchemaVersion  int
	OccurredAt     time.Time
	IdempotencyKey string
	Payload        any
	TraceID        string
}

func NewEnvelope(input NewEnvelopeInput) (Envelope, error) {
	payload, err := json.Marshal(input.Payload)
	if err != nil {
		return Envelope{}, err
	}
	if input.OccurredAt.IsZero() {
		input.OccurredAt = time.Now().UTC()
	}
	if input.SchemaVersion == 0 {
		input.SchemaVersion = 1
	}
	env := Envelope{
		EventID:        input.EventID,
		TenantID:       input.TenantID,
		Type:           input.Type,
		SchemaVersion:  input.SchemaVersion,
		OccurredAt:     input.OccurredAt.UTC(),
		IdempotencyKey: input.IdempotencyKey,
		Payload:        payload,
		TraceID:        input.TraceID,
	}
	if err := env.Validate(); err != nil {
		return Envelope{}, err
	}
	return env, nil
}

func (e Envelope) Validate() error {
	switch {
	case strings.TrimSpace(e.EventID) == "":
		return errors.New("event_id is required")
	case strings.TrimSpace(e.TenantID) == "":
		return errors.New("tenant_id is required")
	case strings.TrimSpace(e.Type) == "":
		return errors.New("type is required")
	case e.SchemaVersion < 1:
		return errors.New("schema_version must be positive")
	case e.OccurredAt.IsZero():
		return errors.New("occurred_at is required")
	case strings.TrimSpace(e.IdempotencyKey) == "":
		return errors.New("idempotency_key is required")
	case len(e.Payload) == 0 || !json.Valid(e.Payload):
		return errors.New("payload must be valid json")
	default:
		return nil
	}
}

func DecodePayload[T any](e Envelope) (T, error) {
	var payload T
	err := json.Unmarshal(e.Payload, &payload)
	return payload, err
}
