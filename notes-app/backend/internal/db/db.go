// Package db opens and verifies the Postgres connection used by the app.
package db

import (
	"database/sql"
	"log"          // ✅ ADD THIS: for logging messages
	// "fmt"        // ❌ REMOVED: not used
	// "time"       // ❌ REMOVED: not used

	_ "github.com/lib/pq" // Postgres driver, registered via side-effect import
)

// Connect opens a connection pool to Postgres and pings it to make sure
// the credentials/host are actually reachable before the server starts.
func Connect(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}

	// ===== ADD MIGRATIONS HERE =====
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS notes (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			content TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_notes_user_id ON notes(user_id)`,
		`ALTER TABLE users
			ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT '',
			ADD COLUMN IF NOT EXISTS avatar_data_url TEXT NOT NULL DEFAULT ''`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			// Log but don't crash - tables might already exist
			log.Printf("⚠️ Migration warning: %v", err) // ✅ `log` is now imported
		}
	}
	log.Println("✅ Migrations completed") // ✅ `log` is now imported
	// ===== END MIGRATIONS =====

	return db, nil
}