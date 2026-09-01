package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"notes-app/internal/middleware"
	"notes-app/internal/models"
)

type NoteHandler struct {
	DB *sql.DB
}

type noteRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// List returns every note belonging to the authenticated user, newest first.
func (h *NoteHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	rows, err := h.DB.Query(
		`SELECT id, user_id, title, content, created_at, updated_at
		 FROM notes WHERE user_id = $1 ORDER BY updated_at DESC`,
		userID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "we couldn't load your notes - please try again")
		return
	}
	defer rows.Close()

	notes := []models.Note{}
	for rows.Next() {
		var n models.Note
		if err := rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "we couldn't read your notes - please try again")
			return
		}
		notes = append(notes, n)
	}

	writeJSON(w, http.StatusOK, notes)
}

// Create adds a new note for the authenticated user.
func (h *NoteHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	var req noteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "we couldn't read that request")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "give your note a title first")
		return
	}

	var n models.Note
	err := h.DB.QueryRow(
		`INSERT INTO notes (user_id, title, content) VALUES ($1, $2, $3)
		 RETURNING id, user_id, title, content, created_at, updated_at`,
		userID, req.Title, req.Content,
	).Scan(&n.ID, &n.UserID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "we couldn't save that note - please try again")
		return
	}

	writeJSON(w, http.StatusCreated, n)
}

// Update edits a note, but only if it belongs to the authenticated user.
func (h *NoteHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	noteID, err := noteIDFromPath(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "that note id doesn't look right")
		return
	}

	var req noteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "we couldn't read that request")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "give your note a title first")
		return
	}

	var n models.Note
	err = h.DB.QueryRow(
		`UPDATE notes SET title = $1, content = $2, updated_at = now()
		 WHERE id = $3 AND user_id = $4
		 RETURNING id, user_id, title, content, created_at, updated_at`,
		req.Title, req.Content, noteID, userID,
	).Scan(&n.ID, &n.UserID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt)

	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "we couldn't find that note")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "we couldn't save your changes - please try again")
		return
	}

	writeJSON(w, http.StatusOK, n)
}

// Delete removes a note, but only if it belongs to the authenticated user.
func (h *NoteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	noteID, err := noteIDFromPath(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "that note id doesn't look right")
		return
	}

	result, err := h.DB.Exec(`DELETE FROM notes WHERE id = $1 AND user_id = $2`, noteID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "we couldn't delete that note - please try again")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		writeError(w, http.StatusNotFound, "we couldn't find that note")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// noteIDFromPath reads the {id} path value set by Go's ServeMux
// (e.g. "/api/notes/{id}") and parses it as an integer.
func noteIDFromPath(r *http.Request) (int, error) {
	return strconv.Atoi(r.PathValue("id"))
}
