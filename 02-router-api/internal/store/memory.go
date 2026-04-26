package store

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"encoding/base32"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rapid-saas/router-api/internal/model"
)

type MemoryStore struct {
	mu           sync.RWMutex
	endpoints    map[string]model.Endpoint
	destinations map[string]model.Destination
	rules        map[string]model.Rule
	logs         []model.DeliveryLog
	dlq          map[string]model.DLQEntry
	usage        map[string]model.UsageSummary
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		endpoints:    map[string]model.Endpoint{},
		destinations: map[string]model.Destination{},
		rules:        map[string]model.Rule{},
		dlq:          map[string]model.DLQEntry{},
		usage:        map[string]model.UsageSummary{},
	}
}

func (s *MemoryStore) Ping(context.Context) error { return nil }

func (s *MemoryStore) ListEndpoints(_ context.Context, tenantID string) ([]model.Endpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []model.Endpoint
	for _, endpoint := range s.endpoints {
		if endpoint.TenantID == tenantID {
			out = append(out, endpoint)
		}
	}
	return out, nil
}

func (s *MemoryStore) CreateEndpoint(_ context.Context, endpoint *model.Endpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	ensureEndpointDefaults(endpoint)
	if _, ok := s.endpoints[endpoint.ID]; ok {
		return ErrConflict
	}
	s.endpoints[endpoint.ID] = *endpoint
	return nil
}

func (s *MemoryStore) GetEndpoint(_ context.Context, tenantID, endpointID string) (model.Endpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	endpoint, ok := s.endpoints[endpointID]
	if !ok || endpoint.TenantID != tenantID {
		return model.Endpoint{}, ErrNotFound
	}
	return endpoint, nil
}

func (s *MemoryStore) UpdateEndpoint(_ context.Context, endpoint *model.Endpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.endpoints[endpoint.ID]
	if !ok || current.TenantID != endpoint.TenantID {
		return ErrNotFound
	}
	if endpoint.CreatedAt == 0 {
		endpoint.CreatedAt = current.CreatedAt
	}
	s.endpoints[endpoint.ID] = *endpoint
	return nil
}

func (s *MemoryStore) DeleteEndpoint(_ context.Context, tenantID, endpointID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.endpoints[endpointID]
	if !ok || current.TenantID != tenantID {
		return ErrNotFound
	}
	delete(s.endpoints, endpointID)
	return nil
}

func (s *MemoryStore) ListDestinations(_ context.Context, tenantID string) ([]model.Destination, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []model.Destination
	for _, destination := range s.destinations {
		if destination.TenantID == tenantID {
			out = append(out, destination)
		}
	}
	return out, nil
}

func (s *MemoryStore) CreateDestination(_ context.Context, destination *model.Destination) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	ensureDestinationDefaults(destination)
	if _, ok := s.destinations[destination.ID]; ok {
		return ErrConflict
	}
	s.destinations[destination.ID] = *destination
	return nil
}

func (s *MemoryStore) GetDestination(_ context.Context, tenantID, destinationID string) (model.Destination, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	destination, ok := s.destinations[destinationID]
	if !ok || destination.TenantID != tenantID {
		return model.Destination{}, ErrNotFound
	}
	return destination, nil
}

func (s *MemoryStore) UpdateDestination(_ context.Context, destination *model.Destination) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.destinations[destination.ID]
	if !ok || current.TenantID != destination.TenantID {
		return ErrNotFound
	}
	if destination.CreatedAt == 0 {
		destination.CreatedAt = current.CreatedAt
	}
	s.destinations[destination.ID] = *destination
	return nil
}

func (s *MemoryStore) DeleteDestination(_ context.Context, tenantID, destinationID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.destinations[destinationID]
	if !ok || current.TenantID != tenantID {
		return ErrNotFound
	}
	delete(s.destinations, destinationID)
	return nil
}

func (s *MemoryStore) ListRules(_ context.Context, tenantID, endpointID string) ([]model.Rule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []model.Rule
	for _, rule := range s.rules {
		if rule.TenantID == tenantID && (endpointID == "" || rule.EndpointID == endpointID) {
			out = append(out, rule)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Position == out[j].Position {
			return out[i].CreatedAt < out[j].CreatedAt
		}
		return out[i].Position < out[j].Position
	})
	return out, nil
}

func (s *MemoryStore) CreateRule(_ context.Context, rule *model.Rule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	ensureRuleDefaults(rule)
	if _, ok := s.rules[rule.ID]; ok {
		return ErrConflict
	}
	s.rules[rule.ID] = *rule
	return nil
}

func (s *MemoryStore) GetRule(_ context.Context, tenantID, ruleID string) (model.Rule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rule, ok := s.rules[ruleID]
	if !ok || rule.TenantID != tenantID {
		return model.Rule{}, ErrNotFound
	}
	return rule, nil
}

func (s *MemoryStore) UpdateRule(_ context.Context, rule *model.Rule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.rules[rule.ID]
	if !ok || current.TenantID != rule.TenantID {
		return ErrNotFound
	}
	if rule.CreatedAt == 0 {
		rule.CreatedAt = current.CreatedAt
	}
	s.rules[rule.ID] = *rule
	return nil
}

func (s *MemoryStore) DeleteRule(_ context.Context, tenantID, ruleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.rules[ruleID]
	if !ok || current.TenantID != tenantID {
		return ErrNotFound
	}
	delete(s.rules, ruleID)
	return nil
}

func (s *MemoryStore) WriteDeliveryLog(_ context.Context, log model.DeliveryLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if log.ID == "" {
		log.ID = "dlog_" + randomID()
	}
	s.logs = append(s.logs, log)
	return nil
}

func (s *MemoryStore) ListDeliveryLogs(_ context.Context, tenantID string, limit int) ([]model.DeliveryLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > 250 {
		limit = 50
	}
	var out []model.DeliveryLog
	for i := len(s.logs) - 1; i >= 0 && len(out) < limit; i-- {
		if s.logs[i].TenantID == tenantID {
			out = append(out, s.logs[i])
		}
	}
	return out, nil
}

func (s *MemoryStore) IncrementUsage(_ context.Context, tenantID, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	usage := s.usage[tenantID]
	usage.TenantID = tenantID
	usage.Window = "memory-lifetime"
	usage.Ingressed++
	switch status {
	case "delivered":
		usage.Forwarded++
	case "failed", "dlq":
		usage.Failed++
	}
	s.usage[tenantID] = usage
	return nil
}

func (s *MemoryStore) UsageSummary(_ context.Context, tenantID string) (model.UsageSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	usage := s.usage[tenantID]
	if usage.TenantID == "" {
		usage.TenantID = tenantID
		usage.Window = "memory-lifetime"
	}
	return usage, nil
}

func (s *MemoryStore) ParkDLQ(_ context.Context, entry model.DLQEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if entry.ID == "" {
		entry.ID = "dlq_" + randomID()
	}
	if entry.ParkedAt == 0 {
		entry.ParkedAt = time.Now().UnixMilli()
	}
	s.dlq[entry.ID] = entry
	return nil
}

func (s *MemoryStore) ListDLQ(_ context.Context, tenantID string, limit int) ([]model.DLQEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > 250 {
		limit = 50
	}
	out := make([]model.DLQEntry, 0, len(s.dlq))
	for _, entry := range s.dlq {
		if entry.TenantID == tenantID {
			out = append(out, entry)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].ParkedAt > out[j].ParkedAt
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *MemoryStore) ReplayDLQ(_ context.Context, tenantID, dlqID string) (model.QueueEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.dlq[dlqID]
	if !ok || entry.TenantID != tenantID {
		return model.QueueEvent{}, ErrNotFound
	}
	delete(s.dlq, dlqID)
	body, err := base64.StdEncoding.DecodeString(entry.PayloadB64)
	if err != nil {
		return model.QueueEvent{}, err
	}
	raw := json.RawMessage(body)
	if !json.Valid(raw) {
		encoded, err := json.Marshal(string(body))
		if err != nil {
			return model.QueueEvent{}, err
		}
		raw = encoded
	}
	return model.QueueEvent{
		ID:         "replay_" + randomID(),
		TenantID:   tenantID,
		EndpointID: entry.EndpointID,
		Body:       raw,
		ReceivedAt: time.Now().UnixMilli(),
	}, nil
}

func ensureEndpointDefaults(endpoint *model.Endpoint) {
	if endpoint.ID == "" {
		endpoint.ID = "ep_" + randomID()
	}
	if endpoint.CreatedAt == 0 {
		endpoint.CreatedAt = time.Now().UnixMilli()
	}
}

func ensureDestinationDefaults(destination *model.Destination) {
	if destination.ID == "" {
		destination.ID = "dest_" + randomID()
	}
	if destination.CreatedAt == 0 {
		destination.CreatedAt = time.Now().UnixMilli()
	}
}

func ensureRuleDefaults(rule *model.Rule) {
	if rule.ID == "" {
		rule.ID = "rule_" + randomID()
	}
	if rule.TransformKind == "" {
		rule.TransformKind = "passthrough"
	}
	if rule.OnMatch == "" {
		rule.OnMatch = "forward"
	}
	if rule.CreatedAt == 0 {
		rule.CreatedAt = time.Now().UnixMilli()
	}
}

func randomID() string {
	var b [10]byte
	if _, err := rand.Read(b[:]); err != nil {
		return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte(time.Now().Format(time.RFC3339Nano))))
	}
	return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b[:]))
}
