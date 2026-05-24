package repository_test

import (
	"context"
	"testing"

	"github.com/HenronenGIT/thought-box/apps/api-go/internal/repository"
	"github.com/HenronenGIT/thought-box/apps/api-go/internal/testdb"
)

func TestUpsertByGoogleSubCreatesNewUser(t *testing.T) {
	pool := testdb.Pool(t)
	testdb.Truncate(t, pool, "users")
	users := repository.NewUserRepository(pool)

	got, err := users.UpsertByGoogleSub(context.Background(), "google-sub-001", "new@example.com", "New User")
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if got.Email != "new@example.com" {
		t.Errorf("email = %q", got.Email)
	}
	if got.GoogleSub != "google-sub-001" {
		t.Errorf("sub = %q", got.GoogleSub)
	}
	if got.DisplayName != "New User" {
		t.Errorf("display name = %q", got.DisplayName)
	}
	if got.ID.String() == "" {
		t.Error("ID was zero")
	}
}

func TestUpsertByGoogleSubIsIdempotentOnSub(t *testing.T) {
	pool := testdb.Pool(t)
	testdb.Truncate(t, pool, "users")
	users := repository.NewUserRepository(pool)

	first, _ := users.UpsertByGoogleSub(context.Background(), "google-sub-002", "old@example.com", "Old Name")
	second, err := users.UpsertByGoogleSub(context.Background(), "google-sub-002", "new@example.com", "New Name")
	if err != nil {
		t.Fatalf("Upsert second: %v", err)
	}
	if second.ID != first.ID {
		t.Errorf("ID changed across upserts: %v -> %v", first.ID, second.ID)
	}
	if second.Email != "new@example.com" {
		t.Errorf("email did not update: %q", second.Email)
	}
	if second.DisplayName != "New Name" {
		t.Errorf("display name did not update: %q", second.DisplayName)
	}
}

func TestUpsertByGoogleSubBindsToSeededUserOnEmailMatch(t *testing.T) {
	pool := testdb.Pool(t)
	testdb.Truncate(t, pool, "users")
	users := repository.NewUserRepository(pool)
	// Re-seed the row that 00001_users + 00005_auth normally produce.
	if _, err := pool.Exec(context.Background(),
		`insert into users (id, email) values ('00000000-0000-4000-8000-000000000001', 'admin@example.com')`); err != nil {
		t.Fatalf("seed: %v", err)
	}

	got, err := users.UpsertByGoogleSub(context.Background(), "google-sub-admin", "admin@example.com", "Admin")
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if got.ID.String() != "00000000-0000-4000-8000-000000000001" {
		t.Fatalf("expected seeded UUID preserved, got %v", got.ID)
	}
	if got.GoogleSub != "google-sub-admin" {
		t.Errorf("sub not set on existing row: %q", got.GoogleSub)
	}
}

func TestFindByIDReturnsNilForUnknown(t *testing.T) {
	pool := testdb.Pool(t)
	testdb.Truncate(t, pool, "users")
	users := repository.NewUserRepository(pool)
	created, _ := users.UpsertByGoogleSub(context.Background(), "sub-find", "find@example.com", "Find")

	found, err := users.FindByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found == nil || found.ID != created.ID {
		t.Errorf("FindByID round-trip failed: %+v", found)
	}
}
