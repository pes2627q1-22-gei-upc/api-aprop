package auth_handler

// syncRequest is the optional body of POST /auth/sync. Username is only required the first time.
type syncRequest struct {
	Username string `json:"username"`
}
