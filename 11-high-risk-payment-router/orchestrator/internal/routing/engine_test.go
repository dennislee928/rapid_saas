package routing

import (
	"testing"

	"routekit/orchestrator/internal/model"
)

func TestSelectUKVisaPrefersNuvei(t *testing.T) {
	engine := NewEngine(DefaultRules())
	got := engine.Select(model.ChargeRequest{Country: "GB", Brand: "visa", AmountMinor: 19999})
	if got[0] != "nuvei" {
		t.Fatalf("expected nuvei first, got %v", got)
	}
}

func TestSelectEUCountryPrefersTrust(t *testing.T) {
	engine := NewEngine(DefaultRules())
	got := engine.Select(model.ChargeRequest{Country: "DE", Brand: "visa", AmountMinor: 50000})
	if got[0] != "trust" {
		t.Fatalf("expected trust first, got %v", got)
	}
}
