// Package domain holds taskAuth core types (no infrastructure imports).
package domain

// LoginMethodRef identifies a row in auth_login_method.
type LoginMethodRef struct {
	ID         int64
	ObjectID   string
	MethodType string
	Identifier string
}

// PasswordCredential is the frontend-hashed password stored in password_hash.
type PasswordCredential string

// ResetToken is a time-limited password reset token on login_method.
type ResetToken struct {
	Token     string
	ExpiresAt string
}
