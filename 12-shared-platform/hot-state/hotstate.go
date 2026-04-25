package hotstate

import (
	"context"
	"errors"
	"time"
)

var (
	ErrLockHeld          = errors.New("hotstate: lock is already held")
	ErrIdempotencyReplay = errors.New("hotstate: idempotency key already exists")
	ErrTokenBucketDenied = errors.New("hotstate: token bucket denied")
	ErrInvalidTTL        = errors.New("hotstate: ttl must be positive")
)

type Clock interface {
	Now() time.Time
}

type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now().UTC() }

type LockStore interface {
	AcquireLock(ctx context.Context, key, owner string, ttl time.Duration) (bool, error)
	ReleaseLock(ctx context.Context, key, owner string) (bool, error)
}

type IdempotencyStore interface {
	ReserveIdempotencyKey(ctx context.Context, scope, key, bodyHash string, ttl time.Duration) (IdempotencyResult, error)
}

type IdempotencyResult struct {
	Reserved     bool
	Replay       bool
	BodyMismatch bool
	BodyHash     string
}

type TokenBucketStore interface {
	AllowToken(ctx context.Context, key string, cfg TokenBucketConfig, cost int64) (TokenBucketResult, error)
}

type TokenBucketConfig struct {
	Capacity     int64
	RefillTokens int64
	RefillEvery  time.Duration
}

type TokenBucketResult struct {
	Allowed    bool
	Remaining  int64
	RetryAfter time.Duration
}

type VelocityStore interface {
	IncrVelocity(ctx context.Context, key string, window time.Duration, delta int64) (int64, error)
	GetVelocity(ctx context.Context, key string) (int64, error)
}

type BlacklistStore interface {
	PutBlacklist(ctx context.Context, kind, value, reason string, ttl time.Duration) error
	CheckBlacklist(ctx context.Context, kind, value string) (BlacklistEntry, error)
}

type BlacklistEntry struct {
	Listed bool
	Kind   string
	Value  string
	Reason string
}

type PSPHealthStore interface {
	SetPSPHealth(ctx context.Context, psp string, health PSPHealth, ttl time.Duration) error
	GetPSPHealth(ctx context.Context, psp string) (PSPHealth, error)
}

type PSPState string

const (
	PSPStateUnknown  PSPState = "unknown"
	PSPStateHealthy  PSPState = "healthy"
	PSPStateDegraded PSPState = "degraded"
	PSPStateDown     PSPState = "down"
)

type PSPHealth struct {
	PSP          string
	State        PSPState
	LatencyMS    int64
	ErrorRatePPM int64
	CheckedAt    time.Time
	Reason       string
}

type Store interface {
	LockStore
	IdempotencyStore
	TokenBucketStore
	VelocityStore
	BlacklistStore
	PSPHealthStore
}

func LockKey(product, tenant, operation, entity string) string {
	return joinKey("lock", product, tenant, operation, entity)
}

func IdempotencyKey(product, tenant, operation, key string) string {
	return joinKey("idem", product, tenant, operation, key)
}

func VelocityKey(product, tenant, entityKind, entityID, window string) string {
	return joinKey("vel", product, tenant, entityKind, entityID, window)
}

func BlacklistKey(kind, value string) string {
	return joinKey("blacklist", kind, value)
}

func PSPHealthKey(psp string) string {
	return joinKey("psp", "health", psp)
}

func joinKey(parts ...string) string {
	out := ""
	for i, part := range parts {
		if i > 0 {
			out += ":"
		}
		out += part
	}
	return out
}
