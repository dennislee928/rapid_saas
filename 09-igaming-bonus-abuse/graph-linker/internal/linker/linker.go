package linker

import (
	"crypto/sha256"
	"encoding/binary"
	"math/bits"
	"sort"
)

type Config struct {
	SimilarityThreshold float64
}

type Event struct {
	TenantID        string   `json:"tenant_id"`
	AccountID       string   `json:"account_id"`
	DeviceID        string   `json:"device_id"`
	FingerprintHash string   `json:"fingerprint_hash"`
	Features        []string `json:"features"`
	SimHash         uint64   `json:"simhash"`
}

type Link struct {
	SourceAccountID string   `json:"source_account_id"`
	TargetAccountID string   `json:"target_account_id"`
	Via             []string `json:"via"`
	Confidence      float64  `json:"confidence"`
}

type Result struct {
	DeviceID        string `json:"device_id"`
	CandidateCount  int    `json:"candidate_count"`
	InsertedLinks   []Link `json:"inserted_links"`
	MinHashBandKeys []Band `json:"minhash_band_keys"`
}

type Band struct {
	Index uint32 `json:"index"`
	Hash  uint64 `json:"hash"`
}

type Service struct {
	threshold float64
	seen      []Event
}

func NewService(config Config) *Service {
	threshold := config.SimilarityThreshold
	if threshold == 0 {
		threshold = 0.82
	}
	return &Service{threshold: threshold}
}

func (s *Service) Link(event Event) Result {
	bands := MinHashBands(event.Features, 16)
	result := Result{DeviceID: event.DeviceID, MinHashBandKeys: bands}
	for _, candidate := range s.seen {
		if candidate.TenantID != event.TenantID || candidate.AccountID == event.AccountID {
			continue
		}
		result.CandidateCount++
		jaccard := Jaccard(event.Features, candidate.Features)
		simDistance := bits.OnesCount64(event.SimHash ^ candidate.SimHash)
		confidence := (jaccard * 0.85) + ((1 - float64(simDistance)/64) * 0.15)
		if confidence >= s.threshold {
			result.InsertedLinks = append(result.InsertedLinks, Link{
				SourceAccountID: event.AccountID,
				TargetAccountID: candidate.AccountID,
				Via:             []string{"fingerprint_minhash"},
				Confidence:      confidence,
			})
		}
	}
	s.seen = append(s.seen, event)
	return result
}

func Jaccard(a, b []string) float64 {
	left := set(a)
	right := set(b)
	if len(left) == 0 && len(right) == 0 {
		return 1
	}
	intersection := 0
	for value := range left {
		if right[value] {
			intersection++
		}
	}
	union := len(left) + len(right) - intersection
	return float64(intersection) / float64(union)
}

func MinHashBands(features []string, bands int) []Band {
	if bands <= 0 {
		return nil
	}
	values := append([]string(nil), features...)
	sort.Strings(values)
	result := make([]Band, 0, bands)
	for i := 0; i < bands; i++ {
		h := sha256.New()
		var prefix [4]byte
		binary.BigEndian.PutUint32(prefix[:], uint32(i))
		_, _ = h.Write(prefix[:])
		for _, feature := range values {
			_, _ = h.Write([]byte{0})
			_, _ = h.Write([]byte(feature))
		}
		sum := h.Sum(nil)
		result = append(result, Band{Index: uint32(i), Hash: binary.BigEndian.Uint64(sum[:8])})
	}
	return result
}

func set(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}
