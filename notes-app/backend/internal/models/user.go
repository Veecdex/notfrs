package models

import "time"

type User struct {
	ID            int       `json:"id"`
	Email         string    `json:"email"`
	Name          string    `json:"name"`
	AvatarDataURL string    `json:"avatar_data_url"`
	PasswordHash  string    `json:"-"` // never serialize this to JSON
	CreatedAt     time.Time `json:"created_at"`
}
