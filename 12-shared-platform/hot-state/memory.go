package hotstate

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

type MemoryStore struct {
	mu    sync.Mutex
	clock Clock
	items map[string]memoryItem
}

type memoryItem struct {
	value     string
	expiresAt time.Time
}

func NewMemoryStore(clock Clock) *MemoryStore {
	if clock == nil {
		clock = SystemClock{}
	}
	return &MemoryStore{clock: clock, items: map[string]memoryItem{}}
}

func (s *MemoryStore) AcquireLock(ctx context.Context, key, owner string, ttl time.Duration) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if ttl <= 0 {
		return false, ErrInvalidTTL
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expireLocked(key)
	if _, ok := s.items[key]; ok {
		return false, nil
	}
	s.items[key] = memoryItem{value: owner, expiresAt: s.clock.Now().Add(ttl)}
	return true, nil
}

func (s *MemoryStore) ReleaseLock(ctx context.Context, key, owner string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expireLocked(key)
	item, ok := s.items[key]
	if !ok || item.value != owner {
		return false, nil
	}
	delete(s.items, key)
	return true, nil
}

func (s *MemoryStore) ReserveIdempotencyKey(ctx context.Context, scope, key, bodyHash string, ttl time.Duration) (IdempotencyResult, error) {
	if err := ctx.Err(); err != nil {
		return IdempotencyResult{}, err
	}
	if ttl <= 0 {
		return IdempotencyResult{}, ErrInvalidTTL
	}
	cacheKey := IdempotencyKey(scope, "_", "_", key)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expireLocked(cacheKey)
	if item, ok := s.items[cacheKey]; ok {
		return IdempotencyResult{
			Reserved:     false,
			Replay:       item.value == bodyHash,
			BodyMismatch: item.value != bodyHash,
			BodyHash:     item.value,
		}, nil
	}
	s.items[cacheKey] = memoryItem{value: bodyHash, expiresAt: s.clock.Now().Add(ttl)}
	return IdempotencyResult{Reserved: true, BodyHash: bodyHash}, nil
}

func (s *MemoryStore) AllowToken(ctx context.Context, key string, cfg TokenBucketConfig, cost int64) (TokenBucketResult, error) {
	if err := ctx.Err(); err != nil {
		return TokenBucketResult{}, err
	}
	if cfg.Capacity <= 0 || cfg.RefillTokens <= 0 || cfg.RefillEvery <= 0 || cost <= 0 {
		return TokenBucketResult{}, ErrTokenBucketDenied
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.clock.Now()
	state := tokenBucketState{Tokens: cfg.Capacity, UpdatedUnixMS: now.UnixMilli()}
	if item, ok := s.items[key]; ok {
		_ = json.Unmarshal([]byte(item.value), &state)
	}
	elapsed := now.Sub(time.UnixMilli(state.UpdatedUnixMS))
	if elapsed > 0 {
		refills := int64(elapsed / cfg.RefillEvery)
		if refills > 0 {
			state.Tokens = minInt64(cfg.Capacity, state.Tokens+refills*cfg.RefillTokens)
			state.UpdatedUnixMS = now.UnixMilli()
		}
	}
	if state.Tokens < cost {
		deficit := cost - state.Tokens
		periods := (deficit + cfg.RefillTokens - 1) / cfg.RefillTokens
		return TokenBucketResult{Allowed: false, Remaining: state.Tokens, RetryAfter: time.Duration(periods) * cfg.RefillEvery}, nil
	}
	state.Tokens -= cost
	encoded, _ := json.Marshal(state)
	s.items[key] = memoryItem{value: string(encoded), expiresAt: now.Add(cfg.RefillEvery * 2)}
	return TokenBucketResult{Allowed: true, Remaining: state.Tokens}, nil
}

func (s *MemoryStore) IncrVelocity(ctx context.Context, key string, window time.Duration, delta int64) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if window <= 0 {
		return 0, ErrInvalidTTL
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expireLocked(key)
	now := s.clock.Now()
	current := int64(0)
	if item, ok := s.items[key]; ok {
		_ = json.Unmarshal([]byte(item.value), &current)
	}
	current += delta
	encoded, _ := json.Marshal(current)
	s.items[key] = memoryItem{value: string(encoded), expiresAt: now.Add(window)}
	return current, nil
}

func (s *MemoryStore) GetVelocity(ctx context.Context, key string) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expireLocked(key)
	if item, ok := s.items[key]; ok {
		var current int64
		_ = json.Unmarshal([]byte(item.value), &current)
		return current, nil
	}
	return 0, nil
}

func (s *MemoryStore) PutBlacklist(ctx context.Context, kind, value, reason string, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if ttl <= 0 {
		return ErrInvalidTTL
	}
	entry := BlacklistEntry{Listed: true, Kind: kind, Value: value, Reason: reason}
	encoded, _ := json.Marshal(entry)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[BlacklistKey(kind, value)] = memoryItem{value: string(encoded), expiresAt: s.clock.Now().Add(ttl)}
	return nil
}

func (s *MemoryStore) CheckBlacklist(ctx context.Context, kind, value string) (BlacklistEntry, error) {
	if err := ctx.Err(); err != nil {
		return BlacklistEntry{}, err
	}
	key := BlacklistKey(kind, value)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expireLocked(key)
	if item, ok := s.items[key]; ok {
		var entry BlacklistEntry
		_ = json.Unmarshal([]byte(item.value), &entry)
		return entry, nil
	}
	return BlacklistEntry{Kind: kind, Value: value}, nil
}

func (s *MemoryStore) SetPSPHealth(ctx context.Context, psp string, health PSPHealth, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if ttl <= 0 {
		return ErrInvalidTTL
	}
	health.PSP = psp
	if health.State == "" {
		health.State = PSPStateUnknown
	}
	if health.CheckedAt.IsZero() {
		health.CheckedAt = s.clock.Now()
	}
	encoded, _ := json.Marshal(health)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[PSPHealthKey(psp)] = memoryItem{value: string(encoded), expiresAt: s.clock.Now().Add(ttl)}
	return nil
}

func (s *MemoryStore) GetPSPHealth(ctx context.Context, psp string) (PSPHealth, error) {
	if err := ctx.Err(); err != nil {
		return PSPHealth{}, err
	}
	key := PSPHealthKey(psp)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expireLocked(key)
	if item, ok := s.items[key]; ok {
		var health PSPHealth
		_ = json.Unmarshal([]byte(item.value), &health)
		return health, nil
	}
	return PSPHealth{PSP: psp, State: PSPStateUnknown}, nil
}

func (s *MemoryStore) expireLocked(key string) {
	item, ok := s.items[key]
	if ok && !item.expiresAt.IsZero() && !s.clock.Now().Before(item.expiresAt) {
		delete(s.items, key)
	}
}

type tokenBucketState struct {
	Tokens        int64 `json:"tokens"`
	UpdatedUnixMS int64 `json:"updated_unix_ms"`
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
