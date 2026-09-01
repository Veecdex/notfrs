package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"notes-app/internal/auth"
	"notes-app/internal/middleware"
	"notes-app/internal/models"

	"github.com/lib/pq"
)

type ProfileHandler struct {
	DB *sql.DB
}

// Me returns the signed-in user's profile.
func (h *ProfileHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	var user models.User
	err := h.DB.QueryRow(
		`SELECT id, email, name, avatar_data_url, created_at FROM users WHERE id = $1`,
		userID,
	).Scan(&user.ID, &user.Email, &user.Name, &user.AvatarDataURL, &user.CreatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "we couldn't load your profile - please try again")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

type updateProfileRequest struct {
	Name          string `json:"name"`
	Email         string `json:"email"`
	AvatarDataURL string `json:"avatar_data_url"`
}

// UpdateProfile edits name, email, and/or avatar for the signed-in user.
func (h *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	var req updateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "we couldn't read that request")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if req.Name == "" || req.Email == "" {
		writeError(w, http.StatusBadRequest, "name and email can't be empty")
		return
	}

	// A rough size guard so nobody accidentally stores a multi-megabyte
	// image as text in Postgres. ~2MB of base64 is plenty for a small,
	// resized profile photo.
	if len(req.AvatarDataURL) > 2_000_000 {
		writeError(w, http.StatusBadRequest, "that photo is too large - try a smaller one")
		return
	}

	var user models.User
	err := h.DB.QueryRow(
		`UPDATE users SET name = $1, email = $2, avatar_data_url = $3
		 WHERE id = $4
		 RETURNING id, email, name, avatar_data_url, created_at`,
		req.Name, req.Email, req.AvatarDataURL, userID,
	).Scan(&user.ID, &user.Email, &user.Name, &user.AvatarDataURL, &user.CreatedAt)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			writeError(w, http.StatusConflict, "that email is already being used by another account")
			return
		}
		writeError(w, http.StatusInternalServerError, "we couldn't save your changes - please try again")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

type updatePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// UpdatePassword changes the signed-in user's password after verifying
// their current one.
func (h *ProfileHandler) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	var req updatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "we couldn't read that request")
		return
	}
	if len(req.NewPassword) < 8 {
		writeError(w, http.StatusBadRequest, "your new password needs to be at least 8 characters")
		return
	}

	var currentHash string
	if err := h.DB.QueryRow(`SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&currentHash); err != nil {
		writeError(w, http.StatusInternalServerError, "something went wrong - please try again")
		return
	}

	if !auth.CheckPassword(req.CurrentPassword, currentHash) {
		writeError(w, http.StatusUnauthorized, "your current password doesn't look right")
		return
	}

	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "something went wrong - please try again")
		return
	}

	if _, err := h.DB.Exec(`UPDATE users SET password_hash = $1 WHERE id = $2`, newHash, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "we couldn't update your password - please try again")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Your password has been updated."})
}

type deleteAccountRequest struct {
	Password string `json:"password"`
}

// DeleteAccount permanently removes the signed-in user (and, via the
// foreign key's ON DELETE CASCADE, all of their notes) after confirming
// their password.
func (h *ProfileHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	var req deleteAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "we couldn't read that request")
		return
	}

	var currentHash string
	if err := h.DB.QueryRow(`SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&currentHash); err != nil {
		writeError(w, http.StatusInternalServerError, "something went wrong - please try again")
		return
	}

	if !auth.CheckPassword(req.Password, currentHash) {
		writeError(w, http.StatusUnauthorized, "that password doesn't look right")
		return
	}

	if _, err := h.DB.Exec(`DELETE FROM users WHERE id = $1`, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "we couldn't delete your account - please try again")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Your account has been deleted."})
}
