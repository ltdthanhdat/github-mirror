package mirror

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Worker processes sync jobs from the queue.
type Worker struct {
	// JobQueue is a function that claims the next job.
	// Returns nil if no job is available.
	ClaimJob func() (*Job, error)
	// UpdateJobStatus updates the status of a job.
	UpdateJobStatus func(jobID uint64, status string, err error) error
}

// Job represents a sync job to be processed.
type Job struct {
	ID             uint64
	MirrorConfigID uint64
	Ref            string
	RefType        string
	BranchOrTag    string
	AfterSHA       string
	Deleted        bool
	SourceURL      string
	TargetURL      string
	AllowForce     bool
	SyncDeletes    bool
	CacheDir       string
}

// Run starts the worker loop. It polls for jobs and processes them.
func (w *Worker) Run(ctx context.Context) {
	log.Println("Worker: starting")
	for {
		select {
		case <-ctx.Done():
			log.Println("Worker: shutting down")
			return
		default:
			w.processNext(ctx)
		}
	}
}

func (w *Worker) processNext(ctx context.Context) {
	job, err := w.ClaimJob()
	if err != nil {
		log.Printf("Worker: failed to claim job: %v", err)
		time.Sleep(2 * time.Second)
		return
	}
	if job == nil {
		time.Sleep(2 * time.Second)
		return
	}

	log.Printf("Worker: processing job %d (mirror %d, ref: %s %s)", job.ID, job.MirrorConfigID, job.RefType, job.BranchOrTag)

	// Set timeout for the entire sync operation
	syncCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	var syncErr error

	if job.Deleted && job.SyncDeletes {
		// Handle deleted ref (branch/tag deletion)
		syncErr = w.handleDelete(syncCtx, job)
	} else {
		// Normal sync
		syncErr = w.handleSync(syncCtx, job)
	}

	// Update job status
	if syncErr != nil {
		log.Printf("Worker: job %d failed: %v", job.ID, syncErr)
		w.UpdateJobStatus(job.ID, "failed", syncErr)
	} else {
		log.Printf("Worker: job %d completed successfully", job.ID)
		w.UpdateJobStatus(job.ID, "succeeded", nil)
	}
}

func (w *Worker) handleSync(ctx context.Context, job *Job) error {
	// Ensure bare repo exists
	if err := InitBareRepo(job.CacheDir); err != nil {
		return err
	}

	// Acquire lock for this mirror
	lock, err := AcquireLock(job.CacheDir)
	if err != nil {
		return err
	}
	defer ReleaseLock(lock)

	// Sync the ref
	return SyncRef(ctx, job.CacheDir, job.SourceURL, job.TargetURL, job.RefType, job.BranchOrTag, job.AllowForce)
}

func (w *Worker) handleDelete(ctx context.Context, job *Job) error {
	if job.RefType != "branch" {
		return nil // only branch deletes are supported
	}

	// Ensure bare repo exists
	if err := InitBareRepo(job.CacheDir); err != nil {
		return err
	}

	lock, err := AcquireLock(job.CacheDir)
	if err != nil {
		return err
	}
	defer ReleaseLock(lock)

	// Push delete to target (format: :refs/heads/<branch>)
	deleteRef := fmt.Sprintf("refs/heads/%s", job.BranchOrTag)
	return DeleteRef(ctx, job.CacheDir, "target", deleteRef)
}
