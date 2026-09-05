// Package accounts holds domain types for account/org event handlers (G2+).
package accounts

// UserCreatedPayload is the USER_CREATED event data contract.
type UserCreatedPayload struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

// CompanyCreatedPayload is published after USER_CREATED succeeds.
type CompanyCreatedPayload struct {
	CompanyID int64  `json:"company_id"`
	UserID    string `json:"user_id"`
	Name      string `json:"name"`
}
