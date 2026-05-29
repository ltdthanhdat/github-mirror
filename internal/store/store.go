package store

import (
	"github.com/dat-lt-amira/github-mirror/internal/models"
)

// MirrorConfigStore defines operations for mirror configuration storage.
type MirrorConfigStore interface {
	CreateMirrorConfig(cfg *models.MirrorConfig) error
	GetMirrorConfig(id uint64) (*models.MirrorConfig, error)
	ListMirrorConfigsByUser(userID uint64) ([]*models.MirrorConfig, error)
	UpdateMirrorConfig(cfg *models.MirrorConfig) error
	DeleteMirrorConfig(id uint64) error
}

// SyncJobStore defines operations for sync job storage.
type SyncJobStore interface {
	CreateJob(job *models.SyncJob) error
	GetJob(id uint64) (*models.SyncJob, error)
	ListJobsByMirrorConfig(mirrorConfigID uint64) ([]*models.SyncJob, error)
	ClaimNextJob() (*models.SyncJob, error)
	UpdateJob(job *models.SyncJob) error
}
