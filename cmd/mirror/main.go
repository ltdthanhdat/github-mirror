package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	nethttp "net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/dat-lt-amira/github-mirror/internal/auth"
	"github.com/dat-lt-amira/github-mirror/internal/http"
	"github.com/dat-lt-amira/github-mirror/internal/mirror"
	"github.com/dat-lt-amira/github-mirror/internal/store"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: mirror <server|worker|scheduler>")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "server":
		runServer()
	case "worker":
		runWorker()
	case "scheduler":
		runScheduler()
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		fmt.Println("Usage: mirror <server|worker|scheduler>")
		os.Exit(1)
	}
}

func runServer() {
	runtimeDeps, err := initRuntime()
	if err != nil {
		log.Fatalf("initialize runtime: %v", err)
	}
	defer runtimeDeps.db.Close()

	handler := &http.Handler{}
	router := http.NewRouter(handler, runtimeDeps.userStore, runtimeDeps.mirrorStore, runtimeDeps.jobStore)
	cacheDir := os.Getenv("MIRROR_CACHE_DIR")
	if cacheDir == "" {
		cacheDir = filepath.Join(os.TempDir(), "github-mirror-cache")
	}
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()
	go newWorker(runtimeDeps.mirrorStore, runtimeDeps.jobStore, cacheDir).Run(workerCtx)

	addr := ":8080"
	if v := os.Getenv("SERVER_ADDRESS"); v != "" {
		addr = v
	}

	srv := &nethttp.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Server: shutting down...")
		workerCancel()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	log.Printf("Server starting on %s", addr)
	log.Printf("Local login loaded from BASIC_AUTH_USERNAME / BASIC_AUTH_PASSWORD")
	if err := srv.ListenAndServe(); err != nil && err != nethttp.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
	log.Println("Server stopped")
}

func runWorker() {
	runtimeDeps, err := initRuntime()
	if err != nil {
		log.Fatalf("initialize runtime: %v", err)
	}
	defer runtimeDeps.db.Close()

	cacheDir := os.Getenv("MIRROR_CACHE_DIR")
	if cacheDir == "" {
		cacheDir = filepath.Join(os.TempDir(), "github-mirror-cache")
	}
	worker := newWorker(runtimeDeps.mirrorStore, runtimeDeps.jobStore, cacheDir)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Worker: received shutdown signal, finishing current job...")
		cancel()
	}()

	log.Printf("Worker starting (cache: %s)", cacheDir)
	worker.Run(ctx)
	log.Println("Worker stopped")
}

func runScheduler() {
	runtimeDeps, err := initRuntime()
	if err != nil {
		log.Fatalf("initialize runtime: %v", err)
	}
	defer runtimeDeps.db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Scheduler: received shutdown signal...")
		cancel()
	}()

	log.Printf("Scheduler starting (poll interval: %s)", schedulerPollInterval())
	newScheduler(runtimeDeps.mirrorStore, runtimeDeps.jobStore).Run(ctx)
	log.Println("Scheduler stopped")
}

func newWorker(mirrorStore store.MirrorConfigStore, jobStore store.SyncJobStore, cacheDir string) *mirror.Worker {
	return &mirror.Worker{
		ClaimJob: func() (*mirror.Job, error) {
			job, err := jobStore.ClaimNextJob()
			if err != nil {
				return nil, err
			}
			if job == nil {
				return nil, nil
			}

			cfg, err := mirrorStore.GetMirrorConfig(job.MirrorConfigID)
			if err != nil {
				return nil, err
			}

			sourceURL, err := resolveRemoteURL(cfg.SourceRepoURL, cfg.SourceOwner, cfg.SourceRepo, cfg.SourceTokenEnc)
			if err != nil {
				return nil, err
			}
			targetURL, err := resolveRemoteURL(cfg.TargetRepoURL, cfg.TargetOwner, cfg.TargetRepo, cfg.TargetTokenEnc)
			if err != nil {
				return nil, err
			}

			return &mirror.Job{
				ID:             job.ID,
				MirrorConfigID: job.MirrorConfigID,
				Ref:            job.Ref,
				RefType:        job.RefType,
				BranchOrTag:    job.BranchOrTag,
				AfterSHA:       job.AfterSHA,
				Deleted:        job.Deleted,
				SourceURL:      sourceURL,
				TargetURL:      targetURL,
				AllowForce:     cfg.AllowForceUpdate,
				SyncDeletes:    cfg.SyncDeletes,
				CacheDir:       fmt.Sprintf("%s/%d", cacheDir, job.MirrorConfigID),
			}, nil
		},
		UpdateJobStatus: func(jobID uint64, status string, jobErr error) error {
			job, err := jobStore.GetJob(jobID)
			if err != nil {
				return err
			}
			now := time.Now()
			job.FinishedAt = &now

			switch {
			case status == "succeeded":
				job.Status = "succeeded"
				job.LastError = ""
			case job.Attempts < job.MaxAttempts:
				job.Status = "retrying"
				if jobErr != nil {
					job.LastError = jobErr.Error()
				}
			default:
				job.Status = "failed"
				if jobErr != nil {
					job.LastError = jobErr.Error()
				}
			}

			if err := jobStore.UpdateJob(job); err != nil {
				return err
			}

			if status == "succeeded" {
				cfg, err := mirrorStore.GetMirrorConfig(job.MirrorConfigID)
				if err != nil {
					return err
				}
				cfg.LastSyncedAt = &now
				return mirrorStore.UpdateMirrorConfig(cfg)
			}

			return nil
		},
	}
}

func resolveRemoteURL(repoURL, owner, repo, token string) (string, error) {
	if strings.Contains(repoURL, "github.com/") {
		return mirror.BuildGitHubURL(owner, repo, token, nil)
	}
	return repoURL, nil
}

type runtimeDependencies struct {
	db          *sql.DB
	userStore   auth.UserStore
	mirrorStore store.MirrorConfigStore
	jobStore    store.SyncJobStore
}

func initRuntime() (*runtimeDependencies, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL must be set")
	}

	db, err := store.OpenPostgresDB(databaseURL)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := store.RunMigrations(ctx, db); err != nil {
		db.Close()
		return nil, err
	}

	username := os.Getenv("BASIC_AUTH_USERNAME")
	password := os.Getenv("BASIC_AUTH_PASSWORD")
	if username == "" || password == "" {
		db.Close()
		return nil, fmt.Errorf("BASIC_AUTH_USERNAME and BASIC_AUTH_PASSWORD must be set")
	}

	userStore := auth.NewPostgresUserStore(db)
	if err := userStore.EnsureAdminUser(ctx, username, password, "Local Admin"); err != nil {
		db.Close()
		return nil, fmt.Errorf("ensure admin user: %w", err)
	}

	return &runtimeDependencies{
		db:          db,
		userStore:   userStore,
		mirrorStore: store.NewPostgresMirrorConfigStore(db),
		jobStore:    store.NewPostgresSyncJobStore(db),
	}, nil
}
