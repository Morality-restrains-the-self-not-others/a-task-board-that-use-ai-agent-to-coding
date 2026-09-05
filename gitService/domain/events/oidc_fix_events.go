// Package events defines domain events for cross-context communication.
package events

import "time"

// OIDCFixApplied is raised when the SWD.url_builder fix is successfully applied to a GitLab instance.
type OIDCFixApplied struct {
	// ContainerName identifies which GitLab instance was fixed.
	ContainerName string
	// Scheme is the URI scheme applied (e.g. "http").
	Scheme string
	// InitializerPath is the Rails initializer file path.
	InitializerPath string
	// OccurredAt is when the fix was applied.
	OccurredAt time.Time
}

// OIDCFixVerified is raised when an applied fix passes diagnostic verification.
type OIDCFixVerified struct {
	// ContainerName identifies the verified GitLab instance.
	ContainerName string
	// DiscoveryURL is the OIDC well-known URL that was tested.
	DiscoveryURL string
	// VerifiedAt is when verification completed.
	VerifiedAt time.Time
	// Passed indicates whether verification succeeded.
	Passed bool
}

// PlaywrightE2ECompleted is raised when a Playwright E2E test run finishes.
type PlaywrightE2ECompleted struct {
	// TestFile is the playwright test file that ran.
	TestFile string
	// SSOLoginPassed indicates the SSO login flow test result.
	SSOLoginPassed bool
	// DiagnosticPassed indicates the diagnostic test result.
	DiagnosticPassed bool
	// CompletedAt is when the test run finished.
	CompletedAt time.Time
}
