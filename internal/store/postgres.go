package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"time"

	"github.com/dat-lt-amira/github-mirror/internal/models"
	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// OpenPostgresDB opens and validates a PostgreSQL connection.
func OpenPostgresDB(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return db, nil
}

// RunMigrations applies embedded SQL migrations exactly once in lexical order.
func RunMigrations(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, int64(20260601)); err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}
	defer db.ExecContext(context.Background(), `SELECT pg_advisory_unlock($1)`, int64(20260601))

	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := fs.Glob(migrationFS, "migrations/*.sql")
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	sort.Strings(entries)

	for _, entry := range entries {
		var alreadyApplied bool
		if err := db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE filename = $1)`, entry).Scan(&alreadyApplied); err != nil {
			return fmt.Errorf("check migration %s: %w", entry, err)
		}
		if alreadyApplied {
			continue
		}

		contents, err := migrationFS.ReadFile(entry)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry, err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", entry, err)
		}

		if _, err := tx.ExecContext(ctx, string(contents)); err != nil {
			tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", entry, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (filename) VALUES ($1)`, entry); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %s: %w", entry, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", entry, err)
		}
	}

	return nil
}

type PostgresMirrorConfigStore struct {
	db *sql.DB
}

func NewPostgresMirrorConfigStore(db *sql.DB) *PostgresMirrorConfigStore {
	return &PostgresMirrorConfigStore{db: db}
}

func (s *PostgresMirrorConfigStore) CreateMirrorConfig(cfg *models.MirrorConfig) error {
	if cfg.BranchPattern == "" {
		cfg.BranchPattern = "*"
	}

	return s.db.QueryRow(`
		INSERT INTO mirror_configs (
			user_id, name,
			source_owner, source_repo, source_repo_url,
			target_owner, target_repo, target_repo_url,
			source_token_enc, target_token_enc, webhook_secret_enc,
			branch_pattern, sync_tags, sync_deletes, allow_force_update,
			sync_schedule, enabled, last_synced_at, last_scheduled_at
		) VALUES (
			$1, $2,
			$3, $4, $5,
			$6, $7, $8,
			$9, $10, $11,
			$12, $13, $14, $15,
			$16, $17, $18, $19
		)
		RETURNING id, created_at, updated_at
	`,
		cfg.UserID, cfg.Name,
		cfg.SourceOwner, cfg.SourceRepo, cfg.SourceRepoURL,
		cfg.TargetOwner, cfg.TargetRepo, cfg.TargetRepoURL,
		cfg.SourceTokenEnc, cfg.TargetTokenEnc, cfg.WebhookSecretEnc,
		cfg.BranchPattern, cfg.SyncTags, cfg.SyncDeletes, cfg.AllowForceUpdate,
		nullableString(cfg.SyncSchedule), cfg.Enabled, cfg.LastSyncedAt, cfg.LastScheduledAt,
	).Scan(&cfg.ID, &cfg.CreatedAt, &cfg.UpdatedAt)
}

func (s *PostgresMirrorConfigStore) GetMirrorConfig(id uint64) (*models.MirrorConfig, error) {
	row := s.db.QueryRow(`
		SELECT
			id, user_id, name,
			source_owner, source_repo, source_repo_url,
			target_owner, target_repo, target_repo_url,
			source_token_enc, target_token_enc, webhook_secret_enc,
			branch_pattern, sync_tags, sync_deletes, allow_force_update,
			sync_schedule, enabled, last_synced_at, last_scheduled_at, created_at, updated_at
		FROM mirror_configs
		WHERE id = $1
	`, id)

	cfg, err := scanMirrorConfig(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("mirror config not found")
	}
	return cfg, err
}

func (s *PostgresMirrorConfigStore) ListMirrorConfigsByUser(userID uint64) ([]*models.MirrorConfig, error) {
	rows, err := s.db.Query(`
		SELECT
			id, user_id, name,
			source_owner, source_repo, source_repo_url,
			target_owner, target_repo, target_repo_url,
			source_token_enc, target_token_enc, webhook_secret_enc,
			branch_pattern, sync_tags, sync_deletes, allow_force_update,
			sync_schedule, enabled, last_synced_at, last_scheduled_at, created_at, updated_at
		FROM mirror_configs
		WHERE user_id = $1
		ORDER BY id
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*models.MirrorConfig
	for rows.Next() {
		cfg, err := scanMirrorConfig(rows)
		if err != nil {
			return nil, err
		}
		configs = append(configs, cfg)
	}
	return configs, rows.Err()
}

func (s *PostgresMirrorConfigStore) ListScheduledMirrorConfigs() ([]*models.MirrorConfig, error) {
	rows, err := s.db.Query(`
		SELECT
			id, user_id, name,
			source_owner, source_repo, source_repo_url,
			target_owner, target_repo, target_repo_url,
			source_token_enc, target_token_enc, webhook_secret_enc,
			branch_pattern, sync_tags, sync_deletes, allow_force_update,
			sync_schedule, enabled, last_synced_at, last_scheduled_at, created_at, updated_at
		FROM mirror_configs
		WHERE enabled = TRUE
		  AND sync_schedule IS NOT NULL
		  AND sync_schedule <> ''
		ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*models.MirrorConfig
	for rows.Next() {
		cfg, err := scanMirrorConfig(rows)
		if err != nil {
			return nil, err
		}
		configs = append(configs, cfg)
	}
	return configs, rows.Err()
}

func (s *PostgresMirrorConfigStore) UpdateMirrorConfig(cfg *models.MirrorConfig) error {
	row := s.db.QueryRow(`
		UPDATE mirror_configs
		SET
			user_id = $2,
			name = $3,
			source_owner = $4,
			source_repo = $5,
			source_repo_url = $6,
			target_owner = $7,
			target_repo = $8,
			target_repo_url = $9,
			source_token_enc = $10,
			target_token_enc = $11,
			webhook_secret_enc = $12,
			branch_pattern = $13,
			sync_tags = $14,
			sync_deletes = $15,
			allow_force_update = $16,
			sync_schedule = $17,
			enabled = $18,
			last_synced_at = $19,
			last_scheduled_at = $20,
			updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`,
		cfg.ID,
		cfg.UserID,
		cfg.Name,
		cfg.SourceOwner,
		cfg.SourceRepo,
		cfg.SourceRepoURL,
		cfg.TargetOwner,
		cfg.TargetRepo,
		cfg.TargetRepoURL,
		cfg.SourceTokenEnc,
		cfg.TargetTokenEnc,
		cfg.WebhookSecretEnc,
		cfg.BranchPattern,
		cfg.SyncTags,
		cfg.SyncDeletes,
		cfg.AllowForceUpdate,
		nullableString(cfg.SyncSchedule),
		cfg.Enabled,
		cfg.LastSyncedAt,
		cfg.LastScheduledAt,
	)
	if err := row.Scan(&cfg.UpdatedAt); errors.Is(err, sql.ErrNoRows) {
		return errors.New("mirror config not found")
	} else {
		return err
	}
}

func (s *PostgresMirrorConfigStore) DeleteMirrorConfig(id uint64) error {
	result, err := s.db.Exec(`DELETE FROM mirror_configs WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("mirror config not found")
	}
	return nil
}

type PostgresSyncJobStore struct {
	db *sql.DB
}

func NewPostgresSyncJobStore(db *sql.DB) *PostgresSyncJobStore {
	return &PostgresSyncJobStore{db: db}
}

func (s *PostgresSyncJobStore) CreateJob(job *models.SyncJob) error {
	if job.Status == "" {
		job.Status = "queued"
	}
	if job.MaxAttempts == 0 {
		job.MaxAttempts = 3
	}

	return s.db.QueryRow(`
		INSERT INTO sync_jobs (
			mirror_config_id, github_delivery_id, ref, ref_type, branch_or_tag,
			after_sha, deleted, status, attempts, max_attempts, last_error,
			started_at, finished_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11,
			$12, $13
		)
		RETURNING id, status, created_at
	`,
		job.MirrorConfigID, nullableString(job.GitHubDeliveryID), job.Ref, job.RefType, job.BranchOrTag,
		job.AfterSHA, job.Deleted, job.Status, job.Attempts, job.MaxAttempts, job.LastError,
		job.StartedAt, job.FinishedAt,
	).Scan(&job.ID, &job.Status, &job.CreatedAt)
}

func (s *PostgresSyncJobStore) GetJob(id uint64) (*models.SyncJob, error) {
	row := s.db.QueryRow(`
		SELECT
			id, mirror_config_id, github_delivery_id, ref, ref_type, branch_or_tag,
			after_sha, deleted, status, attempts, max_attempts, last_error,
			started_at, finished_at, created_at
		FROM sync_jobs
		WHERE id = $1
	`, id)

	job, err := scanSyncJob(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("job not found")
	}
	return job, err
}

func (s *PostgresSyncJobStore) ListJobsByMirrorConfig(mirrorConfigID uint64) ([]*models.SyncJob, error) {
	rows, err := s.db.Query(`
		SELECT
			id, mirror_config_id, github_delivery_id, ref, ref_type, branch_or_tag,
			after_sha, deleted, status, attempts, max_attempts, last_error,
			started_at, finished_at, created_at
		FROM sync_jobs
		WHERE mirror_config_id = $1
		ORDER BY created_at DESC, id DESC
	`, mirrorConfigID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*models.SyncJob
	for rows.Next() {
		job, err := scanSyncJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (s *PostgresSyncJobStore) HasActiveJobForMirror(mirrorConfigID uint64) (bool, error) {
	var exists bool
	err := s.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM sync_jobs
			WHERE mirror_config_id = $1
			  AND status IN ('queued', 'running')
		)
	`, mirrorConfigID).Scan(&exists)
	return exists, err
}

func (s *PostgresSyncJobStore) ClaimNextJob() (*models.SyncJob, error) {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	row := tx.QueryRow(`
		WITH next_job AS (
			SELECT id
			FROM sync_jobs
			WHERE status IN ('queued', 'retrying')
			  AND attempts < max_attempts
			ORDER BY created_at, id
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		UPDATE sync_jobs
		SET
			status = 'running',
			attempts = attempts + 1,
			started_at = NOW()
		WHERE id IN (SELECT id FROM next_job)
		RETURNING
			id, mirror_config_id, github_delivery_id, ref, ref_type, branch_or_tag,
			after_sha, deleted, status, attempts, max_attempts, last_error,
			started_at, finished_at, created_at
	`)

	job, err := scanSyncJob(row)
	if errors.Is(err, sql.ErrNoRows) {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return job, nil
}

func (s *PostgresSyncJobStore) UpdateJob(job *models.SyncJob) error {
	result, err := s.db.Exec(`
		UPDATE sync_jobs
		SET
			mirror_config_id = $2,
			github_delivery_id = $3,
			ref = $4,
			ref_type = $5,
			branch_or_tag = $6,
			after_sha = $7,
			deleted = $8,
			status = $9,
			attempts = $10,
			max_attempts = $11,
			last_error = $12,
			started_at = $13,
			finished_at = $14
		WHERE id = $1
	`,
		job.ID, job.MirrorConfigID, nullableString(job.GitHubDeliveryID), job.Ref, job.RefType,
		job.BranchOrTag, job.AfterSHA, job.Deleted, job.Status, job.Attempts,
		job.MaxAttempts, job.LastError, job.StartedAt, job.FinishedAt,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("job not found")
	}
	return nil
}

type mirrorConfigScanner interface {
	Scan(dest ...any) error
}

func scanMirrorConfig(scanner mirrorConfigScanner) (*models.MirrorConfig, error) {
	cfg := &models.MirrorConfig{}
	var syncSchedule sql.NullString
	if err := scanner.Scan(
		&cfg.ID,
		&cfg.UserID,
		&cfg.Name,
		&cfg.SourceOwner,
		&cfg.SourceRepo,
		&cfg.SourceRepoURL,
		&cfg.TargetOwner,
		&cfg.TargetRepo,
		&cfg.TargetRepoURL,
		&cfg.SourceTokenEnc,
		&cfg.TargetTokenEnc,
		&cfg.WebhookSecretEnc,
		&cfg.BranchPattern,
		&cfg.SyncTags,
		&cfg.SyncDeletes,
		&cfg.AllowForceUpdate,
		&syncSchedule,
		&cfg.Enabled,
		&cfg.LastSyncedAt,
		&cfg.LastScheduledAt,
		&cfg.CreatedAt,
		&cfg.UpdatedAt,
	); err != nil {
		return nil, err
	}
	cfg.SyncSchedule = syncSchedule.String
	return cfg, nil
}

type syncJobScanner interface {
	Scan(dest ...any) error
}

func scanSyncJob(scanner syncJobScanner) (*models.SyncJob, error) {
	job := &models.SyncJob{}
	var deliveryID sql.NullString
	if err := scanner.Scan(
		&job.ID,
		&job.MirrorConfigID,
		&deliveryID,
		&job.Ref,
		&job.RefType,
		&job.BranchOrTag,
		&job.AfterSHA,
		&job.Deleted,
		&job.Status,
		&job.Attempts,
		&job.MaxAttempts,
		&job.LastError,
		&job.StartedAt,
		&job.FinishedAt,
		&job.CreatedAt,
	); err != nil {
		return nil, err
	}
	if deliveryID.Valid {
		job.GitHubDeliveryID = deliveryID.String
	}
	return job, nil
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
