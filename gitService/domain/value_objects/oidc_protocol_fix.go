// Package value_objects defines immutable domain values.
package value_objects

import "errors"

// OIDCProtocolFix represents the SWD.url_builder fix configuration.
// It is a value object — two fixes are equal if they target the same scheme.
type OIDCProtocolFix struct {
	// TargetScheme is the URI scheme to use for OIDC discovery (must be "http" or "https").
	targetScheme string
	// InitializerPath is the absolute path to the Rails initializer inside the container.
	initializerPath string
}

// NewOIDCProtocolFix creates a validated fix configuration.
func NewOIDCProtocolFix(scheme, path string) (OIDCProtocolFix, error) {
	if scheme != "http" && scheme != "https" {
		return OIDCProtocolFix{}, errors.New("target scheme must be http or https")
	}
	if path == "" {
		return OIDCProtocolFix{}, errors.New("initializer path must not be empty")
	}
	return OIDCProtocolFix{targetScheme: scheme, initializerPath: path}, nil
}

// Scheme returns the target URI scheme.
func (f OIDCProtocolFix) Scheme() string { return f.targetScheme }

// InitializerPath returns the Rails initializer path.
func (f OIDCProtocolFix) InitializerPath() string { return f.initializerPath }

// InitializerContent returns the Ruby code to inject.
func (f OIDCProtocolFix) InitializerContent() string {
	return `require "swd"
SWD.url_builder = URI::` + f.targetScheme + `
`
}

// IsHTTP returns true if the fix targets HTTP (the default for local dev).
func (f OIDCProtocolFix) IsHTTP() bool { return f.targetScheme == "http" }

// Equals compares two OIDCProtocolFix instances by value.
func (f OIDCProtocolFix) Equals(other OIDCProtocolFix) bool {
	return f.targetScheme == other.targetScheme && f.initializerPath == other.initializerPath
}
