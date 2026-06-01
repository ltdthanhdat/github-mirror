package http

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/dat-lt-amira/github-mirror/internal/auth"
	"github.com/dat-lt-amira/github-mirror/internal/mirror"
	"github.com/dat-lt-amira/github-mirror/internal/models"
	"github.com/dat-lt-amira/github-mirror/internal/schedule"
	"github.com/go-chi/chi/v5"
)

var githubAPIBaseURL = "https://api.github.com"

var githubAPIClient = &http.Client{Timeout: 5 * time.Second}

type mirrorFormRequest struct {
	Name             string `json:"name"`
	SourceURL        string `json:"source_url"`
	TargetURL        string `json:"target_url"`
	SourceToken      string `json:"source_token"`
	TargetToken      string `json:"target_token"`
	BranchPattern    string `json:"branch_pattern"`
	SyncSchedule     string `json:"sync_schedule"`
	SyncTags         *bool  `json:"sync_tags"`
	SyncDeletes      *bool  `json:"sync_deletes"`
	AllowForceUpdate *bool  `json:"allow_force_update"`
}

// ListMirrorsHandler returns all mirror configurations for the authenticated user.
func (h *Handler) ListMirrorsHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.AuthenticatedUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	configs, err := h.MirrorStore.ListMirrorConfigsByUser(user.ID)
	if err != nil {
		http.Error(w, "Failed to list mirrors", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(configs)
}

// CreateMirrorHandler creates a new mirror configuration.
func (h *Handler) CreateMirrorHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.AuthenticatedUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req mirrorFormRequest

	switch {
	case isJSONRequest(r):
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
	default:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}
		req.Name = r.FormValue("name")
		req.SourceURL = r.FormValue("source_url")
		req.TargetURL = r.FormValue("target_url")
		req.SourceToken = r.FormValue("source_token")
		req.TargetToken = r.FormValue("target_token")
		req.BranchPattern = r.FormValue("branch_pattern")
		req.SyncSchedule = r.FormValue("sync_schedule")
		req.SyncTags = boolPtr(r.Form.Has("sync_tags"))
		req.SyncDeletes = boolPtr(r.Form.Has("sync_deletes"))
		req.AllowForceUpdate = boolPtr(r.Form.Has("allow_force_update"))
	}

	if req.Name == "" || req.SourceURL == "" || req.TargetURL == "" {
		if renderMirrorFormErrorIfHTML(w, r, h, newMirrorFormData(req), "Missing required fields") {
			return
		}
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}
	if err := validateMirrorSyncSchedule(req.SyncSchedule); err != nil {
		if renderMirrorFormErrorIfHTML(w, r, h, newMirrorFormData(req), err.Error()) {
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sourceOwner, sourceRepo, sourceRepoURL, err := parseGitHubRepoURL(req.SourceURL)
	if err != nil {
		if renderMirrorFormErrorIfHTML(w, r, h, newMirrorFormData(req), "Invalid source repository URL") {
			return
		}
		http.Error(w, "Invalid source repository URL", http.StatusBadRequest)
		return
	}
	targetOwner, targetRepo, targetRepoURL, err := parseGitHubRepoURL(req.TargetURL)
	if err != nil {
		if renderMirrorFormErrorIfHTML(w, r, h, newMirrorFormData(req), "Invalid target repository URL") {
			return
		}
		http.Error(w, "Invalid target repository URL", http.StatusBadRequest)
		return
	}

	// TODO: Encrypt tokens with APP_ENCRYPTION_KEY
	// For now, store as-is (will be encrypted when encryption key is configured)
	sourceTokenEnc := req.SourceToken
	targetTokenEnc := req.TargetToken

	cfg := &models.MirrorConfig{
		UserID:           user.ID,
		Name:             req.Name,
		SourceOwner:      sourceOwner,
		SourceRepo:       sourceRepo,
		SourceRepoURL:    sourceRepoURL,
		TargetOwner:      targetOwner,
		TargetRepo:       targetRepo,
		TargetRepoURL:    targetRepoURL,
		SourceTokenEnc:   sourceTokenEnc,
		TargetTokenEnc:   targetTokenEnc,
		BranchPattern:    req.BranchPattern,
		SyncTags:         true,
		SyncDeletes:      false,
		AllowForceUpdate: true,
		SyncSchedule:     strings.TrimSpace(req.SyncSchedule),
		Enabled:          true,
	}

	if req.SyncTags != nil {
		cfg.SyncTags = *req.SyncTags
	}
	if req.SyncDeletes != nil {
		cfg.SyncDeletes = *req.SyncDeletes
	}
	if req.AllowForceUpdate != nil {
		cfg.AllowForceUpdate = *req.AllowForceUpdate
	}
	if cfg.BranchPattern == "" {
		cfg.BranchPattern = "*"
	}

	if err := h.MirrorStore.CreateMirrorConfig(cfg); err != nil {
		http.Error(w, "Failed to create mirror", http.StatusInternalServerError)
		return
	}

	// Enqueue initial sync job
	initialJob := enqueueFullSyncJob(cfg.ID)
	h.JobStore.CreateJob(initialJob)
	log.Printf("Initial sync job enqueued for mirror %d", cfg.ID)

	if isHTMLRequest(r) {
		navigateOrRedirectHTMX(w, r, fmt.Sprintf("/mirrors/%d", cfg.ID), flashHTML("Mirror created and initial sync queued.", false))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(cfg)
}

// UpdateMirrorHandler updates an existing mirror configuration.
func (h *Handler) UpdateMirrorHandler(w http.ResponseWriter, r *http.Request) {
	cfg, err := getMirrorFromRequest(r, h)
	if err != nil {
		http.Error(w, err.Error(), errCode(err))
		return
	}

	var req mirrorFormRequest

	switch {
	case isJSONRequest(r):
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
	default:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}
		req.Name = r.FormValue("name")
		req.SourceURL = r.FormValue("source_url")
		req.TargetURL = r.FormValue("target_url")
		req.SourceToken = r.FormValue("source_token")
		req.TargetToken = r.FormValue("target_token")
		req.BranchPattern = r.FormValue("branch_pattern")
		req.SyncSchedule = r.FormValue("sync_schedule")
		req.SyncTags = boolPtr(r.Form.Has("sync_tags"))
		req.SyncDeletes = boolPtr(r.Form.Has("sync_deletes"))
		req.AllowForceUpdate = boolPtr(r.Form.Has("allow_force_update"))
	}

	if req.Name == "" || req.SourceURL == "" || req.TargetURL == "" {
		if renderMirrorFormErrorIfHTML(w, r, h, editMirrorFormData(cfg, req), "Missing required fields") {
			return
		}
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}
	if err := validateMirrorSyncSchedule(req.SyncSchedule); err != nil {
		if renderMirrorFormErrorIfHTML(w, r, h, editMirrorFormData(cfg, req), err.Error()) {
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sourceOwner, sourceRepo, sourceRepoURL, err := parseGitHubRepoURL(req.SourceURL)
	if err != nil {
		if renderMirrorFormErrorIfHTML(w, r, h, editMirrorFormData(cfg, req), "Invalid source repository URL") {
			return
		}
		http.Error(w, "Invalid source repository URL", http.StatusBadRequest)
		return
	}
	targetOwner, targetRepo, targetRepoURL, err := parseGitHubRepoURL(req.TargetURL)
	if err != nil {
		if renderMirrorFormErrorIfHTML(w, r, h, editMirrorFormData(cfg, req), "Invalid target repository URL") {
			return
		}
		http.Error(w, "Invalid target repository URL", http.StatusBadRequest)
		return
	}

	cfg.Name = req.Name
	cfg.SourceOwner = sourceOwner
	cfg.SourceRepo = sourceRepo
	cfg.SourceRepoURL = sourceRepoURL
	cfg.TargetOwner = targetOwner
	cfg.TargetRepo = targetRepo
	cfg.TargetRepoURL = targetRepoURL
	cfg.BranchPattern = req.BranchPattern
	cfg.SyncSchedule = strings.TrimSpace(req.SyncSchedule)
	if cfg.BranchPattern == "" {
		cfg.BranchPattern = "*"
	}
	if req.SourceToken != "" {
		cfg.SourceTokenEnc = req.SourceToken
	}
	if req.TargetToken != "" {
		cfg.TargetTokenEnc = req.TargetToken
	}
	if req.SyncTags != nil {
		cfg.SyncTags = *req.SyncTags
	}
	if req.SyncDeletes != nil {
		cfg.SyncDeletes = *req.SyncDeletes
	}
	if req.AllowForceUpdate != nil {
		cfg.AllowForceUpdate = *req.AllowForceUpdate
	}

	if err := h.MirrorStore.UpdateMirrorConfig(cfg); err != nil {
		http.Error(w, "Failed to update mirror", http.StatusInternalServerError)
		return
	}

	if isHTMLRequest(r) {
		navigateOrRedirectHTMX(w, r, fmt.Sprintf("/mirrors/%d", cfg.ID), flashHTML("Mirror updated.", false))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}

// UpdateMirrorScheduleHandler updates only the persisted mirror schedule.
func (h *Handler) UpdateMirrorScheduleHandler(w http.ResponseWriter, r *http.Request) {
	cfg, err := getMirrorFromRequest(r, h)
	if err != nil {
		http.Error(w, err.Error(), errCode(err))
		return
	}

	rawSchedule := ""
	switch {
	case isJSONRequest(r):
		var req struct {
			SyncSchedule string `json:"sync_schedule"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		rawSchedule = req.SyncSchedule
	default:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}
		rawSchedule = r.FormValue("sync_schedule")
	}

	if err := validateMirrorSyncSchedule(rawSchedule); err != nil {
		if renderMirrorScheduleErrorIfHTML(w, r, h, cfg, rawSchedule, err.Error()) {
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cfg.SyncSchedule = strings.TrimSpace(rawSchedule)
	if err := h.MirrorStore.UpdateMirrorConfig(cfg); err != nil {
		http.Error(w, "Failed to update mirror schedule", http.StatusInternalServerError)
		return
	}

	message := "Automatic sync schedule cleared."
	if cfg.SyncSchedule != "" {
		message = "Automatic sync schedule updated."
	}

	if isHTMLRequest(r) {
		navigateOrRedirectHTMX(w, r, fmt.Sprintf("/mirrors/%d", cfg.ID), flashHTML(message, false))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"sync_schedule": cfg.SyncSchedule,
		"message":       message,
	})
}

// GetMirrorHandler returns a specific mirror configuration.
func (h *Handler) GetMirrorHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.AuthenticatedUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid mirror ID", http.StatusBadRequest)
		return
	}

	cfg, err := h.MirrorStore.GetMirrorConfig(id)
	if err != nil {
		http.Error(w, "Mirror not found", http.StatusNotFound)
		return
	}

	if cfg.UserID != user.ID && !user.IsAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Mask tokens in response
	cfg.SourceTokenEnc = maskToken(cfg.SourceTokenEnc)
	cfg.TargetTokenEnc = maskToken(cfg.TargetTokenEnc)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}

// DeleteMirrorHandler deletes a mirror configuration.
func (h *Handler) DeleteMirrorHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.AuthenticatedUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid mirror ID", http.StatusBadRequest)
		return
	}

	cfg, err := h.MirrorStore.GetMirrorConfig(id)
	if err != nil {
		http.Error(w, "Mirror not found", http.StatusNotFound)
		return
	}

	if cfg.UserID != user.ID && !user.IsAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := h.MirrorStore.DeleteMirrorConfig(id); err != nil {
		http.Error(w, "Failed to delete mirror", http.StatusInternalServerError)
		return
	}

	if isHTMLRequest(r) {
		navigateOrRedirectHTMX(w, r, "/", flashHTML("Mirror deleted.", false))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Mirror deleted"})
}

// WebhookHandler receives GitHub webhook events.
func (h *Handler) WebhookHandler(w http.ResponseWriter, r *http.Request) {
	body, err := mirror.ReadBody(r)
	if err != nil {
		log.Printf("Failed to read webhook body: %v", err)
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	signature := r.Header.Get("X-Hub-Signature-256")
	if !mirror.VerifyGitHubSignature(signature, body, "") {
		log.Printf("Invalid webhook signature: %s", signature)
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	event := r.Header.Get("X-GitHub-Event")
	if event != "push" {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	payload, err := mirror.ParsePushPayload(body)
	if err != nil {
		log.Printf("Failed to parse push payload: %v", err)
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	refType, name := mirror.ParseRef(payload.Ref)
	log.Printf("Received push event: %s %s", refType, name)

	w.WriteHeader(http.StatusAccepted)
}

// TestMirrorHandler tests source and target tokens by calling the GitHub API.
func (h *Handler) TestMirrorHandler(w http.ResponseWriter, r *http.Request) {
	cfg, err := getMirrorFromRequest(r, h)
	if err != nil {
		http.Error(w, err.Error(), errCode(err))
		return
	}

	// Test source token by making a lightweight request to GitHub
	sourceStatus := testGitHubAccess(cfg.SourceOwner, cfg.SourceRepo, cfg.SourceTokenEnc, false)
	targetStatus := testGitHubAccess(cfg.TargetOwner, cfg.TargetRepo, cfg.TargetTokenEnc, true)

	resp := map[string]string{
		"source": sourceStatus,
		"target": targetStatus,
	}

	if isHTMLRequest(r) {
		isError := sourceStatus != "ok" || targetStatus != "ok"
		message := fmt.Sprintf(
			"Token test complete. Source: %s. Target: %s.",
			humanizeTokenStatus(sourceStatus),
			humanizeTokenStatus(targetStatus),
		)
		triggerMirrorRefresh(w, r, cfg.ID)
		redirectOrHTMX(w, r, fmt.Sprintf("/mirrors/%d", cfg.ID), flashHTML(message, isError))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// RetryMirrorHandler retries failed jobs for a mirror.
func (h *Handler) RetryMirrorHandler(w http.ResponseWriter, r *http.Request) {
	cfg, err := getMirrorFromRequest(r, h)
	if err != nil {
		http.Error(w, err.Error(), errCode(err))
		return
	}

	// Find failed jobs for this mirror and reset them to retrying
	jobs, err := h.JobStore.ListJobsByMirrorConfig(cfg.ID)
	if err != nil {
		http.Error(w, "Failed to list jobs", http.StatusInternalServerError)
		return
	}

	retried := 0
	for _, job := range jobs {
		if job.Status == "failed" && job.Attempts < job.MaxAttempts {
			job.Status = "retrying"
			job.LastError = ""
			h.JobStore.UpdateJob(job)
			retried++
		}
	}

	if isHTMLRequest(r) {
		triggerMirrorRefresh(w, r, cfg.ID)
		if retried == 0 {
			redirectOrHTMX(w, r, fmt.Sprintf("/mirrors/%d", cfg.ID), flashHTML("No retryable failed jobs were found.", true))
			return
		}
		redirectOrHTMX(w, r, fmt.Sprintf("/mirrors/%d", cfg.ID), flashHTML(fmt.Sprintf("Retry initiated for %d failed job(s).", retried), false))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Retry initiated",
		"retried": retried,
	})
}

// SyncMirrorHandler triggers a manual sync for a mirror.
func (h *Handler) SyncMirrorHandler(w http.ResponseWriter, r *http.Request) {
	cfg, err := getMirrorFromRequest(r, h)
	if err != nil {
		http.Error(w, err.Error(), errCode(err))
		return
	}

	// Enqueue a sync job that mirrors all branches
	job := enqueueFullSyncJob(cfg.ID)

	if err := h.JobStore.CreateJob(job); err != nil {
		http.Error(w, "Failed to enqueue sync job", http.StatusInternalServerError)
		return
	}

	if isHTMLRequest(r) {
		triggerMirrorRefresh(w, r, cfg.ID)
		redirectOrHTMX(w, r, fmt.Sprintf("/mirrors/%d", cfg.ID), flashHTML("Sync job enqueued.", false))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Sync job enqueued",
	})
}

// HealthHandler returns service health status.
func (h *Handler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// FaviconHandler avoids noisy 404s for browsers requesting /favicon.ico.
func (h *Handler) FaviconHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

// getMirrorFromRequest extracts the authenticated user and mirror config from a request.
func getMirrorFromRequest(r *http.Request, h *Handler) (*models.MirrorConfig, error) {
	user := auth.AuthenticatedUser(r)
	if user == nil {
		return nil, authErr("Unauthorized")
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return nil, authErr("Invalid mirror ID")
	}

	cfg, err := h.MirrorStore.GetMirrorConfig(id)
	if err != nil {
		return nil, authErr("Mirror not found")
	}

	if cfg.UserID != user.ID && !user.IsAdmin {
		return nil, authErr("Forbidden")
	}

	return cfg, nil
}

// errCode returns a corresponding HTTP status code for common errors.
func errCode(err error) int {
	switch err.Error() {
	case "Unauthorized":
		return http.StatusUnauthorized
	case "Forbidden":
		return http.StatusForbidden
	case "Mirror not found":
		return http.StatusNotFound
	case "Invalid mirror ID":
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// authErr creates a simple error for auth failures.
type authErr string

func (e authErr) Error() string { return string(e) }

// testGitHubAccess tests if a token can read a repository and, optionally, push to it.
func testGitHubAccess(owner, repo, token string, requirePush bool) string {
	if token == "" {
		return "no_token"
	}
	if len(token) < 4 {
		return "invalid_token"
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/repos/%s/%s", githubAPIBaseURL, owner, repo), nil)
	if err != nil {
		return "request_failed"
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := githubAPIClient.Do(req)
	if err != nil {
		return "request_failed"
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	default:
		return fmt.Sprintf("github_error_%d", resp.StatusCode)
	}

	if !requirePush {
		return "ok"
	}

	var payload struct {
		Permissions struct {
			Push bool `json:"push"`
		} `json:"permissions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "ok"
	}
	if !payload.Permissions.Push {
		return "read_only"
	}
	return "ok"
}

func humanizeTokenStatus(status string) string {
	switch status {
	case "ok":
		return "OK"
	case "no_token":
		return "missing token"
	case "invalid_token":
		return "invalid token format"
	case "unauthorized":
		return "unauthorized"
	case "forbidden":
		return "forbidden"
	case "not_found":
		return "repository not found"
	case "read_only":
		return "read-only access"
	case "request_failed":
		return "request failed"
	default:
		if strings.HasPrefix(status, "github_error_") {
			return strings.ReplaceAll(status, "_", " ")
		}
		return status
	}
}

// maskToken returns a masked version of a token for API responses.
func maskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "****" + token[len(token)-4:]
}

func isJSONRequest(r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	return contentType == "application/json" || contentType == "application/json; charset=utf-8"
}

func isHTMLRequest(r *http.Request) bool {
	return r.Method != http.MethodGet && !isJSONRequest(r)
}

func boolPtr(v bool) *bool {
	return &v
}

func parseGitHubRepoURL(raw string) (owner string, repo string, normalized string, err error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", "", "", err
	}

	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return "", "", "", fmt.Errorf("unsupported scheme")
	}

	host := parsed.Hostname()
	if host != "github.com" && host != "www.github.com" {
		return "", "", "", fmt.Errorf("unsupported host")
	}

	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", "", "", fmt.Errorf("unsupported URL suffix")
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("invalid repository path")
	}

	owner = parts[0]
	repo = strings.TrimSuffix(parts[1], ".git")
	if owner == "" || repo == "" {
		return "", "", "", fmt.Errorf("invalid repository path")
	}

	normalized = "https://github.com/" + owner + "/" + repo + ".git"
	return owner, repo, normalized, nil
}

func redirectOrHTMX(w http.ResponseWriter, r *http.Request, location, htmxBody string) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(htmxBody))
		return
	}

	target := location
	if htmxBody != "" {
		target = location + "?flash=" + url.QueryEscape(stripFlashHTML(htmxBody))
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func navigateOrRedirectHTMX(w http.ResponseWriter, r *http.Request, location, htmxBody string) {
	target := location
	if htmxBody != "" {
		target = location + "?flash=" + url.QueryEscape(stripFlashHTML(htmxBody))
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Location", target)
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, target, http.StatusSeeOther)
}

func flashHTML(message string, isError bool) string {
	className := "flash-success"
	if isError {
		className = "flash-error"
	}
	return fmt.Sprintf(`<div class="flash %s">%s</div>`, className, html.EscapeString(message))
}

func stripFlashHTML(body string) string {
	start := len(`<div class="flash flash-success">`)
	if start >= len(body) {
		return body
	}
	end := len(body) - len(`</div>`)
	if end <= start {
		return body
	}
	return body[start:end]
}

func triggerMirrorRefresh(w http.ResponseWriter, r *http.Request, mirrorID uint64) {
	if r.Header.Get("HX-Request") != "true" {
		return
	}

	events := []string{"refresh-mirrors"}
	if currentHTMXPath(r) == fmt.Sprintf("/mirrors/%d", mirrorID) {
		events = append(events, "refresh-mirror-detail")
	}

	w.Header().Set("HX-Trigger", strings.Join(events, ", "))
}

func currentHTMXPath(r *http.Request) string {
	raw := r.Header.Get("HX-Current-URL")
	if raw == "" {
		raw = r.Referer()
	}
	if raw == "" {
		return ""
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Path == "" {
		return ""
	}

	return path.Clean(parsed.Path)
}

func renderMirrorFormErrorIfHTML(w http.ResponseWriter, r *http.Request, h *Handler, data map[string]interface{}, message string) bool {
	if !isHTMLRequest(r) || h.UIRenderer == nil {
		return false
	}

	data["ErrorFlash"] = message
	if currentPath := currentHTMXPath(r); currentPath != "" {
		w.Header().Set("HX-Replace-Url", currentPath)
	}
	h.UIRenderer.RenderMirrorFormPage(w, data)
	return true
}

func renderMirrorScheduleErrorIfHTML(w http.ResponseWriter, r *http.Request, h *Handler, cfg *models.MirrorConfig, rawSchedule, message string) bool {
	if !isHTMLRequest(r) || h.UIRenderer == nil {
		return false
	}

	data := mirrorScheduleFormData(cfg, strings.TrimSpace(rawSchedule))
	data["ErrorFlash"] = message
	if currentPath := currentHTMXPath(r); currentPath != "" {
		w.Header().Set("HX-Replace-Url", currentPath)
	}
	h.UIRenderer.RenderMirrorSchedulePage(w, data)
	return true
}

func newMirrorFormData(req mirrorFormRequest) map[string]interface{} {
	syncTags := true
	if req.SyncTags != nil {
		syncTags = *req.SyncTags
	}

	syncDeletes := false
	if req.SyncDeletes != nil {
		syncDeletes = *req.SyncDeletes
	}

	allowForceUpdate := true
	if req.AllowForceUpdate != nil {
		allowForceUpdate = *req.AllowForceUpdate
	}

	branchPattern := req.BranchPattern
	if branchPattern == "" {
		branchPattern = "*"
	}

	return map[string]interface{}{
		"PageTitle":        "New Mirror",
		"FormAction":       "/mirrors",
		"FormTitle":        "Create Mirror Configuration",
		"FormDescription":  "Define the source and target repositories, choose sync behavior, and queue the initial mirror sync in one step.",
		"SubmitLabel":      "Create Mirror",
		"Name":             req.Name,
		"SourceURL":        req.SourceURL,
		"TargetURL":        req.TargetURL,
		"BranchPattern":    branchPattern,
		"SyncSchedule":     req.SyncSchedule,
		"SyncTags":         syncTags,
		"SyncDeletes":      syncDeletes,
		"AllowForceUpdate": allowForceUpdate,
	}
}

func editMirrorFormData(cfg *models.MirrorConfig, req mirrorFormRequest) map[string]interface{} {
	data := newMirrorFormData(req)
	data["PageTitle"] = "Edit Mirror"
	data["FormAction"] = fmt.Sprintf("/mirrors/%d", cfg.ID)
	data["FormTitle"] = "Edit Mirror Configuration"
	data["FormDescription"] = "Update repository settings, replace tokens if needed, and keep the existing tokens by leaving those fields blank."
	data["SubmitLabel"] = "Update Mirror"
	data["IsEdit"] = true
	return data
}

func mirrorScheduleFormData(cfg *models.MirrorConfig, scheduleValue string) map[string]interface{} {
	return map[string]interface{}{
		"PageTitle":       "Edit Schedule",
		"FormAction":      fmt.Sprintf("/mirrors/%d/schedule", cfg.ID),
		"MirrorID":        cfg.ID,
		"MirrorName":      cfg.Name,
		"SyncSchedule":    strings.TrimSpace(scheduleValue),
		"SubmitLabel":     "Save Schedule",
		"BackURL":         fmt.Sprintf("/mirrors/%d", cfg.ID),
		"FormTitle":       "Manage Cron Schedule",
		"FormDescription": "Update or clear the automatic sync schedule for this mirror. Schedules use standard 5-field cron in UTC.",
	}
}

func validateMirrorSyncSchedule(raw string) error {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	if err := schedule.ValidateCron(value); err != nil {
		return fmt.Errorf("Invalid cron schedule")
	}
	return nil
}

func enqueueFullSyncJob(mirrorConfigID uint64) *models.SyncJob {
	return &models.SyncJob{
		MirrorConfigID: mirrorConfigID,
		Ref:            "refs/heads/*",
		RefType:        "branch",
		BranchOrTag:    "*",
		AfterSHA:       "0000000",
		Deleted:        false,
		Status:         "queued",
		Attempts:       0,
		MaxAttempts:    3,
		CreatedAt:      time.Now(),
	}
}
