package payment

import (
	"context"
	"testing"

	"routekit/orchestrator/internal/model"
	"routekit/orchestrator/internal/psp"
	"routekit/orchestrator/internal/routing"
	"routekit/orchestrator/internal/webhook"
)

func TestChargeCascadesOnRetriableDecline(t *testing.T) {
	service := NewService(routing.NewEngine(routing.DefaultRules()), psp.DefaultSandboxAdapters(), webhook.NewMemoryOutbox())
	txn, err := service.Charge(context.Background(), model.ChargeRequest{
		MerchantID:         "m_test",
		IdempotencyKey:     "idem_1",
		PaymentMethodToken: "btok_decline_do_not_honor",
		AmountMinor:        1200,
		Currency:           "GBP",
		Country:            "GB",
		Brand:              "visa",
	})
	if err != nil {
		t.Fatal(err)
	}
	if txn.AttemptCount != 3 {
		t.Fatalf("expected all fallback PSPs to be attempted, got %d", txn.AttemptCount)
	}
	if txn.State != model.StateFailedTerminal {
		t.Fatalf("expected terminal failure after exhausting PSPs, got %s", txn.State)
	}
}

func TestChargeDoesNotCascadeInsufficientFunds(t *testing.T) {
	service := NewService(routing.NewEngine(routing.DefaultRules()), psp.DefaultSandboxAdapters(), webhook.NewMemoryOutbox())
	txn, err := service.Charge(context.Background(), model.ChargeRequest{
		MerchantID:         "m_test",
		IdempotencyKey:     "idem_2",
		PaymentMethodToken: "btok_decline_insufficient",
		AmountMinor:        1200,
		Currency:           "GBP",
		Country:            "GB",
		Brand:              "visa",
	})
	if err != nil {
		t.Fatal(err)
	}
	if txn.AttemptCount != 1 {
		t.Fatalf("insufficient funds must not cascade, got %d attempts", txn.AttemptCount)
	}
}

func TestRejectsRawPANShapedToken(t *testing.T) {
	err := ValidateTokenOnlyCharge(model.ChargeRequest{
		MerchantID:         "m_test",
		IdempotencyKey:     "idem_3",
		PaymentMethodToken: "4242 4242 4242 4242",
		AmountMinor:        100,
		Currency:           "GBP",
	})
	if err == nil {
		t.Fatal("expected raw PAN shaped value to be rejected")
	}
}

func TestIdempotencyReturnsExistingTransaction(t *testing.T) {
	service := NewService(routing.NewEngine(routing.DefaultRules()), psp.DefaultSandboxAdapters(), webhook.NewMemoryOutbox())
	req := model.ChargeRequest{
		MerchantID:         "m_test",
		IdempotencyKey:     "idem_4",
		PaymentMethodToken: "btok_demo",
		AmountMinor:        100,
		Currency:           "GBP",
		Country:            "GB",
		Brand:              "visa",
	}
	first, err := service.Charge(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Charge(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("expected same transaction for idempotency key, got %s and %s", first.ID, second.ID)
	}
}
