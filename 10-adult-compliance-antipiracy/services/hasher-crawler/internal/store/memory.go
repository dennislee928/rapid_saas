package store

import "time"

type Asset struct {
	ID         string
	TenantID   string
	Kind       string
	Label      string
	PHash      uint64
	Uploaded   time.Time
	SourceGone bool
}

type MemoryStore struct {
	assets []Asset
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

func (m *MemoryStore) AddAsset(asset Asset) {
	m.assets = append(m.assets, asset)
}

func (m *MemoryStore) Assets() []Asset {
	out := make([]Asset, len(m.assets))
	copy(out, m.assets)
	return out
}
