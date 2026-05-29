package models

import "time"

// SyncJob represents a synchronization job.
type SyncJob struct {
	ID               uint64    `json:"id"`
	MirrorConfigID   uint64    `json:"mirror_config_id"`
	GitHubDeliveryID string    `json:"github_delivery_id,omitempty"`
	Ref              string    `json:"ref"`
	RefType          string    `json:"ref_type"`
	BranchOrTag      string    `json:"branch_or_tag"`
	AfterSHA         string    `json:"after_sha,omitempty"`
	Deleted          bool      `json:"deleted"`
	Status           string    `json:"status"`
	Attempts         int       `json:"attempts"`
	MaxAttempts      int       `json:"max_attempts"`
	LastError        string    `json:"last_error,omitempty"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	FinishedAt       *time.Time `json:"finished_at,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}
