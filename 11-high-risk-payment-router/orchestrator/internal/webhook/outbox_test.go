package webhook

import (
	"strings"
	"testing"
	"time"
)

func TestSignPayloadShape(t *testing.T) {
	sig, err := SignPayload("secret", 1710000000, map[string]string{"event": "transaction.captured"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(sig, "t=1710000000,v1=") {
		t.Fatalf("unexpected signature shape: %s", sig)
	}
}

func TestRetrySchedule(t *testing.T) {
	if NextRetryDelay(0) != 30*time.Second {
		t.Fatal("first retry should be 30 seconds")
	}
	if NextRetryDelay(7) != 72*time.Hour {
		t.Fatal("eighth retry should be 72 hours")
	}
	if NextRetryDelay(8) != 0 {
		t.Fatal("attempts beyond schedule should be dead-lettered")
	}
}

