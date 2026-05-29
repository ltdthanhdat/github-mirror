package auth

import (
	"os"
	"testing"

	"github.com/dat-lt-amira/github-mirror/internal/store"
)

func TestPostgresUserStoreEnsureAdminUser(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	db, err := store.OpenPostgresDB(databaseURL)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer db.Close()

	if err := store.RunMigrations(t.Context(), db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	if _, err := db.Exec(`TRUNCATE TABLE sync_jobs, mirror_configs, users RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("reset test tables: %v", err)
	}

	userStore := NewPostgresUserStore(db)
	if err := userStore.EnsureAdminUser(t.Context(), "admin@example.com", "secret123", "Local Admin"); err != nil {
		t.Fatalf("ensure admin user: %v", err)
	}

	user, err := userStore.GetUserByEmail("admin@example.com")
	if err != nil {
		t.Fatalf("get admin user: %v", err)
	}
	if !user.IsAdmin {
		t.Fatalf("expected admin user, got %+v", user)
	}
	if err := user.CheckPassword("secret123"); err != nil {
		t.Fatalf("check admin password: %v", err)
	}

	if err := userStore.EnsureAdminUser(t.Context(), "admin@example.com", "new-secret", "Rotated Admin"); err != nil {
		t.Fatalf("rotate admin user: %v", err)
	}

	rotated, err := userStore.GetUserByEmail("admin@example.com")
	if err != nil {
		t.Fatalf("get rotated admin user: %v", err)
	}
	if rotated.FullName != "Rotated Admin" {
		t.Fatalf("expected rotated admin name, got %+v", rotated)
	}
	if err := rotated.CheckPassword("new-secret"); err != nil {
		t.Fatalf("check rotated admin password: %v", err)
	}
}
