package main

import (
	"os"
	"testing"

	"github.com/dat-lt-amira/github-mirror/internal/models"
	"github.com/dat-lt-amira/github-mirror/internal/store"
)

func TestInitRuntimePersistsMirrorConfigsAcrossReconnect(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	resetRuntimeTestDB(t, databaseURL)

	t.Setenv("DATABASE_URL", databaseURL)
	t.Setenv("BASIC_AUTH_USERNAME", "admin@example.com")
	t.Setenv("BASIC_AUTH_PASSWORD", "secret123")

	runtimeOne, err := initRuntime()
	if err != nil {
		t.Fatalf("init runtime one: %v", err)
	}

	admin, err := runtimeOne.userStore.GetUserByEmail("admin@example.com")
	if err != nil {
		runtimeOne.db.Close()
		t.Fatalf("get admin user: %v", err)
	}

	cfg := &models.MirrorConfig{
		UserID:           admin.ID,
		Name:             "persisted-mirror",
		SourceOwner:      "source-org",
		SourceRepo:       "source-repo",
		SourceRepoURL:    "https://github.com/source-org/source-repo.git",
		TargetOwner:      "target-org",
		TargetRepo:       "target-repo",
		TargetRepoURL:    "https://github.com/target-org/target-repo.git",
		SourceTokenEnc:   "source-token",
		TargetTokenEnc:   "target-token",
		WebhookSecretEnc: "secret",
		BranchPattern:    "*",
		SyncTags:         true,
		AllowForceUpdate: true,
		Enabled:          true,
	}
	if err := runtimeOne.mirrorStore.CreateMirrorConfig(cfg); err != nil {
		runtimeOne.db.Close()
		t.Fatalf("create mirror config: %v", err)
	}
	runtimeOne.db.Close()

	runtimeTwo, err := initRuntime()
	if err != nil {
		t.Fatalf("init runtime two: %v", err)
	}
	defer runtimeTwo.db.Close()

	adminAfterRestart, err := runtimeTwo.userStore.GetUserByEmail("admin@example.com")
	if err != nil {
		t.Fatalf("get admin user after restart: %v", err)
	}

	configs, err := runtimeTwo.mirrorStore.ListMirrorConfigsByUser(adminAfterRestart.ID)
	if err != nil {
		t.Fatalf("list mirror configs after restart: %v", err)
	}
	if len(configs) != 1 || configs[0].Name != "persisted-mirror" {
		t.Fatalf("unexpected mirror configs after restart: %+v", configs)
	}
}

func resetRuntimeTestDB(t *testing.T, databaseURL string) {
	t.Helper()

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
}
