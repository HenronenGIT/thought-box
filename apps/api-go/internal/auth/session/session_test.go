package session_test

import (
	"context"
	"testing"
	"time"

	"github.com/HenronenGIT/thought-box/apps/api-go/internal/auth/session"
	"github.com/HenronenGIT/thought-box/apps/api-go/internal/testdb"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestIssueThenLookupReturnsUserID(t *testing.T) {
	pool := freshPool(t)
	store := session.NewStore(pool)
	userID := seedUser(t, pool, "issuer@example.com")

	ctx := context.Background()
	token, err := store.Issue(ctx, userID, 30*24*time.Hour)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if token == "" {
		t.Fatal("Issue returned empty token")
	}

	got, err := store.Lookup(ctx, token)
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if got != userID {
		t.Errorf("user id = %v, want %v", got, userID)
	}
}

func TestLookupUnknownTokenReturnsNotFound(t *testing.T) {
	store := session.NewStore(freshPool(t))
	_, err := store.Lookup(context.Background(), "this-token-was-never-issued")
	if err != session.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestLookupExpiredTokenReturnsExpired(t *testing.T) {
	pool := freshPool(t)
	store := session.NewStoreForTest(pool, 1*time.Millisecond, time.Now)
	userID := seedUser(t, pool, "expired@example.com")

	ctx := context.Background()
	token, err := store.Issue(ctx, userID, 1*time.Millisecond)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	time.Sleep(10 * time.Millisecond)

	_, err = store.Lookup(ctx, token)
	if err != session.ErrExpired {
		t.Fatalf("expected ErrExpired, got %v", err)
	}
}

func TestLookupSlidesExpiry(t *testing.T) {
	pool := freshPool(t)
	now := time.Now()
	clock := &fakeClock{t: now}
	// 1-hour TTL so we can advance time without crossing initial expiry.
	store := session.NewStoreForTest(pool, time.Hour, clock.Now)
	userID := seedUser(t, pool, "slide@example.com")

	ctx := context.Background()
	token, _ := store.Issue(ctx, userID, time.Hour)
	clock.Advance(30 * time.Minute)

	if _, err := store.Lookup(ctx, token); err != nil {
		t.Fatalf("Lookup mid-life: %v", err)
	}
	// Advance 50 more minutes — total 80 min from issue, but Lookup at min 30
	// should have slid expiry to min 90. So this should still succeed.
	clock.Advance(50 * time.Minute)
	if _, err := store.Lookup(ctx, token); err != nil {
		t.Fatalf("Lookup after slide: %v", err)
	}
}

func TestRevokeInvalidatesToken(t *testing.T) {
	pool := freshPool(t)
	store := session.NewStore(pool)
	userID := seedUser(t, pool, "revoke@example.com")

	ctx := context.Background()
	token, _ := store.Issue(ctx, userID, time.Hour)
	if err := store.Revoke(ctx, token); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if _, err := store.Lookup(ctx, token); err != session.ErrNotFound {
		t.Fatalf("expected ErrNotFound after revoke, got %v", err)
	}
}

func TestRevokeAllAffectsOnlyTargetUser(t *testing.T) {
	pool := freshPool(t)
	store := session.NewStore(pool)
	alice := seedUser(t, pool, "alice@example.com")
	bob := seedUser(t, pool, "bob@example.com")

	ctx := context.Background()
	aliceA, _ := store.Issue(ctx, alice, time.Hour)
	aliceB, _ := store.Issue(ctx, alice, time.Hour)
	bobTok, _ := store.Issue(ctx, bob, time.Hour)

	if err := store.RevokeAll(ctx, alice); err != nil {
		t.Fatalf("RevokeAll: %v", err)
	}
	if _, err := store.Lookup(ctx, aliceA); err != session.ErrNotFound {
		t.Errorf("alice token A still valid: %v", err)
	}
	if _, err := store.Lookup(ctx, aliceB); err != session.ErrNotFound {
		t.Errorf("alice token B still valid: %v", err)
	}
	if _, err := store.Lookup(ctx, bobTok); err != nil {
		t.Errorf("bob's session was wrongly invalidated: %v", err)
	}
}

func TestIssueStoresHashNotRawToken(t *testing.T) {
	pool := freshPool(t)
	store := session.NewStore(pool)
	userID := seedUser(t, pool, "hash@example.com")

	token, _ := store.Issue(context.Background(), userID, time.Hour)

	// The raw token must never appear in the DB; only its hash should.
	var count int
	err := pool.QueryRow(context.Background(),
		"select count(*) from sessions where token_hash::text = $1", token).Scan(&count)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 0 {
		t.Errorf("raw token was stored verbatim; want hashed")
	}
}

type fakeClock struct{ t time.Time }

func (c *fakeClock) Now() time.Time          { return c.t }
func (c *fakeClock) Advance(d time.Duration) { c.t = c.t.Add(d) }

func freshPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := testdb.Pool(t)
	testdb.Truncate(t, pool, "sessions")
	return pool
}

func seedUser(t *testing.T, pool *pgxpool.Pool, email string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if _, err := pool.Exec(context.Background(),
		`insert into users (id, email) values ($1, $2)`, id, email); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "delete from users where id = $1", id)
	})
	return id
}
