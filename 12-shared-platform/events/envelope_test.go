package events

import (
	"testing"
	"time"
)

func TestEnvelopeValidationAndTypedPayload(t *testing.T) {
	type webhookPayload struct {
		TransactionID string `json:"transaction_id"`
		Status        string `json:"status"`
	}

	env, err := NewEnvelope(NewEnvelopeInput{
		EventID:        "evt_123",
		TenantID:       "tenant_routekit_demo",
		Type:           "routekit.webhook.delivery_requested",
		SchemaVersion:  1,
		OccurredAt:     time.Date(2026, 4, 26, 8, 0, 0, 0, time.UTC),
		IdempotencyKey: "tenant_routekit_demo:txn_123:webhook",
		Payload:        webhookPayload{TransactionID: "txn_123", Status: "captured"},
		TraceID:        "trace_abc",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := env.Validate(); err != nil {
		t.Fatal(err)
	}

	payload, err := DecodePayload[webhookPayload](env)
	if err != nil {
		t.Fatal(err)
	}
	if payload.TransactionID != "txn_123" || payload.Status != "captured" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestEnvelopeRequiresIdempotencyKey(t *testing.T) {
	_, err := NewEnvelope(NewEnvelopeInput{
		EventID:       "evt_123",
		TenantID:      "tenant_routekit_demo",
		Type:          "routekit.webhook.delivery_requested",
		OccurredAt:    time.Now(),
		SchemaVersion: 1,
		Payload:       map[string]string{"transaction_id": "txn_123"},
	})
	if err == nil {
		t.Fatal("expected missing idempotency key to fail")
	}
}
