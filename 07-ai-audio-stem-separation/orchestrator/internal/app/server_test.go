package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rapid-saas/audio-stem-separation/orchestrator/internal/store"
)

func TestCalculateCostMillicredits(t *testing.T) {
	if got := CalculateCostMillicredits(60, "htdemucs"); got != 49980 {
		t.Fatalf("expected one minute htdemucs cost near 50000, got %d", got)
	}
	if got := CalculateCostMillicredits(60, "htdemucs_ft"); got != 99960 {
		t.Fatalf("expected ft multiplier, got %d", got)
	}
}

func TestCreateJobDebitsCreditsAndIsIdempotent(t *testing.T) {
	mem := store.NewMemory()
	mem.SeedUser("u1", "u1@example.com", 100000)
	srv := NewServer(mem, Config{})

	body := map[string]any{
		"user_id":         "u1",
		"idempotency_key": "idem-1",
		"model":           "htdemucs",
		"input_r2_key":    "in/test.wav",
		"input_bytes":     2048,
		"input_seconds":   30,
	}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewReader(payload))
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", rec.Code, rec.Body.String())
	}
	firstBalance := mem.Balance("u1")

	req = httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewReader(payload))
	rec = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected idempotent 200, got %d", rec.Code)
	}
	if got := mem.Balance("u1"); got != firstBalance {
		t.Fatalf("idempotent replay changed balance: before=%d after=%d", firstBalance, got)
	}
}

func TestFailureRefundsCredits(t *testing.T) {
	mem := store.NewMemory()
	mem.SeedUser("u1", "u1@example.com", 100000)
	job, _, err := mem.CreateJob(store.Job{
		UserID:       "u1",
		Model:        "htdemucs",
		InputR2Key:   "in/test.wav",
		InputBytes:   2048,
		InputSeconds: 30,
		CostCredits:  24990,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mem.FailAndRefund(job.ID, "space_error"); err != nil {
		t.Fatal(err)
	}
	if got := mem.Balance("u1"); got != 100000 {
		t.Fatalf("expected refund to restore balance, got %d", got)
	}
}
