package security

import "testing"

func TestEncryptDecryptSecret(t *testing.T) {
	ciphertext, err := EncryptSecret("local-test-key", "super-secret")
	if err != nil {
		t.Fatalf("EncryptSecret() error = %v", err)
	}
	if ciphertext == "super-secret" {
		t.Fatal("ciphertext must not equal plaintext")
	}
	plaintext, err := DecryptSecret("local-test-key", ciphertext)
	if err != nil {
		t.Fatalf("DecryptSecret() error = %v", err)
	}
	if plaintext != "super-secret" {
		t.Fatalf("wrong plaintext: %q", plaintext)
	}
}

func TestRedactSecret(t *testing.T) {
	if got := RedactSecret("abcd1234wxyz"); got != "abcd...redacted...wxyz" {
		t.Fatalf("unexpected redaction: %q", got)
	}
}
