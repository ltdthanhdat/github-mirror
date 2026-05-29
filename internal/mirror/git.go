package mirror

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/dat-lt-amira/github-mirror/internal/crypto"
)

// InitBareRepo initializes a bare repository at the given path.
func InitBareRepo(cacheDir string) error {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache dir: %w", err)
	}

	// Check if already initialized
	if _, err := os.Stat(filepath.Join(cacheDir, "HEAD")); err == nil {
		return nil // already exists
	}

	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = cacheDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git init --bare failed: %v, output: %s", err, string(output))
	}
	return nil
}

// UpdateRemoteURL sets or updates a remote URL in the repository.
func UpdateRemoteURL(dir, name, url string) error {
	// Try to set URL (works if remote exists), fall back to add
	cmd := exec.Command("git", "remote", "set-url", name, url)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		// Remote doesn't exist yet, add it
		cmd = exec.Command("git", "remote", "add", name, url)
		cmd.Dir = dir
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git remote add %s %s failed: %v, output: %s", name, url, err, string(output))
		}
	}
	return nil
}

// FetchRef fetches a ref from the remote into the local bare repo.
func FetchRef(ctx context.Context, dir, remote, refSpec string) error {
	cmd := exec.CommandContext(ctx, "git", "fetch", remote, refSpec)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git fetch %s %s failed: %v, output: %s", remote, refSpec, err, string(output))
	}
	return nil
}

// PushRef pushes a refspec to the remote.
func PushRef(ctx context.Context, dir, remote, refSpec string, force bool) error {
	args := []string{"push"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, remote, refSpec)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push failed: %v, output: %s", err, string(output))
	}
	return nil
}

// DeleteRef deletes a ref from the remote by pushing an empty refspec.
func DeleteRef(ctx context.Context, dir, remote, refSpec string) error {
	// Delete by pushing ":refs/heads/branch"
	deleteSpec := fmt.Sprintf(":%s", refSpec)
	cmd := exec.CommandContext(ctx, "git", "push", remote, deleteSpec)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push delete %s %s failed: %v, output: %s", remote, refSpec, err, string(output))
	}
	return nil
}

// RunGC runs git gc --auto on the repo.
func RunGC(ctx context.Context, dir string) error {
	cmd := exec.CommandContext(ctx, "git", "gc", "--auto")
	cmd.Dir = dir
	return cmd.Run()
}

// AcquireLock creates a lock file for the given cache directory and returns the file handle.
// Caller must close/release the lock.
func AcquireLock(cacheDir string) (*os.File, error) {
	lockPath := filepath.Join(cacheDir, "mirror.lock")
	if err := os.MkdirAll(filepath.Dir(lockPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create lock dir: %w", err)
	}

	f, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	return f, nil
}

// ReleaseLock releases the flock and closes the lock file.
func ReleaseLock(f *os.File) {
	syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	f.Close()
}

// BuildGitHubURL builds a GitHub remote URL with an access token.
func BuildGitHubURL(owner, repo, tokenEnc string, key []byte) (string, error) {
	token := tokenEnc
	if len(key) > 0 && tokenEnc != "" {
		decrypted, err := crypto.Decrypt(tokenEnc, key)
		if err != nil {
			return "", fmt.Errorf("failed to decrypt token: %w", err)
		}
		token = string(decrypted)
	}
	if token == "" {
		return fmt.Sprintf("https://github.com/%s/%s.git", owner, repo), nil
	}
	return fmt.Sprintf("https://x-access-token:%s@github.com/%s/%s.git", token, owner, repo), nil
}

// MaskGitURL replaces the token portion of a GitHub URL with asterisks for safe logging.
func MaskGitURL(url string) string {
	// Input:  https://x-access-token:ghp_abc123@github.com/owner/repo.git
	// Output: https://x-access-token:***@github.com/owner/repo.git
	if len(url) < 50 {
		return url
	}
	atIdx := -1
	for i := len(url) - 1; i >= 0; i-- {
		if url[i] == '@' {
			atIdx = i
			break
		}
	}
	if atIdx < 0 {
		return url
	}
	return url[:atIdx-3] + "***@" + url[atIdx+1:]
}

// SyncRef syncs a single ref from source to target.
func SyncRef(ctx context.Context, cacheDir, sourceURL, targetURL, refType, refName string, allowForce bool) error {
	switch refType {
	case "branch":
		return SyncBranch(ctx, cacheDir, sourceURL, targetURL, refName, allowForce)
	case "tag":
		return SyncTag(ctx, cacheDir, sourceURL, targetURL, refName, allowForce)
	default:
		return fmt.Errorf("unsupported ref type: %s", refType)
	}
}

// SyncBranch syncs a branch from source to target.
func SyncBranch(ctx context.Context, cacheDir, sourceURL, targetURL, branch string, allowForce bool) error {
	// Update source URL (token may have rotated)
	if err := UpdateRemoteURL(cacheDir, "source", sourceURL); err != nil {
		return err
	}
	if err := UpdateRemoteURL(cacheDir, "target", targetURL); err != nil {
		return err
	}

	// Fetch the branch from source
	sourceRefSpec := fmt.Sprintf("+refs/heads/%s:refs/heads/%s", branch, branch)
	if err := FetchRef(ctx, cacheDir, "source", sourceRefSpec); err != nil {
		return fmt.Errorf("failed to fetch branch %s: %w", branch, err)
	}

	// Push to target
	pushRefSpec := fmt.Sprintf("refs/heads/%s:refs/heads/%s", branch, branch)
	return PushRef(ctx, cacheDir, "target", pushRefSpec, allowForce)
}

// SyncTag syncs a tag from source to target.
func SyncTag(ctx context.Context, cacheDir, sourceURL, targetURL, tag string, allowForce bool) error {
	if err := UpdateRemoteURL(cacheDir, "source", sourceURL); err != nil {
		return err
	}
	if err := UpdateRemoteURL(cacheDir, "target", targetURL); err != nil {
		return err
	}

	sourceRefSpec := fmt.Sprintf("+refs/tags/%s:refs/tags/%s", tag, tag)
	if err := FetchRef(ctx, cacheDir, "source", sourceRefSpec); err != nil {
		return fmt.Errorf("failed to fetch tag %s: %w", tag, err)
	}

	pushRefSpec := fmt.Sprintf("refs/tags/%s:refs/tags/%s", tag, tag)
	return PushRef(ctx, cacheDir, "target", pushRefSpec, allowForce)
}
