package main

import (
	"context"
	"testing"
	"time"

	"github.com/dat-lt-amira/github-mirror/internal/models"
	"github.com/dat-lt-amira/github-mirror/internal/store"
)

func TestSchedulerEnqueuesDueMirrorAndRecordsScheduleState(t *testing.T) {
	mirrorStore := store.NewInMemoryMirrorConfigStore()
	jobStore := store.NewInMemorySyncJobStore()
	cfg := &models.MirrorConfig{
		ID:               1,
		UserID:           1,
		Name:             "scheduled",
		SourceOwner:      "source",
		SourceRepo:       "repo",
		SourceRepoURL:    "https://github.com/source/repo.git",
		TargetOwner:      "target",
		TargetRepo:       "repo",
		TargetRepoURL:    "https://github.com/target/repo.git",
		SourceTokenEnc:   "source-token",
		TargetTokenEnc:   "target-token",
		WebhookSecretEnc: "secret",
		BranchPattern:    "*",
		SyncSchedule:     "*/10 * * * *",
		SyncTags:         true,
		AllowForceUpdate: true,
		Enabled:          true,
	}
	if err := mirrorStore.CreateMirrorConfig(cfg); err != nil {
		t.Fatalf("create mirror config: %v", err)
	}

	s := &scheduler{
		mirrorStore:  mirrorStore,
		jobStore:     jobStore,
		pollInterval: time.Minute,
		now: func() time.Time {
			return time.Date(2026, 6, 1, 10, 20, 0, 0, time.UTC)
		},
	}

	s.runOnce(context.Background())

	jobs, err := jobStore.ListJobsByMirrorConfig(cfg.ID)
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected one scheduled job, got %d", len(jobs))
	}
	loaded, err := mirrorStore.GetMirrorConfig(cfg.ID)
	if err != nil {
		t.Fatalf("get mirror config: %v", err)
	}
	wantDue := time.Date(2026, 6, 1, 10, 20, 0, 0, time.UTC)
	if loaded.LastScheduledAt == nil || !loaded.LastScheduledAt.Equal(wantDue) {
		t.Fatalf("expected last scheduled at %s, got %+v", wantDue, loaded.LastScheduledAt)
	}

	s.runOnce(context.Background())
	jobs, err = jobStore.ListJobsByMirrorConfig(cfg.ID)
	if err != nil {
		t.Fatalf("list jobs after rerun: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected rerun not to enqueue duplicate job, got %d", len(jobs))
	}
}

func TestSchedulerSkipsEnqueueWhenMirrorAlreadyHasActiveJob(t *testing.T) {
	mirrorStore := store.NewInMemoryMirrorConfigStore()
	jobStore := store.NewInMemorySyncJobStore()
	cfg := &models.MirrorConfig{
		UserID:           1,
		Name:             "scheduled",
		SourceOwner:      "source",
		SourceRepo:       "repo",
		SourceRepoURL:    "https://github.com/source/repo.git",
		TargetOwner:      "target",
		TargetRepo:       "repo",
		TargetRepoURL:    "https://github.com/target/repo.git",
		SourceTokenEnc:   "source-token",
		TargetTokenEnc:   "target-token",
		WebhookSecretEnc: "secret",
		BranchPattern:    "*",
		SyncSchedule:     "*/10 * * * *",
		SyncTags:         true,
		AllowForceUpdate: true,
		Enabled:          true,
	}
	if err := mirrorStore.CreateMirrorConfig(cfg); err != nil {
		t.Fatalf("create mirror config: %v", err)
	}
	if err := jobStore.CreateJob(newFullSyncJob(cfg.ID)); err != nil {
		t.Fatalf("create existing active job: %v", err)
	}

	s := &scheduler{
		mirrorStore:  mirrorStore,
		jobStore:     jobStore,
		pollInterval: time.Minute,
		now: func() time.Time {
			return time.Date(2026, 6, 1, 10, 20, 0, 0, time.UTC)
		},
	}

	s.runOnce(context.Background())

	jobs, err := jobStore.ListJobsByMirrorConfig(cfg.ID)
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected active queued job to suppress duplicate, got %d jobs", len(jobs))
	}
	loaded, err := mirrorStore.GetMirrorConfig(cfg.ID)
	if err != nil {
		t.Fatalf("get mirror config: %v", err)
	}
	if loaded.LastScheduledAt == nil {
		t.Fatalf("expected scheduler to record due window even when job already active")
	}
}
