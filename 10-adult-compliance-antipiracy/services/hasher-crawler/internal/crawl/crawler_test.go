package crawl

import (
	"testing"

	"github.com/rapid-saas/aegis-adult/hasher-crawler/internal/hash"
	"github.com/rapid-saas/aegis-adult/hasher-crawler/internal/store"
)

func TestReviewStatusHumanReviewDefault(t *testing.T) {
	if got := ReviewStatus(4); got != "match_requires_review" {
		t.Fatalf("expected human-review match, got %s", got)
	}
	if got := ReviewStatus(9); got != "possible_match_requires_review" {
		t.Fatalf("expected possible human-review match, got %s", got)
	}
	if got := ReviewStatus(20); got != "no_match" {
		t.Fatalf("expected no_match, got %s", got)
	}
}

func TestCrawlerMatchesCandidatesOffline(t *testing.T) {
	assetHash := hash.ImagePHash([]byte("reference"))
	crawler := NewCrawler(Policy{RateLimitPerSecond: 2, RespectRobots: true})
	matches := crawler.MatchCandidates(
		[]store.Asset{{ID: "asset_1", PHash: assetHash}},
		[]Candidate{{URL: "https://example.invalid/post", PHash: assetHash}},
	)
	if len(matches) != 1 {
		t.Fatalf("expected one match, got %d", len(matches))
	}
	if matches[0].Status != "match_requires_review" {
		t.Fatalf("expected review status, got %s", matches[0].Status)
	}
}
