package events

import "time"

type RetryPolicy struct {
	Delays []time.Duration
}

func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{Delays: []time.Duration{
		30 * time.Second,
		2 * time.Minute,
		10 * time.Minute,
		time.Hour,
		6 * time.Hour,
	}}
}

func (p RetryPolicy) MaxAttempts() int {
	return len(p.Delays) + 1
}

func (p RetryPolicy) ShouldDLQ(attempt int) bool {
	return attempt >= p.MaxAttempts()
}

func (p RetryPolicy) NextDelay(attempt int) time.Duration {
	if attempt < 1 || attempt > len(p.Delays) {
		return 0
	}
	return p.Delays[attempt-1]
}
