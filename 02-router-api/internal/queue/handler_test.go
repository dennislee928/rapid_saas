package queue

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestDecodeQueueEventAcceptsSharedEnvelope(t *testing.T) {
	payload := map[string]any{
		"endpointId": "ep_1",
		"tenantId":   "tenant_1",
		"headers": map[string]string{
			"x-test": "yes",
		},
		"bodyBase64": base64.StdEncoding.EncodeToString([]byte(`{"alert":true}`)),
		"bodySha256": "abc123",
		"receivedAt":  "2026-04-26T00:00:00Z",
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(map[string]any{
		"event_id":       "evt_1",
		"tenant_id":      "tenant_1",
		"type":           "security.webhook.received",
		"schema_version": 1,
		"occurred_at":    "2026-04-26T00:00:00Z",
		"payload":        json.RawMessage(payloadBytes),
	})
	if err != nil {
		t.Fatal(err)
	}

	event, err := decodeQueueEvent(raw)
	if err != nil {
		t.Fatalf("decodeQueueEvent() error = %v", err)
	}
	if event.ID != "evt_1" || event.TenantID != "tenant_1" || event.EndpointID != "ep_1" {
		t.Fatalf("decoded wrong event identity: %#v", event)
	}
	if string(event.Body) != `{"alert":true}` {
		t.Fatalf("decoded wrong body: %s", event.Body)
	}
	if event.RequestHash != "abc123" {
		t.Fatalf("decoded wrong request hash: %s", event.RequestHash)
	}
}

func TestDecodeQueueEventKeepsLegacyPayload(t *testing.T) {
	raw := []byte(`{"id":"evt_legacy","tenant_id":"tenant_1","endpoint_id":"ep_1","headers":{},"body":{"ok":true},"received_at_ms":1770000000000,"request_hash":"hash_1"}`)
	event, err := decodeQueueEvent(raw)
	if err != nil {
		t.Fatalf("decodeQueueEvent() error = %v", err)
	}
	if event.ID != "evt_legacy" || string(event.Body) != `{"ok":true}` {
		t.Fatalf("decoded wrong legacy event: %#v", event)
	}
}
