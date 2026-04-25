package events

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestFailedConsumerRetryIsVisibleThenDLQ(t *testing.T) {
	now := time.Date(2026, 4, 26, 8, 0, 0, 0, time.UTC)
	queue := NewMemoryQueue(RetryPolicy{Delays: []time.Duration{time.Minute}}, nil)
	queue.SetNowForTest(func() time.Time { return now })
	env := mustEnvelope(t, "evt_retry", "idem_retry")

	if err := queue.Publish(context.Background(), "routekit.webhooks", env); err != nil {
		t.Fatal(err)
	}
	processed, err := queue.DrainReady(context.Background(), "routekit.webhooks", func(context.Context, Envelope) error {
		return errors.New("destination returned 500")
	})
	if err != nil {
		t.Fatal(err)
	}
	if processed != 1 {
		t.Fatalf("processed = %d, want 1", processed)
	}

	pending := queue.Pending()
	if len(pending) != 1 {
		t.Fatalf("pending = %d, want 1", len(pending))
	}
	if pending[0].Attempt != 2 || pending[0].LastErr != "destination returned 500" {
		t.Fatalf("unexpected retry state: %#v", pending[0])
	}
	if !pending[0].Visible.Equal(now.Add(time.Minute)) {
		t.Fatalf("retry visible at %s, want %s", pending[0].Visible, now.Add(time.Minute))
	}

	now = now.Add(time.Minute)
	processed, err = queue.DrainReady(context.Background(), "routekit.webhooks", func(context.Context, Envelope) error {
		return errors.New("poison payload")
	})
	if err != nil {
		t.Fatal(err)
	}
	if processed != 1 {
		t.Fatalf("processed = %d, want 1", processed)
	}
	letters := queue.DLQ()
	if len(letters) != 1 {
		t.Fatalf("dlq = %d, want 1", len(letters))
	}
	if letters[0].Reason != "poison payload" || letters[0].Attempt != 2 {
		t.Fatalf("unexpected dlq letter: %#v", letters[0])
	}
}

func TestReplaySkipsDuplicateSideEffects(t *testing.T) {
	queue := NewMemoryQueue(RetryPolicy{Delays: []time.Duration{0}}, nil)
	recorder := NewSideEffectRecorder()
	env := mustEnvelope(t, "evt_replay", "tenant:txn_123:webhook")

	if err := queue.Publish(context.Background(), "routekit.webhooks", env); err != nil {
		t.Fatal(err)
	}
	processed, err := queue.DrainReady(context.Background(), "routekit.webhooks", func(_ context.Context, event Envelope) error {
		return recorder.Apply(event.IdempotencyKey)
	})
	if err != nil {
		t.Fatal(err)
	}
	if processed != 1 || recorder.Count(env.IdempotencyKey) != 1 {
		t.Fatalf("first drain processed=%d side_effects=%d", processed, recorder.Count(env.IdempotencyKey))
	}

	if err := queue.Publish(context.Background(), "routekit.webhooks", env); err != nil {
		t.Fatal(err)
	}
	processed, err = queue.DrainReady(context.Background(), "routekit.webhooks", func(_ context.Context, event Envelope) error {
		return recorder.Apply(event.IdempotencyKey)
	})
	if err != nil {
		t.Fatal(err)
	}
	if processed != 1 {
		t.Fatalf("duplicate replay processed = %d, want 1 queue record", processed)
	}
	if recorder.Count(env.IdempotencyKey) != 1 {
		t.Fatalf("duplicate replay side effects = %d, want 1", recorder.Count(env.IdempotencyKey))
	}
}

func TestDLQReplayCanRecoverPoisonEvent(t *testing.T) {
	queue := NewMemoryQueue(RetryPolicy{Delays: []time.Duration{time.Nanosecond}}, nil)
	env := mustEnvelope(t, "evt_poison", "idem_poison")

	if err := queue.Publish(context.Background(), "routekit.webhooks", env); err != nil {
		t.Fatal(err)
	}
	_, err := queue.DrainReady(context.Background(), "routekit.webhooks", func(context.Context, Envelope) error {
		return errors.New("missing destination secret")
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = queue.DrainReady(context.Background(), "routekit.webhooks", func(context.Context, Envelope) error {
		return errors.New("missing destination secret")
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(queue.DLQ()) != 1 {
		t.Fatalf("dlq = %d, want 1", len(queue.DLQ()))
	}

	replayed := queue.ReplayDLQ(context.Background(), "routekit.webhooks", func(letter DeadLetter) bool {
		return letter.Envelope.EventID == "evt_poison"
	})
	if replayed != 1 {
		t.Fatalf("replayed = %d, want 1", replayed)
	}
	if len(queue.DLQ()) != 0 {
		t.Fatalf("dlq after replay = %d, want 0", len(queue.DLQ()))
	}
	processed, err := queue.DrainReady(context.Background(), "routekit.webhooks", func(context.Context, Envelope) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if processed != 1 {
		t.Fatalf("processed replay = %d, want 1", processed)
	}
}

func mustEnvelope(t *testing.T, eventID string, idempotencyKey string) Envelope {
	t.Helper()
	env, err := NewEnvelope(NewEnvelopeInput{
		EventID:        eventID,
		TenantID:       "tenant_routekit_demo",
		Type:           "routekit.webhook.delivery_requested",
		SchemaVersion:  1,
		OccurredAt:     time.Date(2026, 4, 26, 8, 0, 0, 0, time.UTC),
		IdempotencyKey: idempotencyKey,
		Payload:        map[string]string{"transaction_id": "txn_123"},
		TraceID:        "trace_123",
	})
	if err != nil {
		t.Fatal(err)
	}
	return env
}
