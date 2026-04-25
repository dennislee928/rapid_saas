package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type OutboundEvent struct {
	ID            string    `json:"id"`
	MerchantID    string    `json:"merchant_id"`
	TransactionID string    `json:"transaction_id"`
	EventType     string    `json:"event_type"`
	Payload       any       `json:"payload"`
	Attempts      int       `json:"attempts"`
	NextAttemptAt time.Time `json:"next_attempt_at"`
	Status        string    `json:"status"`
}

type MemoryOutbox struct {
	mu     sync.Mutex
	events []OutboundEvent
}

func NewMemoryOutbox() *MemoryOutbox {
	return &MemoryOutbox{}
}

func (o *MemoryOutbox) Enqueue(event OutboundEvent) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if event.ID == "" {
		event.ID = fmt.Sprintf("whout_%d", time.Now().UnixNano())
	}
	if event.Status == "" {
		event.Status = "pending"
	}
	if event.NextAttemptAt.IsZero() {
		event.NextAttemptAt = time.Now().Add(30 * time.Second)
	}
	o.events = append(o.events, event)
}

func (o *MemoryOutbox) Events() []OutboundEvent {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]OutboundEvent(nil), o.events...)
}

func SignPayload(secret string, timestamp int64, payload any) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("%d.", timestamp)))
	mac.Write(body)
	return fmt.Sprintf("t=%d,v1=%s", timestamp, hex.EncodeToString(mac.Sum(nil))), nil
}

func NextRetryDelay(attempt int) time.Duration {
	delays := []time.Duration{
		30 * time.Second,
		time.Minute,
		5 * time.Minute,
		15 * time.Minute,
		time.Hour,
		6 * time.Hour,
		24 * time.Hour,
		72 * time.Hour,
	}
	if attempt < 0 {
		attempt = 0
	}
	if attempt >= len(delays) {
		return 0
	}
	return delays[attempt]
}

