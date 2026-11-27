package password_reset_handler

// resetRequest is the body of POST /auth/password-reset. Identifier is the email or the username.
type resetRequest struct {
	Identifier string `json:"identifier"`
}
