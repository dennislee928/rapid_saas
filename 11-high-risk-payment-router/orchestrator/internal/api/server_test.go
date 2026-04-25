package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"routekit/orchestrator/internal/payment"
	"routekit/orchestrator/internal/psp"
	"routekit/orchestrator/internal/routing"
	"routekit/orchestrator/internal/webhook"
)

func TestChargeEndpointRejectsRawPAN(t *testing.T) {
	service := payment.NewService(routing.NewEngine(routing.DefaultRules()), psp.DefaultSandboxAdapters(), webhook.NewMemoryOutbox())
	server := NewServer(service, webhook.NewIngressStore())
	req := httptest.NewRequest(http.MethodPost, "/charges", strings.NewReader(`{
		"merchant_id":"m_test",
		"payment_method_token":"4242424242424242",
		"amount_minor":100,
		"currency":"GBP",
		"country":"GB",
		"brand":"visa"
	}`))
	req.Header.Set("Idempotency-Key", "idem_raw_pan_reject")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d with %s", rec.Code, rec.Body.String())
	}
}

func TestWebhookIngressDedupes(t *testing.T) {
	service := payment.NewService(routing.NewEngine(routing.DefaultRules()), psp.DefaultSandboxAdapters(), webhook.NewMemoryOutbox())
	server := NewServer(service, webhook.NewIngressStore())
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/webhooks/nuvei", strings.NewReader(`{"event":"payment.updated"}`))
		req.Header.Set("X-RouteKit-Sandbox-Event-Id", "evt_1")
		req.Header.Set("X-RouteKit-Sandbox-Signature", "stub")
		rec := httptest.NewRecorder()
		server.Routes().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	}
}
