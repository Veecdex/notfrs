// Package config loads application settings from environment variables
// (and, if present, a local .env file) into a single Config struct.
package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Port           string
	DatabaseURL    string // set by most cloud platforms (e.g. Railway) as one connection string
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	DBSSLMode      string
	JWTSecret      string
	FrontendOrigin string
}

// LoadDotEnv reads a ".env" file (if it exists) and injects any values
// into the process environment, without overriding variables that are
// already set (e.g. exported in your shell). This keeps things simple
// for beginners without needing an extra dependency.
func LoadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		// No .env file is fine - the app can still run on real env vars.
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, value)
		}
	}
}

// Load builds a Config from environment variables, applying sane
// defaults where possible.
func Load() Config {
	return Config{
		Port:           getEnv("PORT", "8080"),
		DatabaseURL:    getEnv("DATABASE_URL", ""),
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnv("DB_PORT", "5432"),
		DBUser:         getEnv("DB_USER", "postgres"),
		DBPassword:     getEnv("DB_PASSWORD", ""),
		DBName:         getEnv("DB_NAME", "notes_app"),
		DBSSLMode:      getEnv("DB_SSLMODE", "disable"),
		JWTSecret:      getEnv("JWT_SECRET", ""),
		FrontendOrigin: getEnv("FRONTEND_ORIGIN", "*"),
	}
}

// Validate does a quick sanity check so the app fails fast with a clear
// message instead of a confusing error later on.
func (c Config) Validate() error {
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is not set - add it to your .env file")
	}
	if len(c.JWTSecret) < 16 {
		return fmt.Errorf("JWT_SECRET is too short - use at least 16 random characters")
	}
	return nil
}

// DSN builds the Postgres connection string used by lib/pq. If
// DATABASE_URL is set (the norm on Railway, Render, Heroku, etc.) it is
// used as-is; otherwise the discrete DB_* variables are assembled, which
// is more convenient for local development against pgAdmin.
func (c Config) DSN() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
