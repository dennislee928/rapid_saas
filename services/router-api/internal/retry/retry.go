package retry

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/rapid-saas/router-api/internal/model"
)

type Job struct {
	Event       model.QueueEvent
	Rule        model.Rule
	Destination model.Destination
	Attempt     int
	DueAt       time.Time
	LastError   string
}

type Policy interface {
	MaxAttempts() int
	Delay(int) time.Duration
}

type Scheduler interface {
	Policy() Policy
	Schedule(context.Context, Job) error
}

type DLQ interface {
	Park(context.Context, Job) error
}

type BackoffPolicy struct {
	delays []time.Duration
}

func DefaultPolicy() BackoffPolicy {
	return BackoffPolicy{delays: []time.Duration{30 * time.Second, 2 * time.Minute, 10 * time.Minute, time.Hour, 6 * time.Hour}}
}

func (p BackoffPolicy) MaxAttempts() int {
	return len(p.delays)
}

func (p BackoffPolicy) Delay(attempt int) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}
	if attempt > len(p.delays) {
		attempt = len(p.delays)
	}
	base := p.delays[attempt-1]
	jitter := 0.8 + rand.Float64()*0.4
	return time.Duration(float64(base) * jitter)
}

type MemoryScheduler struct {
	mu     sync.Mutex
	policy Policy
	jobs   []Job
}

func NewMemoryScheduler(policy Policy) *MemoryScheduler {
	return &MemoryScheduler{policy: policy}
}

func (s *MemoryScheduler) Policy() Policy {
	return s.policy
}

func (s *MemoryScheduler) Schedule(_ context.Context, job Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs = append(s.jobs, job)
	return nil
}

type MemoryDLQ struct {
	mu   sync.Mutex
	jobs []Job
}

func NewMemoryDLQ() *MemoryDLQ {
	return &MemoryDLQ{}
}

func (q *MemoryDLQ) Park(_ context.Context, job Job) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.jobs = append(q.jobs, job)
	return nil
}
