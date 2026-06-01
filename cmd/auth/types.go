package main

import "time"

type User struct {
	ID                   int
	Email                string
	PasswordHash         string
	CreatedAt            time.Time
	UpdatedAt            time.Time
	VacationModeEnabled  bool
	VacationModeStartedAt *time.Time
}

type UserResponse struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type VacationStatusResponse struct {
	Enabled        bool    `json:"enabled"`
	StartedAt      *string `json:"started_at"`
	CanConfirm     bool    `json:"can_confirm"`
	RemainingHours float64 `json:"remaining_hours"`
}

type UserVacationStatusResponse struct {
	VacationModeEnabled bool `json:"vacation_mode_enabled"`
}
