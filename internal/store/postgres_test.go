package store

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/dat-lt-amira/github-mirror/internal/models"
)

func TestPostgresMirrorConfigStoreCRUDAndPersistence(t *testing.T) {
	db := openTestPostgresDB(t)
	mirrorStore := NewPostgresMirrorConfigStore(db)
	userID := createTestUser(t, db, "owner@example.com")

	cfg := &models.MirrorConfig{
		UserID:           userID,
		Name:             "prod-mirror",
		SourceOwner:      "source-org",
		SourceRepo:       "source-repo",
		SourceRepoURL:    "https://github.com/source-org/source-repo.git",
		TargetOwner:      "target-org",
		TargetRepo:       "target-repo",
		TargetRepoURL:    "https://github.com/target-org/target-repo.git",
		SourceTokenEnc:   "source-token",
		TargetTokenEnc:   "target-token",
		WebhookSecretEnc: "secret",
		BranchPattern:    "main",
		SyncTags:         true,
		SyncDeletes:      true,
		AllowForceUpdate: true,
		SyncSchedule:     "*/10 * * * *",
		Enabled:          true,
	}

	if err := mirrorStore.CreateMirrorConfig(cfg); err != nil {
		t.Fatalf("create mirror config: %v", err)
	}

	reopened := openAdditionalTestPostgresDB(t)
	reopenedStore := NewPostgresMirrorConfigStore(reopened)

	loaded, err := reopenedStore.GetMirrorConfig(cfg.ID)
	if err != nil {
		t.Fatalf("get mirror config after reopen: %v", err)
	}
	if loaded.Name != cfg.Name || loaded.TargetRepoURL != cfg.TargetRepoURL {
		t.Fatalf("unexpected loaded mirror config: %+v", loaded)
	}
	if loaded.SyncSchedule != "*/10 * * * *" {
		t.Fatalf("expected sync schedule to persist, got %q", loaded.SyncSchedule)
	}

	loaded.Name = "prod-mirror-updated"
	loaded.BranchPattern = "release/*"
	scheduledAt := time.Date(2026, 6, 1, 1, 0, 0, 0, time.UTC)
	loaded.LastScheduledAt = &scheduledAt
	if err := reopenedStore.UpdateMirrorConfig(loaded); err != nil {
		t.Fatalf("update mirror config: %v", err)
	}

	configs, err := reopenedStore.ListMirrorConfigsByUser(cfg.UserID)
	if err != nil {
		t.Fatalf("list mirror configs: %v", err)
	}
	if len(configs) != 1 || configs[0].Name != "prod-mirror-updated" {
		t.Fatalf("unexpected mirror config list: %+v", configs)
	}
	if configs[0].LastScheduledAt == nil || !configs[0].LastScheduledAt.Equal(scheduledAt) {
		t.Fatalf("expected last scheduled at to persist, got %+v", configs[0].LastScheduledAt)
	}

	scheduledConfigs, err := reopenedStore.ListScheduledMirrorConfigs()
	if err != nil {
		t.Fatalf("list scheduled mirror configs: %v", err)
	}
	if len(scheduledConfigs) != 1 || scheduledConfigs[0].ID != cfg.ID {
		t.Fatalf("unexpected scheduled mirror configs: %+v", scheduledConfigs)
	}

	if err := reopenedStore.DeleteMirrorConfig(cfg.ID); err != nil {
		t.Fatalf("delete mirror config: %v", err)
	}
	if _, err := reopenedStore.GetMirrorConfig(cfg.ID); err == nil {
		t.Fatalf("expected mirror config to be deleted")
	}
}

func TestPostgresSyncJobStoreSharedClaiming(t *testing.T) {
	db := openTestPostgresDB(t)
	mirrorStore := NewPostgresMirrorConfigStore(db)
	userID := createTestUser(t, db, "owner@example.com")
	cfg := &models.MirrorConfig{
		UserID:           userID,
		Name:             "job-owner",
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
	if err := mirrorStore.CreateMirrorConfig(cfg); err != nil {
		t.Fatalf("create mirror config: %v", err)
	}
	jobStore := NewPostgresSyncJobStore(db)

	job := &models.SyncJob{
		MirrorConfigID: cfg.ID,
		Ref:            "refs/heads/main",
		RefType:        "branch",
		BranchOrTag:    "main",
		AfterSHA:       "abc123",
		MaxAttempts:    3,
	}
	if err := jobStore.CreateJob(job); err != nil {
		t.Fatalf("create job: %v", err)
	}

	reopened := openAdditionalTestPostgresDB(t)
	reopenedStore := NewPostgresSyncJobStore(reopened)

	claimed, err := reopenedStore.ClaimNextJob()
	if err != nil {
		t.Fatalf("claim job: %v", err)
	}
	if claimed == nil {
		t.Fatalf("expected claimed job")
	}
	if claimed.ID != job.ID || claimed.Status != "running" || claimed.Attempts != 1 {
		t.Fatalf("unexpected claimed job: %+v", claimed)
	}

	claimed.Status = "succeeded"
	if err := reopenedStore.UpdateJob(claimed); err != nil {
		t.Fatalf("update claimed job: %v", err)
	}

	loaded, err := jobStore.GetJob(job.ID)
	if err != nil {
		t.Fatalf("get updated job: %v", err)
	}
	if loaded.Status != "succeeded" {
		t.Fatalf("expected succeeded status, got %+v", loaded)
	}
	active, err := jobStore.HasActiveJobForMirror(cfg.ID)
	if err != nil {
		t.Fatalf("has active job after completion: %v", err)
	}
	if active {
		t.Fatalf("expected no active job after completion")
	}

	next, err := jobStore.ClaimNextJob()
	if err != nil {
		t.Fatalf("claim next job: %v", err)
	}
	if next != nil {
		t.Fatalf("expected no further jobs, got %+v", next)
	}
}

func TestPostgresSyncJobStoreReportsActiveJobs(t *testing.T) {
	db := openTestPostgresDB(t)
	mirrorStore := NewPostgresMirrorConfigStore(db)
	userID := createTestUser(t, db, "owner@example.com")
	cfg := &models.MirrorConfig{
		UserID:           userID,
		Name:             "job-owner",
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
		SyncSchedule:     "*/5 * * * *",
		Enabled:          true,
	}
	if err := mirrorStore.CreateMirrorConfig(cfg); err != nil {
		t.Fatalf("create mirror config: %v", err)
	}
	jobStore := NewPostgresSyncJobStore(db)

	job := &models.SyncJob{
		MirrorConfigID: cfg.ID,
		Ref:            "refs/heads/*",
		RefType:        "branch",
		BranchOrTag:    "*",
		Status:         "queued",
		MaxAttempts:    3,
	}
	if err := jobStore.CreateJob(job); err != nil {
		t.Fatalf("create job: %v", err)
	}

	active, err := jobStore.HasActiveJobForMirror(cfg.ID)
	if err != nil {
		t.Fatalf("has active job: %v", err)
	}
	if !active {
		t.Fatalf("expected queued job to count as active")
	}
}

func openTestPostgresDB(t *testing.T) *sql.DB {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	db, err := OpenPostgresDB(databaseURL)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	if err := RunMigrations(t.Context(), db); err != nil {
		db.Close()
		t.Fatalf("run migrations: %v", err)
	}
	resetTestTables(t, db)
	t.Cleanup(func() {
		resetTestTables(t, db)
		db.Close()
	})
	return db
}

func openAdditionalTestPostgresDB(t *testing.T) *sql.DB {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	reopened, err := OpenPostgresDB(databaseURL)
	if err != nil {
		t.Fatalf("reopen postgres: %v", err)
	}
	t.Cleanup(func() {
		reopened.Close()
	})
	return reopened
}

func resetTestTables(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`TRUNCATE TABLE sync_jobs, mirror_configs, users RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("reset test tables: %v", err)
	}
}

func createTestUser(t *testing.T, db *sql.DB, email string) uint64 {
	t.Helper()
	var id uint64
	if err := db.QueryRow(`
		INSERT INTO users (email, password_hash, full_name, is_admin)
		VALUES ($1, $2, $3, FALSE)
		RETURNING id
	`, email, "hashed-password", "Test User").Scan(&id); err != nil {
		t.Fatalf("create test user: %v", err)
	}
	return id
}
