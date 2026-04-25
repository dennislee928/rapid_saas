package webhook

import (
	"errors"
	"sync"
	"time"
)

type InboundEvent struct {
	ID                string            `json:"id"`
	PSPCode           string            `json:"psp_code"`
	EventID           string            `json:"event_id"`
	SignatureVerified bool              `json:"signature_verified"`
	Headers           map[string]string `json:"headers"`
	RawBody           []byte            `json:"-"`
	Status            string            `json:"status"`
	ProcessedAt       *time.Time        `json:"processed_at,omitempty"`
}

type IngressStore struct {
	mu     sync.Mutex
	events map[string]InboundEvent
}

func NewIngressStore() *IngressStore {
	return &IngressStore{events: map[string]InboundEvent{}}
}

func (s *IngressStore) Insert(event InboundEvent) (bool, error) {
	if event.PSPCode == "" || event.EventID == "" {
		return false, errors.New("psp code and event id are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := event.PSPCode + ":" + event.EventID
	if _, ok := s.events[key]; ok {
		return false, nil
	}
	if event.ID == "" {
		event.ID = "whin_" + key
	}
	if event.Status == "" {
		event.Status = "received"
	}
	s.events[key] = event
	return true, nil
}
