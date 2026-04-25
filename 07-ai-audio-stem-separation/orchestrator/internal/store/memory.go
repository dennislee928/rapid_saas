package store

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrInsufficientCredits = errors.New("insufficient credits")
	ErrJobNotFound         = errors.New("job not found")
)

type User struct {
	ID        string
	Email     string
	CreatedAt int64
}

type LedgerEntry struct {
	ID            string
	UserID        string
	Delta         int
	Kind          string
	JobID         string
	StripeEventID string
	Note          string
	CreatedAt     int64
}

type Job struct {
	ID             string         `json:"id"`
	UserID         string         `json:"user_id"`
	IdempotencyKey string         `json:"idempotency_key,omitempty"`
	Model          string         `json:"model"`
	Params         map[string]any `json:"params"`
	InputR2Key     string         `json:"input_r2_key"`
	InputBytes     int64          `json:"input_bytes"`
	InputSeconds   float64        `json:"input_seconds"`
	CostCredits    int            `json:"cost_credits"`
	Status         string         `json:"status"`
	ErrorCode      string         `json:"error_code,omitempty"`
	OutputZipKey   string         `json:"output_zip_key,omitempty"`
	SpaceURL       string         `json:"space_url,omitempty"`
	CreatedAt      int64          `json:"created_at"`
	StartedAt      int64          `json:"started_at,omitempty"`
	FinishedAt     int64          `json:"finished_at,omitempty"`
	LastHeartbeat  int64          `json:"last_heartbeat,omitempty"`
}

type Memory struct {
	mu          sync.Mutex
	users       map[string]User
	balances    map[string]int
	ledger      []LedgerEntry
	jobs        map[string]Job
	idempotency map[string]string
}

func NewMemory() *Memory {
	return &Memory{
		users:       map[string]User{},
		balances:    map[string]int{},
		jobs:        map[string]Job{},
		idempotency: map[string]string{},
	}
}

func (m *Memory) SeedUser(id, email string, credits int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().Unix()
	m.users[id] = User{ID: id, Email: email, CreatedAt: now}
	m.balances[id] += credits
	m.ledger = append(m.ledger, LedgerEntry{ID: newID("ledge"), UserID: id, Delta: credits, Kind: "grant", Note: "seed", CreatedAt: now})
}

func (m *Memory) Balance(userID string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.balances[userID]
}

func (m *Memory) CreateJob(job Job) (Job, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if job.UserID == "" {
		job.UserID = "user_demo"
	}
	if job.IdempotencyKey != "" {
		key := job.UserID + ":" + job.IdempotencyKey
		if existingID := m.idempotency[key]; existingID != "" {
			return m.jobs[existingID], true, nil
		}
	}
	if m.balances[job.UserID] < job.CostCredits {
		return Job{}, false, ErrInsufficientCredits
	}

	now := time.Now().Unix()
	if job.ID == "" {
		job.ID = newID("job")
	}
	job.Status = "queued"
	job.CreatedAt = now
	m.jobs[job.ID] = job
	m.balances[job.UserID] -= job.CostCredits
	m.ledger = append(m.ledger, LedgerEntry{ID: newID("ledge"), UserID: job.UserID, Delta: -job.CostCredits, Kind: "consume", JobID: job.ID, CreatedAt: now})
	if job.IdempotencyKey != "" {
		m.idempotency[job.UserID+":"+job.IdempotencyKey] = job.ID
	}
	return job, false, nil
}

func (m *Memory) GetJob(id string) (Job, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[id]
	return job, ok
}

func (m *Memory) MarkDispatched(id, spaceURL string) (Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[id]
	if !ok {
		return Job{}, ErrJobNotFound
	}
	now := time.Now().Unix()
	job.Status = "dispatched"
	job.SpaceURL = spaceURL
	job.StartedAt = now
	job.LastHeartbeat = now
	m.jobs[id] = job
	return job, nil
}

func (m *Memory) Heartbeat(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[id]
	if !ok {
		return ErrJobNotFound
	}
	job.LastHeartbeat = time.Now().Unix()
	m.jobs[id] = job
	return nil
}

func (m *Memory) Complete(id, zipKey string) (Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[id]
	if !ok {
		return Job{}, ErrJobNotFound
	}
	job.Status = "done"
	job.OutputZipKey = zipKey
	job.FinishedAt = time.Now().Unix()
	m.jobs[id] = job
	return job, nil
}

func (m *Memory) FailAndRefund(id, code string) (Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[id]
	if !ok {
		return Job{}, ErrJobNotFound
	}
	if job.Status != "failed" && job.Status != "failed_lost" {
		m.balances[job.UserID] += job.CostCredits
		m.ledger = append(m.ledger, LedgerEntry{ID: newID("ledge"), UserID: job.UserID, Delta: job.CostCredits, Kind: "refund", JobID: job.ID, Note: code, CreatedAt: time.Now().Unix()})
	}
	job.Status = "failed"
	job.ErrorCode = code
	job.FinishedAt = time.Now().Unix()
	m.jobs[id] = job
	return job, nil
}

func newID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}
