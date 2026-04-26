package store

import (
	"context"
	"testing"

	"github.com/rapid-saas/router-api/internal/model"
)

func TestMemoryStoreListRulesOrdersByPosition(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	rules := []model.Rule{
		{ID: "rule_30", TenantID: "tenant_1", EndpointID: "ep_1", Position: 30},
		{ID: "rule_10", TenantID: "tenant_1", EndpointID: "ep_1", Position: 10},
		{ID: "rule_20", TenantID: "tenant_1", EndpointID: "ep_1", Position: 20},
	}
	for i := range rules {
		if err := store.CreateRule(ctx, &rules[i]); err != nil {
			t.Fatalf("CreateRule() error = %v", err)
		}
	}

	got, err := store.ListRules(ctx, "tenant_1", "ep_1")
	if err != nil {
		t.Fatalf("ListRules() error = %v", err)
	}
	if got[0].ID != "rule_10" || got[1].ID != "rule_20" || got[2].ID != "rule_30" {
		t.Fatalf("rules not ordered by position: %#v", got)
	}
}

func TestMemoryStoreDLQReplayRemovesEntry(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	if err := store.ParkDLQ(ctx, model.DLQEntry{
		ID:         "dlq_1",
		TenantID:   "tenant_1",
		EndpointID: "ep_1",
		PayloadB64: "eyJvayI6dHJ1ZX0=",
		Attempts:   1,
	}); err != nil {
		t.Fatalf("ParkDLQ() error = %v", err)
	}
	event, err := store.ReplayDLQ(ctx, "tenant_1", "dlq_1")
	if err != nil {
		t.Fatalf("ReplayDLQ() error = %v", err)
	}
	if event.EndpointID != "ep_1" || string(event.Body) != `{"ok":true}` {
		t.Fatalf("unexpected replay event: %#v", event)
	}
	items, err := store.ListDLQ(ctx, "tenant_1", 10)
	if err != nil {
		t.Fatalf("ListDLQ() error = %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected replay to remove entry, got %#v", items)
	}
}

func TestMemoryStoreUsageSummary(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	_ = store.IncrementUsage(ctx, "tenant_1", "delivered")
	_ = store.IncrementUsage(ctx, "tenant_1", "failed")
	usage, err := store.UsageSummary(ctx, "tenant_1")
	if err != nil {
		t.Fatalf("UsageSummary() error = %v", err)
	}
	if usage.Ingressed != 2 || usage.Forwarded != 1 || usage.Failed != 1 {
		t.Fatalf("unexpected usage: %#v", usage)
	}
}
