package allowlist_test

import (
	"context"
	"testing"

	"github.com/HenronenGIT/thought-box/apps/api-go/internal/auth/allowlist"
	"github.com/HenronenGIT/thought-box/apps/api-go/internal/testdb"
)

func TestIsAllowedReturnsTrueForListedEmail(t *testing.T) {
	pool := testdb.Pool(t)
	testdb.Truncate(t, pool, "allowed_emails")
	if _, err := pool.Exec(context.Background(),
		"insert into allowed_emails (email) values ('listed@example.com')"); err != nil {
		t.Fatalf("seed: %v", err)
	}

	gate := allowlist.New(pool)
	ok, err := gate.IsAllowed(context.Background(), "listed@example.com")
	if err != nil {
		t.Fatalf("IsAllowed: %v", err)
	}
	if !ok {
		t.Error("expected listed email to be allowed")
	}
}

func TestIsAllowedReturnsFalseForUnlistedEmail(t *testing.T) {
	pool := testdb.Pool(t)
	testdb.Truncate(t, pool, "allowed_emails")
	gate := allowlist.New(pool)

	ok, err := gate.IsAllowed(context.Background(), "stranger@example.com")
	if err != nil {
		t.Fatalf("IsAllowed: %v", err)
	}
	if ok {
		t.Error("expected unlisted email to be rejected")
	}
}

func TestIsAllowedIsCaseInsensitive(t *testing.T) {
	pool := testdb.Pool(t)
	testdb.Truncate(t, pool, "allowed_emails")
	if _, err := pool.Exec(context.Background(),
		"insert into allowed_emails (email) values ('user@example.com')"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	gate := allowlist.New(pool)

	ok, err := gate.IsAllowed(context.Background(), "USER@Example.COM")
	if err != nil {
		t.Fatalf("IsAllowed: %v", err)
	}
	if !ok {
		t.Error("expected case-insensitive match")
	}
}
