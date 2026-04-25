package crawl

import (
	"time"

	"github.com/rapid-saas/aegis-adult/hasher-crawler/internal/hash"
	"github.com/rapid-saas/aegis-adult/hasher-crawler/internal/store"
)

type Policy struct {
	RateLimitPerSecond float64
	RespectRobots      bool
}

type Candidate struct {
	URL   string
	PHash uint64
}

type Match struct {
	AssetID      string
	CandidateURL string
	Hamming      int
	Score        float64
	Status       string
	DetectedAt   time.Time
}

type Crawler struct {
	policy Policy
}

func NewCrawler(policy Policy) Crawler {
	if policy.RateLimitPerSecond <= 0 {
		policy.RateLimitPerSecond = 1
	}
	return Crawler{policy: policy}
}

func (c Crawler) MatchCandidates(assets []store.Asset, candidates []Candidate) []Match {
	var matches []Match
	for _, asset := range assets {
		for _, candidate := range candidates {
			distance := hash.HammingDistance(asset.PHash, candidate.PHash)
			status := ReviewStatus(distance)
			if status == "no_match" {
				continue
			}
			matches = append(matches, Match{
				AssetID:      asset.ID,
				CandidateURL: candidate.URL,
				Hamming:      distance,
				Score:        ScoreFromHamming(distance),
				Status:       status,
				DetectedAt:   time.Now().UTC(),
			})
		}
	}
	return matches
}

func ReviewStatus(distance int) string {
	switch {
	case distance <= 6:
		return "match_requires_review"
	case distance <= 12:
		return "possible_match_requires_review"
	default:
		return "no_match"
	}
}

func ScoreFromHamming(distance int) float64 {
	if distance < 0 {
		return 0
	}
	if distance > 64 {
		return 0
	}
	return float64(64-distance) / 64
}
