package domain

import "time"

// ─── Domain Events ───────────────────────────────────────────────────────────

// TokenIssued is emitted when a new container access token is created.
type TokenIssued struct {
	TokenID     string
	TaskID      string
	Scope       TaskScope
	OccurredAt  time.Time
}

// TokenExchanged is emitted when an access token is exchanged for a refresh token.
type TokenExchanged struct {
	TokenID        string
	TaskID         string
	Scope          TaskScope
	OccurredAt     time.Time
}

// TokenRefreshed is emitted when a refresh token is used to get a new access token.
type TokenRefreshed struct {
	TokenID    string
	TaskID     string
	Scope      TaskScope
	OccurredAt time.Time
}

// CredentialsFetchAttempted is emitted when repo clone credentials are requested.
type CredentialsFetchAttempted struct {
	TaskID       string
	Scope        TaskScope
	RepoCount    int
	OccurredAt   time.Time
}

// CredentialsFetchFailed is emitted when repo clone credentials are incomplete.
type CredentialsFetchFailed struct {
	TaskID           string
	Scope            TaskScope
	MissingRepoURLs  []string
	ErrorCode        string
	TraceID          string
	OccurredAt       time.Time
}

// TokenAuditRecorded is emitted for every token lifecycle event.
type TokenAuditRecorded struct {
	Event      TokenAuditEvent
	OccurredAt time.Time
}
