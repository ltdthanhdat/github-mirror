package ui

import (
	"bytes"
	"errors"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/dat-lt-amira/github-mirror/internal/auth"
	"github.com/dat-lt-amira/github-mirror/internal/models"
	"github.com/dat-lt-amira/github-mirror/internal/store"
	"github.com/go-chi/chi/v5"
)

// Handler serves UI pages.
type Handler struct {
	MirrorStore store.MirrorConfigStore
	JobStore    store.SyncJobStore
	templates   map[string]*template.Template
}

// NewHandler creates a new UI handler and parses templates.
func NewHandler(mirrorStore store.MirrorConfigStore, jobStore store.SyncJobStore) *Handler {
	return &Handler{
		MirrorStore: mirrorStore,
		JobStore:    jobStore,
		templates:   loadTemplates(),
	}
}

func loadTemplates() map[string]*template.Template {
	funcs := template.FuncMap{
		"eqStr": func(a, b string) bool { return a == b },
	}
	templateDir := templatesDir()
	layoutPath := filepath.Join(templateDir, "layout.html")
	partialsPath := filepath.Join(templateDir, "partials.html")
	pageFiles := []string{"dashboard.html", "mirror_form.html", "mirror_detail.html", "setup_guide.html"}
	templates := make(map[string]*template.Template, len(pageFiles))

	for _, pageFile := range pageFiles {
		pagePath := filepath.Join(templateDir, pageFile)
		tmpl, err := template.New("layout.html").Funcs(funcs).ParseFiles(layoutPath, partialsPath, pagePath)
		if err != nil {
			log.Printf("UI: Failed to parse template set %s: %v", pageFile, err)
			continue
		}
		templates[pageFile] = tmpl
	}

	return templates
}

func templatesDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Join("internal", "ui", "templates")
	}
	return filepath.Join(filepath.Dir(filename), "templates")
}

func (h *Handler) render(w http.ResponseWriter, templateName string, data map[string]interface{}) {
	tmpl, ok := h.templates[templateName]
	if !ok {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("UI: Failed to render %s: %v", templateName, err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

// RenderMirrorFormPage renders the mirror form template for create/edit flows.
func (h *Handler) RenderMirrorFormPage(w http.ResponseWriter, data map[string]interface{}) {
	h.render(w, "mirror_form.html", data)
}

func (h *Handler) renderFragment(w http.ResponseWriter, templateName, fragmentName string, data map[string]interface{}) {
	html, err := h.RenderFragment(templateName, fragmentName, data)
	if err != nil {
		log.Printf("UI: Failed to render fragment %s/%s: %v", templateName, fragmentName, err)
		http.Error(w, "Failed to render fragment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// RenderFragment renders a named template fragment to a string.
func (h *Handler) RenderFragment(templateName, fragmentName string, data map[string]interface{}) (string, error) {
	tmpl, ok := h.templates[templateName]
	if !ok {
		return "", http.ErrMissingFile
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, fragmentName, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// Dashboard renders the main dashboard page.
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	user := auth.AuthenticatedUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	mirrors, err := h.MirrorStore.ListMirrorConfigsByUser(user.ID)
	if err != nil {
		http.Error(w, "Failed to list mirrors", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Mirrors": mirrors,
	}

	data["Flash"] = r.URL.Query().Get("flash")
	h.render(w, "dashboard.html", data)
}

// DashboardMirrorsPartial renders the auto-refreshing mirror list fragment.
func (h *Handler) DashboardMirrorsPartial(w http.ResponseWriter, r *http.Request) {
	user := auth.AuthenticatedUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	mirrors, err := h.MirrorStore.ListMirrorConfigsByUser(user.ID)
	if err != nil {
		http.Error(w, "Failed to list mirrors", http.StatusInternalServerError)
		return
	}

	h.renderFragment(w, "dashboard.html", "dashboard_mirror_list", map[string]interface{}{
		"Mirrors": mirrors,
	})
}

// NewMirrorForm renders the mirror creation form.
func (h *Handler) NewMirrorForm(w http.ResponseWriter, r *http.Request) {
	user := auth.AuthenticatedUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	data := map[string]interface{}{
		"PageTitle":        "New Mirror",
		"FormAction":       "/mirrors",
		"FormTitle":        "Create Mirror Configuration",
		"FormDescription":  "Define the source and target repositories, choose sync behavior, and queue the initial mirror sync in one step.",
		"SubmitLabel":      "Create Mirror",
		"BranchPattern":    "*",
		"SyncSchedule":     "",
		"SyncTags":         true,
		"AllowForceUpdate": true,
	}

	data["Flash"] = r.URL.Query().Get("flash")
	h.render(w, "mirror_form.html", data)
}

// EditMirrorForm renders the mirror edit form.
func (h *Handler) EditMirrorForm(w http.ResponseWriter, r *http.Request) {
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

	data := map[string]interface{}{
		"PageTitle":        "Edit Mirror",
		"FormAction":       "/mirrors/" + idStr,
		"FormTitle":        "Edit Mirror Configuration",
		"FormDescription":  "Update repository settings, replace tokens if needed, and keep the existing tokens by leaving those fields blank.",
		"SubmitLabel":      "Update Mirror",
		"Name":             cfg.Name,
		"SourceURL":        cfg.SourceRepoURL,
		"TargetURL":        cfg.TargetRepoURL,
		"BranchPattern":    cfg.BranchPattern,
		"SyncSchedule":     cfg.SyncSchedule,
		"SyncTags":         cfg.SyncTags,
		"SyncDeletes":      cfg.SyncDeletes,
		"AllowForceUpdate": cfg.AllowForceUpdate,
		"IsEdit":           true,
		"Flash":            r.URL.Query().Get("flash"),
	}

	h.render(w, "mirror_form.html", data)
}

// SetupGuide renders the PAT token and webhook setup guide.
func (h *Handler) SetupGuide(w http.ResponseWriter, r *http.Request) {
	user := auth.AuthenticatedUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	data := map[string]interface{}{
		"Flash": r.URL.Query().Get("flash"),
	}
	h.render(w, "setup_guide.html", data)
}

// MirrorDetail renders the mirror configuration detail page.
func (h *Handler) MirrorDetail(w http.ResponseWriter, r *http.Request) {
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

	// Get sync jobs for this mirror
	jobs, err := h.JobStore.ListJobsByMirrorConfig(id)
	if err != nil {
		jobs = []*models.SyncJob{}
	}
	if jobs == nil {
		jobs = []*models.SyncJob{}
	}

	// Mask tokens
	cfg.SourceTokenEnc = "****"
	cfg.TargetTokenEnc = "****"

	data := map[string]interface{}{
		"Name":             cfg.Name,
		"ID":               cfg.ID,
		"SourceOwner":      cfg.SourceOwner,
		"SourceRepo":       cfg.SourceRepo,
		"TargetOwner":      cfg.TargetOwner,
		"TargetRepo":       cfg.TargetRepo,
		"BranchPattern":    cfg.BranchPattern,
		"SyncSchedule":     cfg.SyncSchedule,
		"SyncTags":         cfg.SyncTags,
		"SyncDeletes":      cfg.SyncDeletes,
		"AllowForceUpdate": cfg.AllowForceUpdate,
		"Enabled":          cfg.Enabled,
		"LastSyncedAt":     cfg.LastSyncedAt,
		"CreatedAt":        cfg.CreatedAt,
		"Jobs":             jobs,
		"Flash":            r.URL.Query().Get("flash"),
	}

	h.render(w, "mirror_detail.html", data)
}

// MirrorConfigurationPartial renders the detail summary card for HTMX refreshes.
func (h *Handler) MirrorConfigurationPartial(w http.ResponseWriter, r *http.Request) {
	data, err := h.mirrorDetailData(r)
	if err != nil {
		if handlePartialError(w, r, err) {
			return
		}
		http.Error(w, err.Error(), uiErrCode(err))
		return
	}

	h.renderFragment(w, "mirror_detail.html", "mirror_detail_configuration", data)
}

// MirrorHistoryPartial renders the job history card for HTMX refreshes.
func (h *Handler) MirrorHistoryPartial(w http.ResponseWriter, r *http.Request) {
	data, err := h.mirrorDetailData(r)
	if err != nil {
		if handlePartialError(w, r, err) {
			return
		}
		http.Error(w, err.Error(), uiErrCode(err))
		return
	}

	h.renderFragment(w, "mirror_detail.html", "mirror_detail_history", data)
}

func (h *Handler) mirrorDetailData(r *http.Request) (map[string]interface{}, error) {
	user := auth.AuthenticatedUser(r)
	if user == nil {
		return nil, errors.New("Unauthorized")
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return nil, errors.New("Invalid mirror ID")
	}

	cfg, err := h.MirrorStore.GetMirrorConfig(id)
	if err != nil {
		return nil, errors.New("Mirror not found")
	}

	if cfg.UserID != user.ID && !user.IsAdmin {
		return nil, errors.New("Forbidden")
	}

	jobs, err := h.JobStore.ListJobsByMirrorConfig(id)
	if err != nil || jobs == nil {
		jobs = []*models.SyncJob{}
	}

	return map[string]interface{}{
		"Name":             cfg.Name,
		"ID":               cfg.ID,
		"SourceOwner":      cfg.SourceOwner,
		"SourceRepo":       cfg.SourceRepo,
		"TargetOwner":      cfg.TargetOwner,
		"TargetRepo":       cfg.TargetRepo,
		"BranchPattern":    cfg.BranchPattern,
		"SyncSchedule":     cfg.SyncSchedule,
		"SyncTags":         cfg.SyncTags,
		"SyncDeletes":      cfg.SyncDeletes,
		"AllowForceUpdate": cfg.AllowForceUpdate,
		"Enabled":          cfg.Enabled,
		"LastSyncedAt":     cfg.LastSyncedAt,
		"CreatedAt":        cfg.CreatedAt,
		"Jobs":             jobs,
	}, nil
}

func uiErrCode(err error) int {
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

func handlePartialError(w http.ResponseWriter, r *http.Request, err error) bool {
	if r.Header.Get("HX-Request") != "true" {
		return false
	}

	switch err.Error() {
	case "Mirror not found":
		w.WriteHeader(http.StatusNoContent)
		return true
	case "Forbidden", "Unauthorized":
		w.Header().Set("HX-Redirect", "/")
		w.WriteHeader(http.StatusOK)
		return true
	default:
		return false
	}
}
