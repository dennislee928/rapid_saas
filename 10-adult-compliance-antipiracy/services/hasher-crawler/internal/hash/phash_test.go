package hash

import "testing"

func TestImagePHashDeterministic(t *testing.T) {
	a := ImagePHash([]byte("same input"))
	b := ImagePHash([]byte("same input"))
	if a != b {
		t.Fatalf("expected deterministic hash, got %x and %x", a, b)
	}
}

func TestHammingDistance(t *testing.T) {
	if got := HammingDistance(0b1010, 0b1001); got != 2 {
		t.Fatalf("expected distance 2, got %d", got)
	}
}
