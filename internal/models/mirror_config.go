package models

import (
	"time"
)

// MirrorConfig represents a mirror configuration.
type MirrorConfig struct {
	ID              uint64    `json:"id"`
	UserID          uint64    `json:"user_id"`
	Name            string    `json:"name"`
	SourceOwner     string    `json:"source_owner"`
	SourceRepo      string    `json:"source_repo"`
	SourceRepoURL   string    `json:"source_repo_url"`
	TargetOwner     string    `json:"target_owner"`
	TargetRepo      string    `json:"target_repo"`
	TargetRepoURL   string    `json:"target_repo_url"`
	SourceTokenEnc  string    `json:"-"` // encrypted, not exposed in JSON
	TargetTokenEnc  string    `json:"-"`
	WebhookSecretEnc string   `json:"-"`
	BranchPattern   string    `json:"branch_pattern"`
	SyncTags        bool      `json:"sync_tags"`
	SyncDeletes     bool      `json:"sync_deletes"`
	AllowForceUpdate bool     `json:"allow_force_update"`
	Enabled         bool      `json:"enabled"`
	LastSyncedAt    *time.Time `json:"last_synced_at,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}