package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/dat-lt-amira/github-mirror/internal/models"
	"github.com/dat-lt-amira/github-mirror/internal/schedule"
	"github.com/dat-lt-amira/github-mirror/internal/store"
)

const defaultSchedulerPollInterval = time.Minute

type scheduler struct {
	mirrorStore  store.MirrorConfigStore
	jobStore     store.SyncJobStore
	pollInterval time.Duration
	now          func() time.Time
}

func newScheduler(mirrorStore store.MirrorConfigStore, jobStore store.SyncJobStore) *scheduler {
	return &scheduler{
		mirrorStore:  mirrorStore,
		jobStore:     jobStore,
		pollInterval: schedulerPollInterval(),
		now:          time.Now,
	}
}

func schedulerPollInterval() time.Duration {
	raw := os.Getenv("SCHEDULER_POLL_INTERVAL")
	if raw == "" {
		return defaultSchedulerPollInterval
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		log.Printf("Scheduler: invalid SCHEDULER_POLL_INTERVAL %q, using default %s", raw, defaultSchedulerPollInterval)
		return defaultSchedulerPollInterval
	}
	return d
}

func (s *scheduler) Run(ctx context.Context) {
	s.runOnce(ctx)

	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runOnce(ctx)
		}
	}
}

func (s *scheduler) runOnce(ctx context.Context) {
	configs, err := s.mirrorStore.ListScheduledMirrorConfigs()
	if err != nil {
		log.Printf("Scheduler: list scheduled mirrors: %v", err)
		return
	}

	now := s.now().UTC()
	for _, cfg := range configs {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if err := s.processMirror(now, cfg); err != nil {
			log.Printf("Scheduler: process mirror %d: %v", cfg.ID, err)
		}
	}
}

func (s *scheduler) processMirror(now time.Time, cfg *models.MirrorConfig) error {
	expr, err := schedule.ParseCron(cfg.SyncSchedule)
	if err != nil {
		return err
	}

	dueAt, ok := expr.PreviousOrSame(now)
	if !ok {
		return nil
	}
	if cfg.LastScheduledAt != nil && !dueAt.After(cfg.LastScheduledAt.UTC()) {
		return nil
	}

	active, err := s.jobStore.HasActiveJobForMirror(cfg.ID)
	if err != nil {
		return err
	}

	cfg.LastScheduledAt = timePtr(dueAt)
	if !active {
		if err := s.jobStore.CreateJob(newFullSyncJob(cfg.ID)); err != nil {
			return err
		}
	}

	return s.mirrorStore.UpdateMirrorConfig(cfg)
}

func newFullSyncJob(mirrorConfigID uint64) *models.SyncJob {
	return &models.SyncJob{
		MirrorConfigID: mirrorConfigID,
		Ref:            "refs/heads/*",
		RefType:        "branch",
		BranchOrTag:    "*",
		AfterSHA:       "0000000",
		Deleted:        false,
		Status:         "queued",
		Attempts:       0,
		MaxAttempts:    3,
		CreatedAt:      time.Now(),
	}
}

func timePtr(ts time.Time) *time.Time {
	value := ts.UTC()
	return &value
}
