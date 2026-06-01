package http

import (
	"github.com/dat-lt-amira/github-mirror/internal/auth"
	"github.com/dat-lt-amira/github-mirror/internal/store"
	"github.com/dat-lt-amira/github-mirror/internal/ui"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates a new chi router with routes and middleware.
func NewRouter(handler *Handler, userStore auth.UserStore, mirrorStore store.MirrorConfigStore, jobStore store.SyncJobStore) *chi.Mux {
	handler.UserStore = userStore
	handler.MirrorStore = mirrorStore
	handler.JobStore = jobStore

	uiHandler := ui.NewHandler(mirrorStore, jobStore)
	handler.UIRenderer = uiHandler

	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60))

	// Public routes (no auth required)
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", handler.RegisterHandler)
		r.Post("/login", handler.LoginHandler)
	})

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(auth.BasicAuth(userStore))

		// UI pages (GET)
		r.Get("/", uiHandler.Dashboard)
		r.Get("/guide", uiHandler.SetupGuide)
		r.Get("/mirrors/new", uiHandler.NewMirrorForm)
		r.Get("/mirrors/{id}/edit", uiHandler.EditMirrorForm)
		r.Get("/mirrors/{id}/schedule", uiHandler.EditMirrorScheduleForm)
		r.Get("/mirrors/{id}", uiHandler.MirrorDetail)
		r.Get("/partials/dashboard/mirrors", uiHandler.DashboardMirrorsPartial)
		r.Get("/partials/mirrors/{id}/configuration", uiHandler.MirrorConfigurationPartial)
		r.Get("/partials/mirrors/{id}/history", uiHandler.MirrorHistoryPartial)

		// API/HTMX endpoints (POST)
		r.Post("/mirrors", handler.CreateMirrorHandler)
		r.Post("/mirrors/{id}", handler.UpdateMirrorHandler)
		r.Post("/mirrors/{id}/schedule", handler.UpdateMirrorScheduleHandler)
		r.Post("/mirrors/{id}/test", handler.TestMirrorHandler)
		r.Post("/mirrors/{id}/retry", handler.RetryMirrorHandler)
		r.Post("/mirrors/{id}/sync", handler.SyncMirrorHandler)
		r.Post("/mirrors/{id}/delete", handler.DeleteMirrorHandler)
		r.Delete("/mirrors/{id}", handler.DeleteMirrorHandler)

		// JSON API
		r.Get("/api/mirrors", handler.ListMirrorsHandler)
		r.Get("/api/mirrors/{id}", handler.GetMirrorHandler)
	})

	// Webhook routes (authenticated via signature, not basic auth)
	r.Post("/webhooks/github/{id}", handler.WebhookHandler)

	// Health check
	r.Get("/health", handler.HealthHandler)
	r.Get("/favicon.ico", handler.FaviconHandler)

	return r
}
