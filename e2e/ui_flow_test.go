package e2e

import (
	"context"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/dat-lt-amira/github-mirror/internal/auth"
	apphttp "github.com/dat-lt-amira/github-mirror/internal/http"
	"github.com/dat-lt-amira/github-mirror/internal/models"
	"github.com/dat-lt-amira/github-mirror/internal/store"
	"net/http/httptest"
)

func TestMirrorUIFlowE2E(t *testing.T) {
	chromePath := findChrome()
	if chromePath == "" {
		t.Skip("chromium or chrome binary not found")
	}

	user := &models.User{
		ID:       1,
		Email:    "demo@example.com",
		FullName: "Demo User",
	}
	if err := user.SetPassword("secret123"); err != nil {
		t.Fatalf("set password: %v", err)
	}

	userStore := auth.NewInMemoryUserStore([]*models.User{user})
	mirrorStore := store.NewInMemoryMirrorConfigStore()
	jobStore := store.NewInMemorySyncJobStore()
	handler := &apphttp.Handler{}

	server := httptest.NewServer(apphttp.NewRouter(handler, userStore, mirrorStore, jobStore))
	defer server.Close()

	artifactDir := screenshotDir(t)
	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte("demo@example.com:secret123"))

	allocCtx, allocCancel := chromedp.NewExecAllocator(
		context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.ExecPath(chromePath),
			chromedp.Headless,
			chromedp.NoSandbox,
			chromedp.DisableGPU,
			chromedp.WindowSize(1440, 1200),
		)...,
	)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, timeoutCancel := context.WithTimeout(ctx, 90*time.Second)
	defer timeoutCancel()

	chromedp.ListenTarget(ctx, func(ev any) {
		if _, ok := ev.(*page.EventJavascriptDialogOpening); ok {
			go func() {
				_ = chromedp.Run(ctx, page.HandleJavaScriptDialog(true))
			}()
		}
	})

	if err := chromedp.Run(ctx,
		network.Enable(),
		network.SetExtraHTTPHeaders(network.Headers{"Authorization": authHeader}),
	); err != nil {
		t.Fatalf("configure browser: %v", err)
	}

	var bodyText string
	if err := chromedp.Run(ctx,
		chromedp.Navigate(server.URL+"/"),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Text("body", &bodyText, chromedp.ByQuery),
	); err != nil {
		t.Fatalf("load dashboard: %v", err)
	}
	requireContains(t, bodyText, "Mirror Configurations")
	requireContains(t, bodyText, "No mirrors configured yet")
	requireContains(t, bodyText, "PAT Tokens & Webhook Setup")
	captureFullPage(t, ctx, filepath.Join(artifactDir, "dashboard-empty.png"))

	if err := chromedp.Run(ctx,
		chromedp.Navigate(server.URL+"/guide"),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Text("body", &bodyText, chromedp.ByQuery),
	); err != nil {
		t.Fatalf("load setup guide: %v", err)
	}
	requireContains(t, bodyText, "PAT Tokens & Webhook Setup")
	requireContains(t, bodyText, "Create the Source Token")
	requireContains(t, bodyText, "Create the Target Token")
	requireContains(t, bodyText, "application/json")
	requireContains(t, bodyText, "Push")
	captureFullPage(t, ctx, filepath.Join(artifactDir, "setup-guide-desktop.png"))

	if err := chromedp.Run(ctx,
		chromedp.EmulateViewport(390, 844),
		chromedp.Navigate(server.URL+"/guide"),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Text("body", &bodyText, chromedp.ByQuery),
	); err != nil {
		t.Fatalf("load setup guide mobile layout: %v", err)
	}
	requireContains(t, bodyText, "PAT Tokens & Webhook Setup")
	requireContains(t, bodyText, "Create Mirror")
	captureFullPage(t, ctx, filepath.Join(artifactDir, "setup-guide-mobile.png"))

	if err := chromedp.Run(ctx, chromedp.EmulateViewport(1440, 1200)); err != nil {
		t.Fatalf("restore desktop viewport: %v", err)
	}

	if err := chromedp.Run(ctx,
		chromedp.Navigate(server.URL+"/mirrors/new"),
		chromedp.WaitVisible(`form[action="/mirrors"]`, chromedp.ByQuery),
		chromedp.Text("body", &bodyText, chromedp.ByQuery),
	); err != nil {
		t.Fatalf("load mirror form: %v", err)
	}
	requireContains(t, bodyText, "Create Mirror Configuration")
	requireContains(t, bodyText, "Need to create a PAT first? Open the setup guide.")
	captureFullPage(t, ctx, filepath.Join(artifactDir, "mirror-form.png"))

	if err := chromedp.Run(ctx,
		chromedp.SendKeys(`input[name="name"]`, "Demo Mirror", chromedp.ByQuery),
		chromedp.SendKeys(`input[name="source_url"]`, "https://github.com/source-org/source-repo", chromedp.ByQuery),
		chromedp.SendKeys(`input[name="source_token"]`, "abcd1234", chromedp.ByQuery),
		chromedp.SendKeys(`input[name="target_url"]`, "https://github.com/target-org/target-repo", chromedp.ByQuery),
		chromedp.SendKeys(`input[name="target_token"]`, "wxyz6789", chromedp.ByQuery),
		chromedp.SendKeys(`input[name="sync_schedule"]`, "*/10 * * * *", chromedp.ByQuery),
		chromedp.Submit(`form[action="/mirrors"]`, chromedp.ByQuery),
		chromedp.WaitVisible(`form[action="/mirrors/1/delete"]`, chromedp.ByQuery),
		chromedp.Text("body", &bodyText, chromedp.ByQuery),
	); err != nil {
		t.Fatalf("create mirror flow: %v", err)
	}
	requireContains(t, bodyText, "Demo Mirror")
	requireContains(t, bodyText, "Mirror created and initial sync queued.")
	requireContains(t, bodyText, "Need the full PAT and webhook walkthrough? Open the setup guide.")
	requireContains(t, bodyText, "*/10 * * * *")
	requireContains(t, bodyText, "UTC")
	captureFullPage(t, ctx, filepath.Join(artifactDir, "mirror-detail.png"))

	if err := chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelector('form[action="/mirrors/1/sync"]').submit()`, nil),
		chromedp.WaitVisible(`form[action="/mirrors/1/delete"]`, chromedp.ByQuery),
		chromedp.Text("body", &bodyText, chromedp.ByQuery),
	); err != nil {
		t.Fatalf("sync mirror flow: %v", err)
	}
	requireContains(t, bodyText, "Sync job enqueued.")
	requireContains(t, bodyText, "queued")

	if err := chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelector('form[action="/mirrors/1/delete"]').submit()`, nil),
		chromedp.WaitVisible(`#mirror-list`, chromedp.ByQuery),
		chromedp.Text("body", &bodyText, chromedp.ByQuery),
	); err != nil {
		t.Fatalf("delete mirror flow: %v", err)
	}
	requireContains(t, bodyText, "Mirror deleted.")
	requireContains(t, bodyText, "No mirrors configured yet")
	captureFullPage(t, ctx, filepath.Join(artifactDir, "dashboard-after-delete.png"))
}

func findChrome() string {
	if path := os.Getenv("CHROMIUM_BIN"); path != "" {
		return path
	}
	for _, candidate := range []string{"chromium", "chromium-browser", "google-chrome", "google-chrome-stable", "/snap/bin/chromium"} {
		if path, err := exec.LookPath(candidate); err == nil {
			return path
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func screenshotDir(t *testing.T) string {
	t.Helper()
	if dir := os.Getenv("UI_E2E_ARTIFACT_DIR"); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("create screenshot dir: %v", err)
		}
		return dir
	}
	return t.TempDir()
}

func captureFullPage(t *testing.T, ctx context.Context, path string) {
	t.Helper()
	var buf []byte
	if err := chromedp.Run(ctx, chromedp.FullScreenshot(&buf, 90)); err != nil {
		t.Fatalf("capture screenshot %s: %v", path, err)
	}
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		t.Fatalf("write screenshot %s: %v", path, err)
	}
}

func requireContains(t *testing.T, body, want string) {
	t.Helper()
	if !strings.Contains(body, want) {
		t.Fatalf("expected body to contain %q", want)
	}
	if strings.Contains(body, "HTTP ERROR") || strings.Contains(body, "This page isn’t working") || strings.Contains(body, "This page isn't working") {
		t.Fatalf("browser error page detected: %s", body)
	}
}
