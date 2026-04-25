package hotstate

import (
	"context"
	"sync"
	"testing"
	"time"
)

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func TestLockCollapsesConcurrentDuplicate(t *testing.T) {
	store := NewMemoryStore(&fakeClock{now: time.Unix(100, 0)})
	ctx := context.Background()
	key := LockKey("routekit", "merchant_1", "charge", "idem_1")

	var winners int
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := store.AcquireLock(ctx, key, "worker", time.Minute)
			if err != nil {
				t.Errorf("AcquireLock: %v", err)
			}
			if ok {
				winners++
			}
		}()
	}
	wg.Wait()
	if winners != 1 {
		t.Fatalf("expected one winner, got %d", winners)
	}
}

func TestIdempotencyReserveReplayAndMismatch(t *testing.T) {
	store := NewMemoryStore(&fakeClock{now: time.Unix(100, 0)})
	ctx := context.Background()

	first, err := store.ReserveIdempotencyKey(ctx, "routekit:merchant_1:charge", "idem_1", "hash_a", time.Hour)
	if err != nil || !first.Reserved {
		t.Fatalf("expected first reservation, got %+v err=%v", first, err)
	}
	replay, err := store.ReserveIdempotencyKey(ctx, "routekit:merchant_1:charge", "idem_1", "hash_a", time.Hour)
	if err != nil || !replay.Replay {
		t.Fatalf("expected replay, got %+v err=%v", replay, err)
	}
	mismatch, err := store.ReserveIdempotencyKey(ctx, "routekit:merchant_1:charge", "idem_1", "hash_b", time.Hour)
	if err != nil || !mismatch.BodyMismatch {
		t.Fatalf("expected body mismatch, got %+v err=%v", mismatch, err)
	}
}

func TestTokenBucketRefillsDeterministically(t *testing.T) {
	clock := &fakeClock{now: time.Unix(100, 0)}
	store := NewMemoryStore(clock)
	ctx := context.Background()
	cfg := TokenBucketConfig{Capacity: 2, RefillTokens: 1, RefillEvery: time.Second}

	for i := 0; i < 2; i++ {
		result, err := store.AllowToken(ctx, "bucket:tenant", cfg, 1)
		if err != nil || !result.Allowed {
			t.Fatalf("expected token %d allowed, got %+v err=%v", i, result, err)
		}
	}
	denied, err := store.AllowToken(ctx, "bucket:tenant", cfg, 1)
	if err != nil || denied.Allowed || denied.RetryAfter != time.Second {
		t.Fatalf("expected one-second denial, got %+v err=%v", denied, err)
	}
	clock.Advance(time.Second)
	allowed, err := store.AllowToken(ctx, "bucket:tenant", cfg, 1)
	if err != nil || !allowed.Allowed || allowed.Remaining != 0 {
		t.Fatalf("expected refill allowance, got %+v err=%v", allowed, err)
	}
}

func TestVelocityCounterWindowExpires(t *testing.T) {
	clock := &fakeClock{now: time.Unix(100, 0)}
	store := NewMemoryStore(clock)
	ctx := context.Background()
	key := VelocityKey("tiltguard", "tenant_1", "fingerprint", "fp_1", "24h")

	count, err := store.IncrVelocity(ctx, key, time.Hour, 2)
	if err != nil || count != 2 {
		t.Fatalf("expected count 2, got %d err=%v", count, err)
	}
	count, err = store.IncrVelocity(ctx, key, time.Hour, 3)
	if err != nil || count != 5 {
		t.Fatalf("expected count 5, got %d err=%v", count, err)
	}
	clock.Advance(time.Hour)
	count, err = store.GetVelocity(ctx, key)
	if err != nil || count != 0 {
		t.Fatalf("expected expired counter, got %d err=%v", count, err)
	}
}

func TestBlacklistAndPSPHealthExpire(t *testing.T) {
	clock := &fakeClock{now: time.Unix(100, 0)}
	store := NewMemoryStore(clock)
	ctx := context.Background()

	if err := store.PutBlacklist(ctx, "ip", "203.0.113.10", "chargeback ring", time.Minute); err != nil {
		t.Fatalf("PutBlacklist: %v", err)
	}
	entry, err := store.CheckBlacklist(ctx, "ip", "203.0.113.10")
	if err != nil || !entry.Listed {
		t.Fatalf("expected listed entry, got %+v err=%v", entry, err)
	}
	if err := store.SetPSPHealth(ctx, "nuvei", PSPHealth{State: PSPStateDown, Reason: "timeouts"}, time.Minute); err != nil {
		t.Fatalf("SetPSPHealth: %v", err)
	}
	health, err := store.GetPSPHealth(ctx, "nuvei")
	if err != nil || health.State != PSPStateDown {
		t.Fatalf("expected down psp, got %+v err=%v", health, err)
	}

	clock.Advance(time.Minute)
	entry, err = store.CheckBlacklist(ctx, "ip", "203.0.113.10")
	if err != nil || entry.Listed {
		t.Fatalf("expected expired blacklist, got %+v err=%v", entry, err)
	}
	health, err = store.GetPSPHealth(ctx, "nuvei")
	if err != nil || health.State != PSPStateUnknown {
		t.Fatalf("expected unknown psp after expiry, got %+v err=%v", health, err)
	}
}
