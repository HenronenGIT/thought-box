package state

import (
	"testing"
	"time"
)

func TestSignAndVerifyRoundTrip(t *testing.T) {
	signer := New(testKey(), nowAt("2026-05-23T12:00:00Z"))
	token := signer.Sign("nonce-123", 5*time.Minute)

	got, err := signer.Verify(token)
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if got != "nonce-123" {
		t.Errorf("nonce = %q, want %q", got, "nonce-123")
	}
}

func TestVerifyRejectsTamperedToken(t *testing.T) {
	signer := New(testKey(), nowAt("2026-05-23T12:00:00Z"))
	token := signer.Sign("nonce-123", 5*time.Minute)

	// Flip the last character of the signature.
	tampered := token[:len(token)-1] + flipChar(token[len(token)-1:])

	if _, err := signer.Verify(tampered); err != ErrInvalid {
		t.Fatalf("expected ErrInvalid for tampered token, got %v", err)
	}
}

func TestVerifyRejectsExpiredToken(t *testing.T) {
	signer := New(testKey(), nowAt("2026-05-23T12:00:00Z"))
	token := signer.Sign("nonce-123", 5*time.Minute)

	// Advance the clock past expiry.
	later := New(testKey(), nowAt("2026-05-23T12:06:00Z"))
	if _, err := later.Verify(token); err != ErrExpired {
		t.Fatalf("expected ErrExpired, got %v", err)
	}
}

func TestVerifyRejectsTokenSignedWithDifferentKey(t *testing.T) {
	signed := New(testKey(), nowAt("2026-05-23T12:00:00Z"))
	token := signed.Sign("nonce-123", 5*time.Minute)

	other := New([]byte("a-different-32-byte-key-bbbbbbbb"), nowAt("2026-05-23T12:00:00Z"))
	if _, err := other.Verify(token); err != ErrInvalid {
		t.Fatalf("expected ErrInvalid for foreign-key token, got %v", err)
	}
}

func TestVerifyRejectsMalformedToken(t *testing.T) {
	signer := New(testKey(), nowAt("2026-05-23T12:00:00Z"))
	for _, garbage := range []string{"", "no-dot", "not-base64.not-base64", ".."} {
		if _, err := signer.Verify(garbage); err == nil {
			t.Errorf("expected error for garbage token %q", garbage)
		}
	}
}

func flipChar(s string) string {
	if s == "A" {
		return "B"
	}
	return "A"
}

func testKey() []byte {
	return []byte("this-is-a-32-byte-test-key-aaaaa")
}

// nowAt returns a clock fixed at the given RFC3339 time, used to keep tests deterministic.
func nowAt(rfc3339 string) func() time.Time {
	t, err := time.Parse(time.RFC3339, rfc3339)
	if err != nil {
		panic(err)
	}
	return func() time.Time { return t }
}
