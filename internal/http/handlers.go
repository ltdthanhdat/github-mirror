package http

import (
	"encoding/json"
	"net/http"

	"github.com/dat-lt-amira/github-mirror/internal/auth"
	"github.com/dat-lt-amira/github-mirror/internal/models"
	"github.com/dat-lt-amira/github-mirror/internal/store"
)

type MirrorFormRenderer interface {
	RenderMirrorFormPage(w http.ResponseWriter, data map[string]interface{})
}

// Handler holds dependencies for the HTTP handlers.
type Handler struct {
	UserStore   auth.UserStore
	MirrorStore store.MirrorConfigStore
	JobStore    store.SyncJobStore
	UIRenderer  MirrorFormRenderer
}

// RegisterHandler handles user registration.
func (h *Handler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		FullName string `json:"full_name,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	if _, err := h.UserStore.GetUserByEmail(req.Email); err == nil {
		http.Error(w, "User already exists", http.StatusConflict)
		return
	}

	user := &models.User{
		Email:    req.Email,
		FullName: req.FullName,
	}
	if err := user.SetPassword(req.Password); err != nil {
		http.Error(w, "Failed to process password", http.StatusInternalServerError)
		return
	}

	if err := h.UserStore.CreateUser(user); err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "User registered successfully",
	})
}

// LoginHandler handles user login and validates credentials.
func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	user, err := h.UserStore.GetUserByEmail(req.Email)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := user.CheckPassword(req.Password); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Login successful",
	})
}
