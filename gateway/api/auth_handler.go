package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/glrs/observer/gateway/auth"
	"github.com/glrs/observer/gateway/config"
)

// AuthHandler handles login and authentication.
type AuthHandler struct {
	config *config.AuthSection
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(cfg *config.AuthSection) *AuthHandler {
	return &AuthHandler{config: cfg}
}

// LoginRequest is the expected login body.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse is returned on successful login.
type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
	Username  string `json:"username"`
}

// Login handles POST /api/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	// Find user and validate password
	var matchedUser *config.UserDef
	for _, u := range h.config.Users {
		if u.Username == req.Username {
			matchedUser = &u
			break
		}
	}

	secret := h.config.JWTSecret

	// In Phase 1, compare plaintext (bcrypt will be used when we pre-hash)
	if matchedUser == nil || matchedUser.Password != req.Password {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid username or password"})
		return
	}

	token, expiresAt, err := auth.GenerateToken(req.Username, secret)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
		return
	}

	// Set HttpOnly cookie for iframe/auth compatibility
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Expires:  time.Unix(expiresAt, 0),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	writeJSON(w, http.StatusOK, LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		Username:  req.Username,
	})
}

// Helper: write JSON response.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
