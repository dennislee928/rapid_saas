package linker

import "testing"

func TestJaccard(t *testing.T) {
	got := Jaccard([]string{"a", "b", "c"}, []string{"b", "c", "d"})
	if got < 0.49 || got > 0.51 {
		t.Fatalf("expected jaccard around 0.5, got %f", got)
	}
}

func TestServiceLinksSimilarAccounts(t *testing.T) {
	service := NewService(Config{SimilarityThreshold: 0.8})
	first := Event{
		TenantID:  "ten_demo",
		AccountID: "acct_1",
		DeviceID:  "dev_1",
		Features:  []string{"canvas:a", "webgl:b", "font:c", "tz:gb"},
		SimHash:   0b11110000,
	}
	second := Event{
		TenantID:  "ten_demo",
		AccountID: "acct_2",
		DeviceID:  "dev_2",
		Features:  []string{"canvas:a", "webgl:b", "font:c", "tz:gb", "screen:wide"},
		SimHash:   0b11110001,
	}

	service.Link(first)
	result := service.Link(second)

	if result.CandidateCount != 1 {
		t.Fatalf("expected one candidate, got %d", result.CandidateCount)
	}
	if len(result.InsertedLinks) != 1 {
		t.Fatalf("expected one inserted link, got %d", len(result.InsertedLinks))
	}
	if len(result.MinHashBandKeys) != 16 {
		t.Fatalf("expected 16 band keys, got %d", len(result.MinHashBandKeys))
	}
}
