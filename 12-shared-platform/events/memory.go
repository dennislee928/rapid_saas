package events

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrDuplicateSideEffect = errors.New("duplicate side effect skipped")

type ConsumerFunc func(context.Context, Envelope) error

type PublishedEvent struct {
	Envelope Envelope
	Topic    string
	Attempt  int
	Visible  time.Time
	LastErr  string
}

type DeadLetter struct {
	Envelope Envelope
	Topic    string
	Attempt  int
	Reason   string
	FailedAt time.Time
}

type IdempotencyStore interface {
	Seen(key string) bool
	Mark(key string)
}

type MemoryQueue struct {
	mu      sync.Mutex
	now     func() time.Time
	policy  RetryPolicy
	events  []PublishedEvent
	dlq     []DeadLetter
	handled IdempotencyStore
}

func NewMemoryQueue(policy RetryPolicy, handled IdempotencyStore) *MemoryQueue {
	if len(policy.Delays) == 0 {
		policy = DefaultRetryPolicy()
	}
	if handled == nil {
		handled = NewMemoryIdempotencyStore()
	}
	return &MemoryQueue{
		now:     func() time.Time { return time.Now().UTC() },
		policy:  policy,
		handled: handled,
	}
}

func (q *MemoryQueue) Publish(_ context.Context, topic string, envelope Envelope) error {
	if err := envelope.Validate(); err != nil {
		return err
	}
	if topic == "" {
		return errors.New("topic is required")
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	q.events = append(q.events, PublishedEvent{
		Envelope: envelope,
		Topic:    topic,
		Attempt:  1,
		Visible:  q.now(),
	})
	return nil
}

func (q *MemoryQueue) DrainReady(ctx context.Context, topic string, consumer ConsumerFunc) (int, error) {
	if consumer == nil {
		return 0, errors.New("consumer is required")
	}
	var processed int
	for {
		event, ok := q.popReady(topic)
		if !ok {
			return processed, nil
		}
		processed++
		if q.handled.Seen(event.Envelope.IdempotencyKey) {
			continue
		}
		if err := consumer(ctx, event.Envelope); err != nil {
			q.rescheduleOrDLQ(event, err)
			continue
		}
		q.handled.Mark(event.Envelope.IdempotencyKey)
	}
}

func (q *MemoryQueue) ReplayDLQ(_ context.Context, topic string, predicate func(DeadLetter) bool) int {
	q.mu.Lock()
	defer q.mu.Unlock()
	var replayed int
	remaining := q.dlq[:0]
	for _, letter := range q.dlq {
		if letter.Topic != topic || (predicate != nil && !predicate(letter)) {
			remaining = append(remaining, letter)
			continue
		}
		q.events = append(q.events, PublishedEvent{
			Envelope: letter.Envelope,
			Topic:    letter.Topic,
			Attempt:  1,
			Visible:  q.now(),
		})
		replayed++
	}
	q.dlq = remaining
	return replayed
}

func (q *MemoryQueue) Pending() []PublishedEvent {
	q.mu.Lock()
	defer q.mu.Unlock()
	return append([]PublishedEvent(nil), q.events...)
}

func (q *MemoryQueue) DLQ() []DeadLetter {
	q.mu.Lock()
	defer q.mu.Unlock()
	return append([]DeadLetter(nil), q.dlq...)
}

func (q *MemoryQueue) SetNowForTest(now func() time.Time) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.now = now
}

func (q *MemoryQueue) popReady(topic string) (PublishedEvent, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	now := q.now()
	for i, event := range q.events {
		if event.Topic != topic || event.Visible.After(now) {
			continue
		}
		q.events = append(q.events[:i], q.events[i+1:]...)
		return event, true
	}
	return PublishedEvent{}, false
}

func (q *MemoryQueue) rescheduleOrDLQ(event PublishedEvent, err error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	reason := err.Error()
	if q.policy.ShouldDLQ(event.Attempt) {
		q.dlq = append(q.dlq, DeadLetter{
			Envelope: event.Envelope,
			Topic:    event.Topic,
			Attempt:  event.Attempt,
			Reason:   reason,
			FailedAt: q.now(),
		})
		return
	}
	event.LastErr = reason
	event.Visible = q.now().Add(q.policy.NextDelay(event.Attempt))
	event.Attempt++
	q.events = append(q.events, event)
}

type MemoryIdempotencyStore struct {
	mu   sync.Mutex
	keys map[string]struct{}
}

func NewMemoryIdempotencyStore() *MemoryIdempotencyStore {
	return &MemoryIdempotencyStore{keys: make(map[string]struct{})}
}

func (s *MemoryIdempotencyStore) Seen(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.keys[key]
	return ok
}

func (s *MemoryIdempotencyStore) Mark(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys[key] = struct{}{}
}

type SideEffectRecorder struct {
	mu      sync.Mutex
	records map[string]int
}

func NewSideEffectRecorder() *SideEffectRecorder {
	return &SideEffectRecorder{records: make(map[string]int)}
}

func (r *SideEffectRecorder) Apply(idempotencyKey string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.records[idempotencyKey] > 0 {
		return fmt.Errorf("%w: %s", ErrDuplicateSideEffect, idempotencyKey)
	}
	r.records[idempotencyKey]++
	return nil
}

func (r *SideEffectRecorder) Count(idempotencyKey string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.records[idempotencyKey]
}
