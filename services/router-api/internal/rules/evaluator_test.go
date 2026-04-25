package rules

import (
	"context"
	"encoding/json"
	"testing"
)

func TestJSONLogicEvaluator(t *testing.T) {
	evaluator := NewJSONLogicEvaluator(NoopListResolver{})
	raw := json.RawMessage(`{"and":[{"==":[{"var":"severity"},"High"]},{"in_cidr":[{"var":"src_ip"},["203.0.113.0/24"]]}]}`)
	data := map[string]any{"severity": "High", "src_ip": "203.0.113.7"}

	matched, err := evaluator.Evaluate(context.Background(), raw, data)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if !matched {
		t.Fatal("expected rule to match")
	}
}
