package store

import "testing"

func TestMemoryStoreReturnsCopy(t *testing.T) {
	repo := NewMemoryStore()
	repo.AddAsset(Asset{ID: "asset_1"})
	assets := repo.Assets()
	assets[0].ID = "mutated"
	if repo.Assets()[0].ID != "asset_1" {
		t.Fatal("expected store assets to be protected from caller mutation")
	}
}
