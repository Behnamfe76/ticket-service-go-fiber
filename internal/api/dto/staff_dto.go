package dto

// StaffLoginRequest payload.
type StaffLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// PasswordResetRequest payload for initiating reset.
type PasswordResetRequest struct {
	Email string `json:"email"`
}

// PasswordResetConfirmRequest payload for confirming reset.
type PasswordResetConfirmRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// PasswordChangeRequest payload for authenticated password changes.
type PasswordChangeRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}
