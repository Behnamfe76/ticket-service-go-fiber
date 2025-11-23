package dto

import "time"

// UserRegisterRequest payload for new users.
type UserRegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UserLoginRequest payload for login.
type UserLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse standard response for auth endpoints.
type AuthResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}
