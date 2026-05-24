// Package state issues and verifies short-lived signed tokens used to defend
// the OAuth redirect dance against CSRF. Each token carries a caller-chosen
// nonce plus an expiry; tampering or expiry invalidates verification.
package state

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Signer struct {
	key []byte
	now func() time.Time
}

func New(key []byte, now func() time.Time) *Signer {
	if now == nil {
		now = time.Now
	}
	return &Signer{key: key, now: now}
}

// Sign returns a token carrying nonce that verifies until ttl from now.
func (s *Signer) Sign(nonce string, ttl time.Duration) string {
	expiry := s.now().Add(ttl).Unix()
	payload := encodePayload(nonce, expiry)
	sig := s.mac(payload)
	return payload + "." + base64.RawURLEncoding.EncodeToString(sig)
}

var (
	ErrInvalid = errors.New("state: invalid token")
	ErrExpired = errors.New("state: expired token")
)

// Verify returns the nonce embedded in token, or an error if the token is
// malformed, signature-invalid, or expired.
func (s *Signer) Verify(token string) (string, error) {
	payload, sigB64, ok := strings.Cut(token, ".")
	if !ok {
		return "", ErrInvalid
	}
	sig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return "", ErrInvalid
	}
	expected := s.mac(payload)
	if !hmac.Equal(sig, expected) {
		return "", ErrInvalid
	}
	nonce, expiry, err := decodePayload(payload)
	if err != nil {
		return "", ErrInvalid
	}
	if s.now().Unix() >= expiry {
		return "", ErrExpired
	}
	return nonce, nil
}

func (s *Signer) mac(payload string) []byte {
	h := hmac.New(sha256.New, s.key)
	h.Write([]byte(payload))
	return h.Sum(nil)
}

func encodePayload(nonce string, expiry int64) string {
	return base64.RawURLEncoding.EncodeToString([]byte(nonce)) + "|" + strconv.FormatInt(expiry, 10)
}

func decodePayload(payload string) (string, int64, error) {
	nonceB64, expiryStr, ok := strings.Cut(payload, "|")
	if !ok {
		return "", 0, fmt.Errorf("malformed payload")
	}
	nonce, err := base64.RawURLEncoding.DecodeString(nonceB64)
	if err != nil {
		return "", 0, err
	}
	expiry, err := strconv.ParseInt(expiryStr, 10, 64)
	if err != nil {
		return "", 0, err
	}
	return string(nonce), expiry, nil
}
