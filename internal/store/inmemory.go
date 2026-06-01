package store

import (
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/dat-lt-amira/github-mirror/internal/models"
)

// InMemoryMirrorConfigStore implements MirrorConfigStore with an in-memory map.
type InMemoryMirrorConfigStore struct {
	mu      sync.RWMutex
	configs map[uint64]*models.MirrorConfig
	nextID  uint64
}

// NewInMemoryMirrorConfigStore creates a new in-memory store.
func NewInMemoryMirrorConfigStore() *InMemoryMirrorConfigStore {
	return &InMemoryMirrorConfigStore{
		configs: make(map[uint64]*models.MirrorConfig),
		nextID:  1,
	}
}

func (s *InMemoryMirrorConfigStore) CreateMirrorConfig(cfg *models.MirrorConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg.ID = s.nextID
	s.nextID++
	cfg.CreatedAt = time.Now()
	cfg.UpdatedAt = time.Now()
	s.configs[cfg.ID] = cfg
	return nil
}

func (s *InMemoryMirrorConfigStore) GetMirrorConfig(id uint64) (*models.MirrorConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cfg, ok := s.configs[id]
	if !ok {
		return nil, errors.New("mirror config not found")
	}
	return cfg, nil
}

func (s *InMemoryMirrorConfigStore) ListMirrorConfigsByUser(userID uint64) ([]*models.MirrorConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*models.MirrorConfig
	for _, cfg := range s.configs {
		if cfg.UserID == userID {
			result = append(result, cfg)
		}
	}
	return result, nil
}

func (s *InMemoryMirrorConfigStore) ListScheduledMirrorConfigs() ([]*models.MirrorConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*models.MirrorConfig
	for _, cfg := range s.configs {
		if cfg.Enabled && cfg.SyncSchedule != "" {
			result = append(result, cfg)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result, nil
}

func (s *InMemoryMirrorConfigStore) UpdateMirrorConfig(cfg *models.MirrorConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.configs[cfg.ID]; !ok {
		return errors.New("mirror config not found")
	}
	cfg.UpdatedAt = time.Now()
	s.configs[cfg.ID] = cfg
	return nil
}

func (s *InMemoryMirrorConfigStore) DeleteMirrorConfig(id uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.configs[id]; !ok {
		return errors.New("mirror config not found")
	}
	delete(s.configs, id)
	return nil
}

// InMemorySyncJobStore implements SyncJobStore with an in-memory map.
type InMemorySyncJobStore struct {
	mu     sync.RWMutex
	jobs   map[uint64]*models.SyncJob
	nextID uint64
}

// NewInMemorySyncJobStore creates a new in-memory sync job store.
func NewInMemorySyncJobStore() *InMemorySyncJobStore {
	return &InMemorySyncJobStore{
		jobs:   make(map[uint64]*models.SyncJob),
		nextID: 1,
	}
}

func (s *InMemorySyncJobStore) CreateJob(job *models.SyncJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job.ID = s.nextID
	s.nextID++
	job.CreatedAt = time.Now()
	job.Status = "queued"
	s.jobs[job.ID] = job
	return nil
}

func (s *InMemorySyncJobStore) GetJob(id uint64) (*models.SyncJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, ok := s.jobs[id]
	if !ok {
		return nil, errors.New("job not found")
	}
	return job, nil
}

func (s *InMemorySyncJobStore) ListJobsByMirrorConfig(mirrorConfigID uint64) ([]*models.SyncJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*models.SyncJob
	for _, job := range s.jobs {
		if job.MirrorConfigID == mirrorConfigID {
			result = append(result, job)
		}
	}
	return result, nil
}

func (s *InMemorySyncJobStore) HasActiveJobForMirror(mirrorConfigID uint64) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, job := range s.jobs {
		if job.MirrorConfigID == mirrorConfigID && (job.Status == "queued" || job.Status == "running") {
			return true, nil
		}
	}
	return false, nil
}

func (s *InMemorySyncJobStore) ClaimNextJob() (*models.SyncJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, job := range s.jobs {
		if job.Status == "queued" && job.Attempts < job.MaxAttempts {
			job.Status = "running"
			job.Attempts++
			now := time.Now()
			job.StartedAt = &now
			return job, nil
		}
		if job.Status == "retrying" && job.Attempts < job.MaxAttempts {
			job.Status = "running"
			job.Attempts++
			now := time.Now()
			job.StartedAt = &now
			return job, nil
		}
	}
	return nil, nil
}

func (s *InMemorySyncJobStore) UpdateJob(job *models.SyncJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.jobs[job.ID]; !ok {
		return errors.New("job not found")
	}
	s.jobs[job.ID] = job
	return nil
}
