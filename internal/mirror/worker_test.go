package mirror

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestWorkerProcessNextSyncsBranchToTarget(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not found")
	}

	tmpDir := t.TempDir()
	sourceBare := filepath.Join(tmpDir, "source.git")
	targetBare := filepath.Join(tmpDir, "target.git")
	worktree := filepath.Join(tmpDir, "worktree")
	cacheDir := filepath.Join(tmpDir, "cache")

	runGit(t, tmpDir, "init", "--bare", sourceBare)
	runGit(t, tmpDir, "init", "--bare", targetBare)
	runGit(t, tmpDir, "init", worktree)
	runGit(t, worktree, "config", "user.email", "demo@example.com")
	runGit(t, worktree, "config", "user.name", "Demo User")
	runGit(t, worktree, "remote", "add", "origin", sourceBare)
	writeFile(t, filepath.Join(worktree, "README.md"), []byte("mirror me\n"))
	runGit(t, worktree, "add", "README.md")
	runGit(t, worktree, "commit", "-m", "initial commit")
	runGit(t, worktree, "branch", "-M", "main")
	runGit(t, worktree, "push", "origin", "main")

	job := &Job{
		ID:          1,
		RefType:     "branch",
		BranchOrTag: "main",
		SourceURL:   sourceBare,
		TargetURL:   targetBare,
		AllowForce:  true,
		CacheDir:    cacheDir,
	}

	var updatedStatus string
	worker := &Worker{
		ClaimJob: func() (*Job, error) {
			if job == nil {
				return nil, nil
			}
			current := job
			job = nil
			return current, nil
		},
		UpdateJobStatus: func(jobID uint64, status string, err error) error {
			updatedStatus = status
			if err != nil {
				t.Fatalf("unexpected worker error: %v", err)
			}
			return nil
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	worker.processNext(ctx)

	if updatedStatus != "succeeded" {
		t.Fatalf("expected succeeded status, got %q", updatedStatus)
	}

	runGit(t, tmpDir, "--git-dir", targetBare, "show-ref", "--verify", "refs/heads/main")
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
	}
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
