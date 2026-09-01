package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"notes-app/internal/auth"
	"notes-app/internal/models"

	"github.com/lib/pq"
)

type AuthHandler struct {
	DB        *sql.DB
	JWTSecret string
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string      `json:"token"`
	User  models.User `json:"user"`
}

// Register creates a new user account with a bcrypt-hashed password.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "we couldn't read that request")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Name = strings.TrimSpace(req.Name)

	if req.Email == "" || req.Password == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "name, email, and password are required")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "your password needs to be at least 8 characters")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "something went wrong on our end - please try again")
		return
	}

	var user models.User
	err = h.DB.QueryRow(
		`INSERT INTO users (email, password_hash, name) VALUES ($1, $2, $3)
		 RETURNING id, email, name, avatar_data_url, created_at`,
		req.Email, hash, req.Name,
	).Scan(&user.ID, &user.Email, &user.Name, &user.AvatarDataURL, &user.CreatedAt)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" { // unique_violation
			writeError(w, http.StatusConflict, "looks like that email already has an account - try logging in instead")
			return
		}
		writeError(w, http.StatusInternalServerError, "we couldn't create your account - please try again")
		return
	}

	token, err := auth.GenerateToken(user.ID, h.JWTSecret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "something went wrong on our end - please try again")
		return
	}

	writeJSON(w, http.StatusCreated, authResponse{Token: token, User: user})
}

// Login verifies credentials and returns a fresh JWT on success.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "we couldn't read that request")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	var user models.User
	err := h.DB.QueryRow(
		`SELECT id, email, password_hash, name, avatar_data_url, created_at FROM users WHERE email = $1`,
		req.Email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.AvatarDataURL, &user.CreatedAt)

	if err == sql.ErrNoRows || !auth.CheckPassword(req.Password, user.PasswordHash) {
		// Deliberately vague: don't reveal whether the email exists.
		writeError(w, http.StatusUnauthorized, "that email and password don't match - want to try again?")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "we couldn't log you in right now - please try again")
		return
	}

	token, err := auth.GenerateToken(user.ID, h.JWTSecret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "something went wrong on our end - please try again")
		return
	}

	writeJSON(w, http.StatusOK, authResponse{Token: token, User: user})
}

type checkEmailRequest struct {
	Email string `json:"email"`
}

// CheckEmail reports whether an account exists for the given email.
//
// NOTE: this intentionally reveals whether an email is registered, which
// is a real privacy/security tradeoff (normal practice is to always say
// "if that email exists, we sent a reset link" regardless of the truth).
// It's implemented this way because there's no email-sending service wired
// up yet, so this is standing in for a proper "forgot password" flow.
// Before this app has real users, swap this for an emailed reset link.
func (h *AuthHandler) CheckEmail(w http.ResponseWriter, r *http.Request) {
	var req checkEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" {
		writeError(w, http.StatusBadRequest, "enter an email address")
		return
	}

	var exists bool
	err := h.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`, email).Scan(&exists)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "something went wrong - please try again")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"exists": exists})
}

type resetPasswordRequest struct {
	Email       string `json:"email"`
	NewPassword string `json:"new_password"`
}

// ResetPassword sets a new password for the given email, with no other
// verification. See the caveat on CheckEmail above - this is a
// simplified stand-in for a real "email me a reset link" flow.
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" {
		writeError(w, http.StatusBadRequest, "enter an email address")
		return
	}
	if len(req.NewPassword) < 8 {
		writeError(w, http.StatusBadRequest, "your new password needs to be at least 8 characters")
		return
	}

	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "something went wrong - please try again")
		return
	}

	result, err := h.DB.Exec(`UPDATE users SET password_hash = $1 WHERE email = $2`, hash, email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "we couldn't update your password - please try again")
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		writeError(w, http.StatusNotFound, "we couldn't find an account with that email")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Your password has been updated. You can log in now."})
}
