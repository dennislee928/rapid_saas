package hotstate

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

type RedisStore struct {
	addr      string
	password  string
	db        int
	tlsConfig *tls.Config
	timeout   time.Duration
	clock     Clock
}

type RedisOption func(*RedisStore)

func WithRedisPassword(password string) RedisOption {
	return func(s *RedisStore) { s.password = password }
}

func WithRedisDB(db int) RedisOption {
	return func(s *RedisStore) { s.db = db }
}

func WithRedisTLS(cfg *tls.Config) RedisOption {
	return func(s *RedisStore) { s.tlsConfig = cfg }
}

func WithRedisTimeout(timeout time.Duration) RedisOption {
	return func(s *RedisStore) { s.timeout = timeout }
}

func WithClock(clock Clock) RedisOption {
	return func(s *RedisStore) { s.clock = clock }
}

func NewRedisStore(addr string, opts ...RedisOption) *RedisStore {
	s := &RedisStore{addr: addr, timeout: 2 * time.Second, clock: SystemClock{}}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *RedisStore) AcquireLock(ctx context.Context, key, owner string, ttl time.Duration) (bool, error) {
	if ttl <= 0 {
		return false, ErrInvalidTTL
	}
	reply, err := s.do(ctx, "SET", key, owner, "PX", strconv.FormatInt(ttl.Milliseconds(), 10), "NX")
	if err != nil {
		return false, err
	}
	return strings.EqualFold(reply.simpleString(), "OK"), nil
}

func (s *RedisStore) ReleaseLock(ctx context.Context, key, owner string) (bool, error) {
	script := `if redis.call("GET", KEYS[1]) == ARGV[1] then return redis.call("DEL", KEYS[1]) else return 0 end`
	reply, err := s.do(ctx, "EVAL", script, "1", key, owner)
	if err != nil {
		return false, err
	}
	return reply.integer == 1, nil
}

func (s *RedisStore) ReserveIdempotencyKey(ctx context.Context, scope, key, bodyHash string, ttl time.Duration) (IdempotencyResult, error) {
	if ttl <= 0 {
		return IdempotencyResult{}, ErrInvalidTTL
	}
	cacheKey := IdempotencyKey(scope, "_", "_", key)
	reply, err := s.do(ctx, "SET", cacheKey, bodyHash, "PX", strconv.FormatInt(ttl.Milliseconds(), 10), "NX")
	if err != nil {
		return IdempotencyResult{}, err
	}
	if strings.EqualFold(reply.simpleString(), "OK") {
		return IdempotencyResult{Reserved: true, BodyHash: bodyHash}, nil
	}
	existing, err := s.do(ctx, "GET", cacheKey)
	if err != nil {
		return IdempotencyResult{}, err
	}
	existingHash := existing.bulk
	return IdempotencyResult{Replay: existingHash == bodyHash, BodyMismatch: existingHash != bodyHash, BodyHash: existingHash}, nil
}

func (s *RedisStore) AllowToken(ctx context.Context, key string, cfg TokenBucketConfig, cost int64) (TokenBucketResult, error) {
	if cfg.Capacity <= 0 || cfg.RefillTokens <= 0 || cfg.RefillEvery <= 0 || cost <= 0 {
		return TokenBucketResult{}, ErrTokenBucketDenied
	}
	script := `
local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local refill_tokens = tonumber(ARGV[2])
local refill_every_ms = tonumber(ARGV[3])
local cost = tonumber(ARGV[4])
local now_ms = tonumber(ARGV[5])
local raw = redis.call("GET", key)
local tokens = capacity
local updated_ms = now_ms
if raw then
  local sep = string.find(raw, ":")
  tokens = tonumber(string.sub(raw, 1, sep - 1))
  updated_ms = tonumber(string.sub(raw, sep + 1))
end
local elapsed = now_ms - updated_ms
if elapsed >= refill_every_ms then
  local periods = math.floor(elapsed / refill_every_ms)
  tokens = math.min(capacity, tokens + periods * refill_tokens)
  updated_ms = now_ms
end
if tokens < cost then
  local deficit = cost - tokens
  local periods = math.ceil(deficit / refill_tokens)
  return {0, tokens, periods * refill_every_ms}
end
tokens = tokens - cost
redis.call("SET", key, tostring(tokens) .. ":" .. tostring(updated_ms), "PX", refill_every_ms * 2)
return {1, tokens, 0}`
	reply, err := s.do(ctx, "EVAL", script, "1", key,
		strconv.FormatInt(cfg.Capacity, 10),
		strconv.FormatInt(cfg.RefillTokens, 10),
		strconv.FormatInt(cfg.RefillEvery.Milliseconds(), 10),
		strconv.FormatInt(cost, 10),
		strconv.FormatInt(s.clock.Now().UnixMilli(), 10),
	)
	if err != nil {
		return TokenBucketResult{}, err
	}
	values := reply.arrayInts()
	if len(values) != 3 {
		return TokenBucketResult{}, fmt.Errorf("hotstate: unexpected token bucket reply")
	}
	return TokenBucketResult{Allowed: values[0] == 1, Remaining: values[1], RetryAfter: time.Duration(values[2]) * time.Millisecond}, nil
}

func (s *RedisStore) IncrVelocity(ctx context.Context, key string, window time.Duration, delta int64) (int64, error) {
	if window <= 0 {
		return 0, ErrInvalidTTL
	}
	script := `local n = redis.call("INCRBY", KEYS[1], ARGV[1]); if n == tonumber(ARGV[1]) then redis.call("PEXPIRE", KEYS[1], ARGV[2]) end; return n`
	reply, err := s.do(ctx, "EVAL", script, "1", key, strconv.FormatInt(delta, 10), strconv.FormatInt(window.Milliseconds(), 10))
	if err != nil {
		return 0, err
	}
	return reply.integer, nil
}

func (s *RedisStore) GetVelocity(ctx context.Context, key string) (int64, error) {
	reply, err := s.do(ctx, "GET", key)
	if err != nil {
		return 0, err
	}
	if reply.nil {
		return 0, nil
	}
	return strconv.ParseInt(reply.bulk, 10, 64)
}

func (s *RedisStore) PutBlacklist(ctx context.Context, kind, value, reason string, ttl time.Duration) error {
	if ttl <= 0 {
		return ErrInvalidTTL
	}
	entry := BlacklistEntry{Listed: true, Kind: kind, Value: value, Reason: reason}
	encoded, _ := json.Marshal(entry)
	_, err := s.do(ctx, "SET", BlacklistKey(kind, value), string(encoded), "PX", strconv.FormatInt(ttl.Milliseconds(), 10))
	return err
}

func (s *RedisStore) CheckBlacklist(ctx context.Context, kind, value string) (BlacklistEntry, error) {
	reply, err := s.do(ctx, "GET", BlacklistKey(kind, value))
	if err != nil {
		return BlacklistEntry{}, err
	}
	if reply.nil {
		return BlacklistEntry{Kind: kind, Value: value}, nil
	}
	var entry BlacklistEntry
	if err := json.Unmarshal([]byte(reply.bulk), &entry); err != nil {
		return BlacklistEntry{}, err
	}
	return entry, nil
}

func (s *RedisStore) SetPSPHealth(ctx context.Context, psp string, health PSPHealth, ttl time.Duration) error {
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
	_, err := s.do(ctx, "SET", PSPHealthKey(psp), string(encoded), "PX", strconv.FormatInt(ttl.Milliseconds(), 10))
	return err
}

func (s *RedisStore) GetPSPHealth(ctx context.Context, psp string) (PSPHealth, error) {
	reply, err := s.do(ctx, "GET", PSPHealthKey(psp))
	if err != nil {
		return PSPHealth{}, err
	}
	if reply.nil {
		return PSPHealth{PSP: psp, State: PSPStateUnknown}, nil
	}
	var health PSPHealth
	if err := json.Unmarshal([]byte(reply.bulk), &health); err != nil {
		return PSPHealth{}, err
	}
	return health, nil
}

func (s *RedisStore) do(ctx context.Context, args ...string) (redisReply, error) {
	if err := ctx.Err(); err != nil {
		return redisReply{}, err
	}
	conn, err := s.dial(ctx)
	if err != nil {
		return redisReply{}, err
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else if s.timeout > 0 {
		_ = conn.SetDeadline(time.Now().Add(s.timeout))
	}
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	if s.password != "" {
		if err := writeRESP(rw, "AUTH", s.password); err != nil {
			return redisReply{}, err
		}
		if _, err := readReply(rw.Reader); err != nil {
			return redisReply{}, err
		}
	}
	if s.db > 0 {
		if err := writeRESP(rw, "SELECT", strconv.Itoa(s.db)); err != nil {
			return redisReply{}, err
		}
		if _, err := readReply(rw.Reader); err != nil {
			return redisReply{}, err
		}
	}
	if err := writeRESP(rw, args...); err != nil {
		return redisReply{}, err
	}
	return readReply(rw.Reader)
}

func (s *RedisStore) dial(ctx context.Context) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: s.timeout}
	if s.tlsConfig != nil {
		return tls.DialWithDialer(dialer, "tcp", s.addr, s.tlsConfig)
	}
	return dialer.DialContext(ctx, "tcp", s.addr)
}

func writeRESP(rw *bufio.ReadWriter, args ...string) error {
	if _, err := fmt.Fprintf(rw, "*%d\r\n", len(args)); err != nil {
		return err
	}
	for _, arg := range args {
		if _, err := fmt.Fprintf(rw, "$%d\r\n%s\r\n", len(arg), arg); err != nil {
			return err
		}
	}
	return rw.Flush()
}

type redisReply struct {
	kind    byte
	str     string
	bulk    string
	integer int64
	array   []redisReply
	nil     bool
}

func (r redisReply) simpleString() string {
	if r.kind == '+' {
		return r.str
	}
	return ""
}

func (r redisReply) arrayInts() []int64 {
	out := make([]int64, 0, len(r.array))
	for _, item := range r.array {
		out = append(out, item.integer)
	}
	return out
}

func readReply(r *bufio.Reader) (redisReply, error) {
	prefix, err := r.ReadByte()
	if err != nil {
		return redisReply{}, err
	}
	switch prefix {
	case '+':
		line, err := readLine(r)
		return redisReply{kind: '+', str: line}, err
	case '-':
		line, _ := readLine(r)
		return redisReply{}, errors.New(line)
	case ':':
		line, err := readLine(r)
		if err != nil {
			return redisReply{}, err
		}
		n, err := strconv.ParseInt(line, 10, 64)
		return redisReply{kind: ':', integer: n}, err
	case '$':
		line, err := readLine(r)
		if err != nil {
			return redisReply{}, err
		}
		n, err := strconv.Atoi(line)
		if err != nil {
			return redisReply{}, err
		}
		if n == -1 {
			return redisReply{kind: '$', nil: true}, nil
		}
		buf := make([]byte, n+2)
		if _, err := io.ReadFull(r, buf); err != nil {
			return redisReply{}, err
		}
		return redisReply{kind: '$', bulk: string(buf[:n])}, nil
	case '*':
		line, err := readLine(r)
		if err != nil {
			return redisReply{}, err
		}
		n, err := strconv.Atoi(line)
		if err != nil {
			return redisReply{}, err
		}
		items := make([]redisReply, 0, n)
		for i := 0; i < n; i++ {
			item, err := readReply(r)
			if err != nil {
				return redisReply{}, err
			}
			items = append(items, item)
		}
		return redisReply{kind: '*', array: items}, nil
	default:
		return redisReply{}, fmt.Errorf("hotstate: unknown redis reply prefix %q", prefix)
	}
}

func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), nil
}
