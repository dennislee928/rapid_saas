package delivery

import (
	"net"
	"testing"
)

func TestBlockedIP(t *testing.T) {
	blocked := []string{"127.0.0.1", "10.0.0.1", "172.16.0.1", "192.168.1.1", "169.254.1.1", "::1", "fc00::1"}
	for _, value := range blocked {
		if !blockedIP(net.ParseIP(value)) {
			t.Fatalf("expected %s to be blocked", value)
		}
	}
	if blockedIP(net.ParseIP("8.8.8.8")) {
		t.Fatal("expected public IP to be allowed")
	}
}
