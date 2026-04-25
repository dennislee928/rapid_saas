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
