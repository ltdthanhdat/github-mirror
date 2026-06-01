package http

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/dat-lt-amira/github-mirror/internal/auth"
	"github.com/dat-lt-amira/github-mirror/internal/models"
	"github.com/dat-lt-amira/github-mirror/internal/store"
)

func TestCreateMirrorHandlerAcceptsRepositoryURLs(t *testing.T) {
	userStore, mirrorStore, jobStore := newMirrorHandlerTestDeps(t)
	handler := &Handler{}
	router := NewRouter(handler, userStore, mirrorStore, jobStore)

	req := httptest.NewRequest(http.MethodPost, "/mirrors", strings.NewReader(`{"name":"Demo Mirror","source_url":"https://github.com/source-org/source-repo","target_url":"https://github.com/target-org/target-repo.git","source_token":"abcd1234","target_token":"wxyz6789"}`))
	req.Header.Set("Authorization", basicAuthHeader("demo@example.com", "secret123"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	configs, err := mirrorStore.ListMirrorConfigsByUser(1)
	if err != nil {
		t.Fatalf("list mirror configs: %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("expected 1 mirror config, got %d", len(configs))
	}

	cfg := configs[0]
	if cfg.SourceOwner != "source-org" || cfg.SourceRepo != "source-repo" {
		t.Fatalf("unexpected source repo fields: %s/%s", cfg.SourceOwner, cfg.SourceRepo)
	}
	if cfg.TargetOwner != "target-org" || cfg.TargetRepo != "target-repo" {
		t.Fatalf("unexpected target repo fields: %s/%s", cfg.TargetOwner, cfg.TargetRepo)
	}
	if cfg.SourceRepoURL != "https://github.com/source-org/source-repo.git" {
		t.Fatalf("unexpected source repo URL: %s", cfg.SourceRepoURL)
	}
	if cfg.TargetRepoURL != "https://github.com/target-org/target-repo.git" {
		t.Fatalf("unexpected target repo URL: %s", cfg.TargetRepoURL)
	}
	if cfg.SyncSchedule != "" {
		t.Fatalf("expected empty sync schedule by default, got %q", cfg.SyncSchedule)
	}
}

func TestCreateMirrorHandlerRejectsInvalidRepositoryURL(t *testing.T) {
	userStore, mirrorStore, jobStore := newMirrorHandlerTestDeps(t)
	handler := &Handler{}
	router := NewRouter(handler, userStore, mirrorStore, jobStore)

	req := httptest.NewRequest(http.MethodPost, "/mirrors", strings.NewReader(`{"name":"Demo Mirror","source_url":"https://github.com/source-org/source-repo/tree/main","target_url":"https://github.com/target-org/target-repo","source_token":"abcd1234","target_token":"wxyz6789"}`))
	req.Header.Set("Authorization", basicAuthHeader("demo@example.com", "secret123"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}

	configs, err := mirrorStore.ListMirrorConfigsByUser(1)
	if err != nil {
		t.Fatalf("list mirror configs: %v", err)
	}
	if len(configs) != 0 {
		t.Fatalf("expected no mirror configs after invalid request, got %d", len(configs))
	}
}

func TestCreateMirrorHandlerUsesHXLocationForBoostedForms(t *testing.T) {
	userStore, mirrorStore, jobStore := newMirrorHandlerTestDeps(t)
	handler := &Handler{}
	router := NewRouter(handler, userStore, mirrorStore, jobStore)

	form := url.Values{
		"name":               {"Demo Mirror"},
		"source_url":         {"https://github.com/source-org/source-repo"},
		"target_url":         {"https://github.com/target-org/target-repo"},
		"source_token":       {"abcd1234"},
		"target_token":       {"wxyz6789"},
		"branch_pattern":     {"*"},
		"sync_tags":          {"on"},
		"allow_force_update": {"on"},
	}
	req := httptest.NewRequest(http.MethodPost, "/mirrors", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", basicAuthHeader("demo@example.com", "secret123"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Current-URL", "http://example.com/mirrors/new")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("HX-Location"); got != "/mirrors/1?flash=Mirror+created+and+initial+sync+queued." {
		t.Fatalf("expected HX-Location header, got %q", got)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("expected empty response body for HX redirect, got %q", rec.Body.String())
	}
}

func TestCreateMirrorHandlerPersistsSyncSchedule(t *testing.T) {
	userStore, mirrorStore, jobStore := newMirrorHandlerTestDeps(t)
	handler := &Handler{}
	router := NewRouter(handler, userStore, mirrorStore, jobStore)

	req := httptest.NewRequest(http.MethodPost, "/mirrors", strings.NewReader(`{"name":"Demo Mirror","source_url":"https://github.com/source-org/source-repo","target_url":"https://github.com/target-org/target-repo.git","source_token":"abcd1234","target_token":"wxyz6789","sync_schedule":"*/10 * * * *"}`))
	req.Header.Set("Authorization", basicAuthHeader("demo@example.com", "secret123"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	cfg, err := mirrorStore.GetMirrorConfig(1)
	if err != nil {
		t.Fatalf("get mirror config: %v", err)
	}
	if cfg.SyncSchedule != "*/10 * * * *" {
		t.Fatalf("expected sync schedule to persist, got %q", cfg.SyncSchedule)
	}
}

func TestCreateMirrorHandlerRendersFormErrorForHTMLValidation(t *testing.T) {
	userStore, mirrorStore, jobStore := newMirrorHandlerTestDeps(t)
	handler := &Handler{}
	router := NewRouter(handler, userStore, mirrorStore, jobStore)

	form := url.Values{
		"name":               {"Broken Mirror"},
		"source_url":         {"https://github.com/source-org/source-repo/tree/main"},
		"target_url":         {"https://github.com/target-org/target-repo"},
		"source_token":       {"abcd1234"},
		"target_token":       {"wxyz6789"},
		"branch_pattern":     {"release/*"},
		"sync_tags":          {"on"},
		"allow_force_update": {"on"},
	}
	req := httptest.NewRequest(http.MethodPost, "/mirrors", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", basicAuthHeader("demo@example.com", "secret123"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Current-URL", "http://example.com/mirrors/new")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Invalid source repository URL") {
		t.Fatalf("expected validation error in form response, got %q", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Broken Mirror") {
		t.Fatalf("expected form values to be preserved, got %q", rec.Body.String())
	}
	if got := rec.Header().Get("HX-Replace-Url"); got != "/mirrors/new" {
		t.Fatalf("expected HX-Replace-Url to keep form URL, got %q", got)
	}
}

func TestCreateMirrorHandlerRejectsInvalidCronSchedule(t *testing.T) {
	userStore, mirrorStore, jobStore := newMirrorHandlerTestDeps(t)
	handler := &Handler{}
	router := NewRouter(handler, userStore, mirrorStore, jobStore)

	form := url.Values{
		"name":          {"Broken Mirror"},
		"source_url":    {"https://github.com/source-org/source-repo"},
		"target_url":    {"https://github.com/target-org/target-repo"},
		"source_token":  {"abcd1234"},
		"target_token":  {"wxyz6789"},
		"sync_schedule": {"invalid cron"},
	}
	req := httptest.NewRequest(http.MethodPost, "/mirrors", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", basicAuthHeader("demo@example.com", "secret123"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Current-URL", "http://example.com/mirrors/new")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Invalid cron schedule") {
		t.Fatalf("expected cron validation error in form response, got %q", rec.Body.String())
	}
	if _, err := mirrorStore.GetMirrorConfig(1); err == nil {
		t.Fatalf("expected mirror config not to be created")
	}
}

func TestUpdateMirrorHandlerKeepsExistingTokensWhenBlank(t *testing.T) {
	userStore, mirrorStore, jobStore := newMirrorHandlerTestDeps(t)
	handler := &Handler{}
	router := NewRouter(handler, userStore, mirrorStore, jobStore)

	cfg := &models.MirrorConfig{
		UserID:           1,
		Name:             "Original",
		SourceOwner:      "source-org",
		SourceRepo:       "source-repo",
		SourceRepoURL:    "https://github.com/source-org/source-repo.git",
		TargetOwner:      "target-org",
		TargetRepo:       "target-repo",
		TargetRepoURL:    "https://github.com/target-org/target-repo.git",
		SourceTokenEnc:   "source-old-token",
		TargetTokenEnc:   "target-old-token",
		BranchPattern:    "*",
		SyncTags:         true,
		SyncDeletes:      false,
		AllowForceUpdate: true,
		Enabled:          true,
	}
	if err := mirrorStore.CreateMirrorConfig(cfg); err != nil {
		t.Fatalf("create mirror config: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/mirrors/1", strings.NewReader(`{"name":"Updated","source_url":"https://github.com/source-org/source-repo","target_url":"https://github.com/target-org/target-repo","source_token":"","target_token":"","branch_pattern":"release/*","sync_tags":false,"sync_deletes":true,"allow_force_update":false}`))
	req.Header.Set("Authorization", basicAuthHeader("demo@example.com", "secret123"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	updated, err := mirrorStore.GetMirrorConfig(cfg.ID)
	if err != nil {
		t.Fatalf("get mirror config: %v", err)
	}
	if updated.Name != "Updated" {
		t.Fatalf("expected updated name, got %q", updated.Name)
	}
	if updated.SourceTokenEnc != "source-old-token" {
		t.Fatalf("expected source token to be preserved, got %q", updated.SourceTokenEnc)
	}
	if updated.TargetTokenEnc != "target-old-token" {
		t.Fatalf("expected target token to be preserved, got %q", updated.TargetTokenEnc)
	}
	if updated.BranchPattern != "release/*" {
		t.Fatalf("expected updated branch pattern, got %q", updated.BranchPattern)
	}
	if updated.SyncTags {
		t.Fatalf("expected sync tags to be false")
	}
	if !updated.SyncDeletes {
		t.Fatalf("expected sync deletes to be true")
	}
	if updated.AllowForceUpdate {
		t.Fatalf("expected allow force update to be false")
	}
}

func TestUpdateMirrorHandlerReplacesTokensWhenProvided(t *testing.T) {
	userStore, mirrorStore, jobStore := newMirrorHandlerTestDeps(t)
	handler := &Handler{}
	router := NewRouter(handler, userStore, mirrorStore, jobStore)

	cfg := &models.MirrorConfig{
		UserID:           1,
		Name:             "Original",
		SourceOwner:      "source-org",
		SourceRepo:       "source-repo",
		SourceRepoURL:    "https://github.com/source-org/source-repo.git",
		TargetOwner:      "target-org",
		TargetRepo:       "target-repo",
		TargetRepoURL:    "https://github.com/target-org/target-repo.git",
		SourceTokenEnc:   "source-old-token",
		TargetTokenEnc:   "target-old-token",
		BranchPattern:    "*",
		SyncTags:         true,
		SyncDeletes:      false,
		AllowForceUpdate: true,
		Enabled:          true,
	}
	if err := mirrorStore.CreateMirrorConfig(cfg); err != nil {
		t.Fatalf("create mirror config: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/mirrors/1", strings.NewReader(`{"name":"Updated","source_url":"https://github.com/source-org/source-repo","target_url":"https://github.com/target-org/target-repo","source_token":"source-new-token","target_token":"target-new-token","branch_pattern":"*","sync_tags":true,"sync_deletes":false,"allow_force_update":true}`))
	req.Header.Set("Authorization", basicAuthHeader("demo@example.com", "secret123"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	updated, err := mirrorStore.GetMirrorConfig(cfg.ID)
	if err != nil {
		t.Fatalf("get mirror config: %v", err)
	}
	if updated.SourceTokenEnc != "source-new-token" {
		t.Fatalf("expected source token to be replaced, got %q", updated.SourceTokenEnc)
	}
	if updated.TargetTokenEnc != "target-new-token" {
		t.Fatalf("expected target token to be replaced, got %q", updated.TargetTokenEnc)
	}
}

func TestUpdateMirrorHandlerRejectsInvalidCronSchedule(t *testing.T) {
	userStore, mirrorStore, jobStore := newMirrorHandlerTestDeps(t)
	handler := &Handler{}
	router := NewRouter(handler, userStore, mirrorStore, jobStore)

	cfg := &models.MirrorConfig{
		UserID:           1,
		Name:             "Original",
		SourceOwner:      "source-org",
		SourceRepo:       "source-repo",
		SourceRepoURL:    "https://github.com/source-org/source-repo.git",
		TargetOwner:      "target-org",
		TargetRepo:       "target-repo",
		TargetRepoURL:    "https://github.com/target-org/target-repo.git",
		SourceTokenEnc:   "source-old-token",
		TargetTokenEnc:   "target-old-token",
		BranchPattern:    "*",
		SyncSchedule:     "*/10 * * * *",
		SyncTags:         true,
		AllowForceUpdate: true,
		Enabled:          true,
	}
	if err := mirrorStore.CreateMirrorConfig(cfg); err != nil {
		t.Fatalf("create mirror config: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/mirrors/1", strings.NewReader(`{"name":"Updated","source_url":"https://github.com/source-org/source-repo","target_url":"https://github.com/target-org/target-repo","source_token":"","target_token":"","branch_pattern":"*","sync_schedule":"bad cron","sync_tags":true,"sync_deletes":false,"allow_force_update":true}`))
	req.Header.Set("Authorization", basicAuthHeader("demo@example.com", "secret123"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
	updated, err := mirrorStore.GetMirrorConfig(cfg.ID)
	if err != nil {
		t.Fatalf("get mirror config: %v", err)
	}
	if updated.SyncSchedule != "*/10 * * * *" {
		t.Fatalf("expected existing sync schedule to remain unchanged, got %q", updated.SyncSchedule)
	}
}

func TestEditMirrorScheduleFormRendersCurrentValue(t *testing.T) {
	userStore, mirrorStore, jobStore := newMirrorHandlerTestDeps(t)
	handler := &Handler{}
	router := NewRouter(handler, userStore, mirrorStore, jobStore)

	cfg := &models.MirrorConfig{
		UserID:           1,
		Name:             "Scheduled Mirror",
		SourceOwner:      "source-org",
		SourceRepo:       "source-repo",
		SourceRepoURL:    "https://github.com/source-org/source-repo.git",
		TargetOwner:      "target-org",
		TargetRepo:       "target-repo",
		TargetRepoURL:    "https://github.com/target-org/target-repo.git",
		BranchPattern:    "*",
		SyncSchedule:     "0 * * * *",
		SyncTags:         true,
		AllowForceUpdate: true,
		Enabled:          true,
	}
	if err := mirrorStore.CreateMirrorConfig(cfg); err != nil {
		t.Fatalf("create mirror config: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/mirrors/1/schedule", nil)
	req.Header.Set("Authorization", basicAuthHeader("demo@example.com", "secret123"))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Manage Cron Schedule") {
		t.Fatalf("expected schedule form title, got %q", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "0 * * * *") || !strings.Contains(rec.Body.String(), "UTC") {
		t.Fatalf("expected schedule form to show current value and UTC guidance, got %q", rec.Body.String())
	}
}

func TestUpdateMirrorScheduleHandlerPersistsOnlySchedule(t *testing.T) {
	userStore, mirrorStore, jobStore := newMirrorHandlerTestDeps(t)
	handler := &Handler{}
	router := NewRouter(handler, userStore, mirrorStore, jobStore)

	cfg := &models.MirrorConfig{
		UserID:           1,
		Name:             "Original",
		SourceOwner:      "source-org",
		SourceRepo:       "source-repo",
		SourceRepoURL:    "https://github.com/source-org/source-repo.git",
		TargetOwner:      "target-org",
		TargetRepo:       "target-repo",
		TargetRepoURL:    "https://github.com/target-org/target-repo.git",
		SourceTokenEnc:   "source-old-token",
		TargetTokenEnc:   "target-old-token",
		BranchPattern:    "release/*",
		SyncSchedule:     "",
		SyncTags:         true,
		SyncDeletes:      true,
		AllowForceUpdate: false,
		Enabled:          true,
	}
	if err := mirrorStore.CreateMirrorConfig(cfg); err != nil {
		t.Fatalf("create mirror config: %v", err)
	}

	form := url.Values{
		"sync_schedule": {"*/30 * * * *"},
	}
	req := httptest.NewRequest(http.MethodPost, "/mirrors/1/schedule", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", basicAuthHeader("demo@example.com", "secret123"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Current-URL", "http://example.com/mirrors/1/schedule")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("HX-Location"); got != "/mirrors/1?flash=Automatic+sync+schedule+updated." {
		t.Fatalf("expected HX-Location header, got %q", got)
	}

	updated, err := mirrorStore.GetMirrorConfig(cfg.ID)
	if err != nil {
		t.Fatalf("get mirror config: %v", err)
	}
	if updated.SyncSchedule != "*/30 * * * *" {
		t.Fatalf("expected updated sync schedule, got %q", updated.SyncSchedule)
	}
	if updated.Name != "Original" || updated.BranchPattern != "release/*" {
		t.Fatalf("expected unrelated fields to remain unchanged, got name=%q branch=%q", updated.Name, updated.BranchPattern)
	}
	if updated.SourceTokenEnc != "source-old-token" || updated.TargetTokenEnc != "target-old-token" {
		t.Fatalf("expected tokens to remain unchanged, got %q %q", updated.SourceTokenEnc, updated.TargetTokenEnc)
	}
}

func TestUpdateMirrorScheduleHandlerClearsSchedule(t *testing.T) {
	userStore, mirrorStore, jobStore := newMirrorHandlerTestDeps(t)
	handler := &Handler{}
	router := NewRouter(handler, userStore, mirrorStore, jobStore)

	cfg := &models.MirrorConfig{
		UserID:           1,
		Name:             "Original",
		SourceOwner:      "source-org",
		SourceRepo:       "source-repo",
		SourceRepoURL:    "https://github.com/source-org/source-repo.git",
		TargetOwner:      "target-org",
		TargetRepo:       "target-repo",
		TargetRepoURL:    "https://github.com/target-org/target-repo.git",
		SyncSchedule:     "*/10 * * * *",
		SyncTags:         true,
		AllowForceUpdate: true,
		Enabled:          true,
	}
	if err := mirrorStore.CreateMirrorConfig(cfg); err != nil {
		t.Fatalf("create mirror config: %v", err)
	}

	form := url.Values{
		"sync_schedule": {""},
	}
	req := httptest.NewRequest(http.MethodPost, "/mirrors/1/schedule", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", basicAuthHeader("demo@example.com", "secret123"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d: %s", http.StatusSeeOther, rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Location"); got != "/mirrors/1?flash=Automatic+sync+schedule+cleared." {
		t.Fatalf("expected redirect location, got %q", got)
	}

	updated, err := mirrorStore.GetMirrorConfig(cfg.ID)
	if err != nil {
		t.Fatalf("get mirror config: %v", err)
	}
	if updated.SyncSchedule != "" {
		t.Fatalf("expected sync schedule to be cleared, got %q", updated.SyncSchedule)
	}
}

func TestUpdateMirrorScheduleHandlerRejectsInvalidCron(t *testing.T) {
	userStore, mirrorStore, jobStore := newMirrorHandlerTestDeps(t)
	handler := &Handler{}
	router := NewRouter(handler, userStore, mirrorStore, jobStore)

	cfg := &models.MirrorConfig{
		UserID:           1,
		Name:             "Original",
		SourceOwner:      "source-org",
		SourceRepo:       "source-repo",
		SourceRepoURL:    "https://github.com/source-org/source-repo.git",
		TargetOwner:      "target-org",
		TargetRepo:       "target-repo",
		TargetRepoURL:    "https://github.com/target-org/target-repo.git",
		BranchPattern:    "*",
		SyncSchedule:     "*/10 * * * *",
		SyncTags:         true,
		AllowForceUpdate: true,
		Enabled:          true,
	}
	if err := mirrorStore.CreateMirrorConfig(cfg); err != nil {
		t.Fatalf("create mirror config: %v", err)
	}

	form := url.Values{
		"sync_schedule": {"bad cron"},
	}
	req := httptest.NewRequest(http.MethodPost, "/mirrors/1/schedule", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", basicAuthHeader("demo@example.com", "secret123"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Current-URL", "http://example.com/mirrors/1/schedule")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Invalid cron schedule") {
		t.Fatalf("expected invalid cron message, got %q", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "bad cron") {
		t.Fatalf("expected submitted value to be preserved, got %q", rec.Body.String())
	}
	if got := rec.Header().Get("HX-Replace-Url"); got != "/mirrors/1/schedule" {
		t.Fatalf("expected HX-Replace-Url to keep schedule form URL, got %q", got)
	}

	updated, err := mirrorStore.GetMirrorConfig(cfg.ID)
	if err != nil {
		t.Fatalf("get mirror config: %v", err)
	}
	if updated.SyncSchedule != "*/10 * * * *" {
		t.Fatalf("expected existing sync schedule to remain unchanged, got %q", updated.SyncSchedule)
	}
}

func TestDeleteMirrorHandlerUsesHXLocationForBoostedForms(t *testing.T) {
	userStore, mirrorStore, jobStore := newMirrorHandlerTestDeps(t)
	handler := &Handler{}
	router := NewRouter(handler, userStore, mirrorStore, jobStore)

	cfg := &models.MirrorConfig{
		UserID:           1,
		Name:             "Original",
		SourceOwner:      "source-org",
		SourceRepo:       "source-repo",
		SourceRepoURL:    "https://github.com/source-org/source-repo.git",
		TargetOwner:      "target-org",
		TargetRepo:       "target-repo",
		TargetRepoURL:    "https://github.com/target-org/target-repo.git",
		SourceTokenEnc:   "source-old-token",
		TargetTokenEnc:   "target-old-token",
		BranchPattern:    "*",
		SyncTags:         true,
		AllowForceUpdate: true,
		Enabled:          true,
	}
	if err := mirrorStore.CreateMirrorConfig(cfg); err != nil {
		t.Fatalf("create mirror config: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/mirrors/1/delete", nil)
	req.Header.Set("Authorization", basicAuthHeader("demo@example.com", "secret123"))
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("HX-Location"); got != "/?flash=Mirror+deleted." {
		t.Fatalf("expected HX-Location header, got %q", got)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("expected empty response body for HX redirect, got %q", rec.Body.String())
	}
}

func TestSyncMirrorHandlerHTMXTriggersDashboardRefresh(t *testing.T) {
	userStore, mirrorStore, jobStore := newMirrorHandlerTestDeps(t)
	handler := &Handler{}
	router := NewRouter(handler, userStore, mirrorStore, jobStore)

	cfg := &models.MirrorConfig{
		UserID:           1,
		Name:             "Original",
		SourceOwner:      "source-org",
		SourceRepo:       "source-repo",
		SourceRepoURL:    "https://github.com/source-org/source-repo.git",
		TargetOwner:      "target-org",
		TargetRepo:       "target-repo",
		TargetRepoURL:    "https://github.com/target-org/target-repo.git",
		SourceTokenEnc:   "source-old-token",
		TargetTokenEnc:   "target-old-token",
		BranchPattern:    "*",
		SyncTags:         true,
		AllowForceUpdate: true,
		Enabled:          true,
	}
	if err := mirrorStore.CreateMirrorConfig(cfg); err != nil {
		t.Fatalf("create mirror config: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/mirrors/1/sync", nil)
	req.Header.Set("Authorization", basicAuthHeader("demo@example.com", "secret123"))
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Current-URL", "http://localhost:8080/")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("HX-Trigger"); got != "refresh-mirrors" {
		t.Fatalf("expected HX-Trigger refresh-mirrors, got %q", got)
	}
	if !strings.Contains(rec.Body.String(), "Sync job enqueued.") {
		t.Fatalf("expected flash body to mention queued sync, got %q", rec.Body.String())
	}
}

func TestSyncMirrorHandlerHTMXTriggersDetailRefresh(t *testing.T) {
	userStore, mirrorStore, jobStore := newMirrorHandlerTestDeps(t)
	handler := &Handler{}
	router := NewRouter(handler, userStore, mirrorStore, jobStore)

	cfg := &models.MirrorConfig{
		UserID:           1,
		Name:             "Original",
		SourceOwner:      "source-org",
		SourceRepo:       "source-repo",
		SourceRepoURL:    "https://github.com/source-org/source-repo.git",
		TargetOwner:      "target-org",
		TargetRepo:       "target-repo",
		TargetRepoURL:    "https://github.com/target-org/target-repo.git",
		SourceTokenEnc:   "source-old-token",
		TargetTokenEnc:   "target-old-token",
		BranchPattern:    "*",
		SyncTags:         true,
		AllowForceUpdate: true,
		Enabled:          true,
	}
	if err := mirrorStore.CreateMirrorConfig(cfg); err != nil {
		t.Fatalf("create mirror config: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/mirrors/1/sync", nil)
	req.Header.Set("Authorization", basicAuthHeader("demo@example.com", "secret123"))
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Current-URL", "http://localhost:8080/mirrors/1")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("HX-Trigger"); got != "refresh-mirrors, refresh-mirror-detail" {
		t.Fatalf("expected detail refresh trigger, got %q", got)
	}
}

func TestTestMirrorHandlerHTMLUsesErrorFlashForBrokenTokens(t *testing.T) {
	userStore, mirrorStore, jobStore := newMirrorHandlerTestDeps(t)
	handler := &Handler{}
	router := NewRouter(handler, userStore, mirrorStore, jobStore)

	cfg := &models.MirrorConfig{
		UserID:           1,
		Name:             "Original",
		SourceOwner:      "demo",
		SourceRepo:       "missing",
		SourceRepoURL:    "https://github.com/demo/missing.git",
		TargetOwner:      "demo",
		TargetRepo:       "read-only",
		TargetRepoURL:    "https://github.com/demo/read-only.git",
		SourceTokenEnc:   "token1234",
		TargetTokenEnc:   "token1234",
		BranchPattern:    "*",
		SyncTags:         true,
		AllowForceUpdate: true,
		Enabled:          true,
	}
	if err := mirrorStore.CreateMirrorConfig(cfg); err != nil {
		t.Fatalf("create mirror config: %v", err)
	}

	originalBaseURL := githubAPIBaseURL
	originalClient := githubAPIClient
	t.Cleanup(func() {
		githubAPIBaseURL = originalBaseURL
		githubAPIClient = originalClient
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/demo/missing":
			http.NotFound(w, r)
		case "/repos/demo/read-only":
			json.NewEncoder(w).Encode(map[string]any{
				"permissions": map[string]bool{"push": false},
			})
		default:
			http.Error(w, "nope", http.StatusUnauthorized)
		}
	}))
	defer server.Close()

	githubAPIBaseURL = server.URL
	githubAPIClient = server.Client()

	req := httptest.NewRequest(http.MethodPost, "/mirrors/1/test", nil)
	req.Header.Set("Authorization", basicAuthHeader("demo@example.com", "secret123"))
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Current-URL", "http://localhost:8080/mirrors/1")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "flash-error") {
		t.Fatalf("expected error flash class, got %q", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "repository not found") || !strings.Contains(rec.Body.String(), "read-only access") {
		t.Fatalf("expected humanized token statuses, got %q", rec.Body.String())
	}
}

func TestRetryMirrorHandlerHTMLShowsEmptyRetryState(t *testing.T) {
	userStore, mirrorStore, jobStore := newMirrorHandlerTestDeps(t)
	handler := &Handler{}
	router := NewRouter(handler, userStore, mirrorStore, jobStore)

	cfg := &models.MirrorConfig{
		UserID:           1,
		Name:             "Original",
		SourceOwner:      "source-org",
		SourceRepo:       "source-repo",
		SourceRepoURL:    "https://github.com/source-org/source-repo.git",
		TargetOwner:      "target-org",
		TargetRepo:       "target-repo",
		TargetRepoURL:    "https://github.com/target-org/target-repo.git",
		SourceTokenEnc:   "source-old-token",
		TargetTokenEnc:   "target-old-token",
		BranchPattern:    "*",
		SyncTags:         true,
		AllowForceUpdate: true,
		Enabled:          true,
	}
	if err := mirrorStore.CreateMirrorConfig(cfg); err != nil {
		t.Fatalf("create mirror config: %v", err)
	}

	job := &models.SyncJob{
		MirrorConfigID: cfg.ID,
		Ref:            "refs/heads/*",
		RefType:        "branch",
		BranchOrTag:    "*",
		Status:         "failed",
		Attempts:       3,
		MaxAttempts:    3,
	}
	if err := jobStore.CreateJob(job); err != nil {
		t.Fatalf("create sync job: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/mirrors/1/retry", nil)
	req.Header.Set("Authorization", basicAuthHeader("demo@example.com", "secret123"))
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Current-URL", "http://localhost:8080/mirrors/1")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "No retryable failed jobs were found.") {
		t.Fatalf("expected empty retry message, got %q", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "flash-error") {
		t.Fatalf("expected error flash class, got %q", rec.Body.String())
	}
}

func TestPartialsRenderUpdatedHTML(t *testing.T) {
	userStore, mirrorStore, jobStore := newMirrorHandlerTestDeps(t)
	handler := &Handler{}
	router := NewRouter(handler, userStore, mirrorStore, jobStore)

	cfg := &models.MirrorConfig{
		UserID:           1,
		Name:             "Mirror Partial",
		SourceOwner:      "source-org",
		SourceRepo:       "source-repo",
		SourceRepoURL:    "https://github.com/source-org/source-repo.git",
		TargetOwner:      "target-org",
		TargetRepo:       "target-repo",
		TargetRepoURL:    "https://github.com/target-org/target-repo.git",
		SourceTokenEnc:   "source-old-token",
		TargetTokenEnc:   "target-old-token",
		BranchPattern:    "*",
		SyncTags:         true,
		AllowForceUpdate: true,
		Enabled:          true,
	}
	if err := mirrorStore.CreateMirrorConfig(cfg); err != nil {
		t.Fatalf("create mirror config: %v", err)
	}

	job := &models.SyncJob{
		MirrorConfigID: cfg.ID,
		Ref:            "refs/heads/*",
		RefType:        "branch",
		BranchOrTag:    "*",
		Status:         "queued",
		MaxAttempts:    3,
	}
	if err := jobStore.CreateJob(job); err != nil {
		t.Fatalf("create sync job: %v", err)
	}

	dashboardReq := httptest.NewRequest(http.MethodGet, "/partials/dashboard/mirrors", nil)
	dashboardReq.Header.Set("Authorization", basicAuthHeader("demo@example.com", "secret123"))
	dashboardRec := httptest.NewRecorder()
	router.ServeHTTP(dashboardRec, dashboardReq)

	if dashboardRec.Code != http.StatusOK {
		t.Fatalf("expected dashboard partial status %d, got %d: %s", http.StatusOK, dashboardRec.Code, dashboardRec.Body.String())
	}
	if !strings.Contains(dashboardRec.Body.String(), "Mirror Partial") {
		t.Fatalf("expected dashboard partial to contain mirror name, got %q", dashboardRec.Body.String())
	}

	historyReq := httptest.NewRequest(http.MethodGet, "/partials/mirrors/1/history", nil)
	historyReq.Header.Set("Authorization", basicAuthHeader("demo@example.com", "secret123"))
	historyRec := httptest.NewRecorder()
	router.ServeHTTP(historyRec, historyReq)

	if historyRec.Code != http.StatusOK {
		t.Fatalf("expected history partial status %d, got %d: %s", http.StatusOK, historyRec.Code, historyRec.Body.String())
	}
	if !strings.Contains(historyRec.Body.String(), "queued") {
		t.Fatalf("expected history partial to contain queued status, got %q", historyRec.Body.String())
	}

	configReq := httptest.NewRequest(http.MethodGet, "/partials/mirrors/1/configuration", nil)
	configReq.Header.Set("Authorization", basicAuthHeader("demo@example.com", "secret123"))
	configRec := httptest.NewRecorder()
	cfg.SyncSchedule = "*/15 * * * *"
	if err := mirrorStore.UpdateMirrorConfig(cfg); err != nil {
		t.Fatalf("update mirror config: %v", err)
	}
	router.ServeHTTP(configRec, configReq)

	if configRec.Code != http.StatusOK {
		t.Fatalf("expected config partial status %d, got %d: %s", http.StatusOK, configRec.Code, configRec.Body.String())
	}
	if !strings.Contains(configRec.Body.String(), "*/15 * * * *") || !strings.Contains(configRec.Body.String(), "UTC") {
		t.Fatalf("expected config partial to show sync schedule, got %q", configRec.Body.String())
	}
}

func TestMirrorDetailPageShowsEditScheduleAction(t *testing.T) {
	userStore, mirrorStore, jobStore := newMirrorHandlerTestDeps(t)
	handler := &Handler{}
	router := NewRouter(handler, userStore, mirrorStore, jobStore)

	cfg := &models.MirrorConfig{
		UserID:           1,
		Name:             "Mirror Detail",
		SourceOwner:      "source-org",
		SourceRepo:       "source-repo",
		SourceRepoURL:    "https://github.com/source-org/source-repo.git",
		TargetOwner:      "target-org",
		TargetRepo:       "target-repo",
		TargetRepoURL:    "https://github.com/target-org/target-repo.git",
		SyncSchedule:     "0 * * * *",
		BranchPattern:    "*",
		SyncTags:         true,
		AllowForceUpdate: true,
		Enabled:          true,
	}
	if err := mirrorStore.CreateMirrorConfig(cfg); err != nil {
		t.Fatalf("create mirror config: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/mirrors/1", nil)
	req.Header.Set("Authorization", basicAuthHeader("demo@example.com", "secret123"))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "/mirrors/1/schedule") || !strings.Contains(rec.Body.String(), "Edit Schedule") {
		t.Fatalf("expected detail page to expose schedule action, got %q", rec.Body.String())
	}
}

func TestTestGitHubAccess(t *testing.T) {
	originalBaseURL := githubAPIBaseURL
	originalClient := githubAPIClient
	t.Cleanup(func() {
		githubAPIBaseURL = originalBaseURL
		githubAPIClient = originalClient
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/demo/source":
			json.NewEncoder(w).Encode(map[string]any{
				"permissions": map[string]bool{"push": false},
			})
		case "/repos/demo/target":
			json.NewEncoder(w).Encode(map[string]any{
				"permissions": map[string]bool{"push": true},
			})
		case "/repos/demo/read-only":
			json.NewEncoder(w).Encode(map[string]any{
				"permissions": map[string]bool{"push": false},
			})
		case "/repos/demo/missing":
			http.NotFound(w, r)
		default:
			http.Error(w, "nope", http.StatusUnauthorized)
		}
	}))
	defer server.Close()

	githubAPIBaseURL = server.URL
	githubAPIClient = server.Client()

	if got := testGitHubAccess("demo", "source", "token1234", false); got != "ok" {
		t.Fatalf("expected ok for readable source repo, got %q", got)
	}
	if got := testGitHubAccess("demo", "target", "token1234", true); got != "ok" {
		t.Fatalf("expected ok for writable target repo, got %q", got)
	}
	if got := testGitHubAccess("demo", "read-only", "token1234", true); got != "read_only" {
		t.Fatalf("expected read_only for non-push target repo, got %q", got)
	}
	if got := testGitHubAccess("demo", "missing", "token1234", false); got != "not_found" {
		t.Fatalf("expected not_found for missing repo, got %q", got)
	}
	if got := testGitHubAccess("demo", "source", "", false); got != "no_token" {
		t.Fatalf("expected no_token for empty token, got %q", got)
	}
	if got := testGitHubAccess("demo", "source", "abc", false); got != "invalid_token" {
		t.Fatalf("expected invalid_token for short token, got %q", got)
	}
}

func newMirrorHandlerTestDeps(t *testing.T) (auth.UserStore, *store.InMemoryMirrorConfigStore, store.SyncJobStore) {
	t.Helper()

	user := &models.User{
		ID:       1,
		Email:    "demo@example.com",
		FullName: "Demo User",
	}
	if err := user.SetPassword("secret123"); err != nil {
		t.Fatalf("set password: %v", err)
	}

	return auth.NewInMemoryUserStore([]*models.User{user}), store.NewInMemoryMirrorConfigStore(), store.NewInMemorySyncJobStore()
}

func basicAuthHeader(email, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(email+":"+password))
}
