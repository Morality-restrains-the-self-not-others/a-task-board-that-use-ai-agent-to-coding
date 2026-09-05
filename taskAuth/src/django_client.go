package main

import (
	"encoding/json"
	"sync"
	"time"
)

// ---- cached system feature policy for region whitelist ---- //
// OPT-052: Formerly fetched from Django /api/public/system-feature-policy/ (retired 2026-07-30).
// All Django callback functions (enrichLogin, postRegister, postActivate, etc.) removed.
// System now uses permissive defaults — isPhoneCountryCodeAllowed returns true for all codes.

var (
	cachedPolicy    *cachedSystemFeaturePolicy
	cachedPolicyMu  sync.Mutex
	cachedPolicyTTL = 60 * time.Second
)

type cachedSystemFeaturePolicy struct {
	AllowedPhoneCountryCodes []string
	FetchedAt                time.Time
}

func isPhoneCountryCodeAllowed(cc string) bool {
	policy := fetchSystemFeaturePolicyCached()
	if policy == nil || len(policy.AllowedPhoneCountryCodes) == 0 {
		return true // empty list = allow all
	}
	for _, c := range policy.AllowedPhoneCountryCodes {
		if c == cc {
			return true
		}
	}
	return false
}

func fetchSystemFeaturePolicyCached() *cachedSystemFeaturePolicy {
	cachedPolicyMu.Lock()
	defer cachedPolicyMu.Unlock()
	if cachedPolicy != nil && time.Since(cachedPolicy.FetchedAt) < cachedPolicyTTL {
		return cachedPolicy
	}
	p, err := fetchPublicSystemFeaturePolicy()
	if err != nil {
		// On error, reuse stale cache if available
		if cachedPolicy != nil {
			return cachedPolicy
		}
		return nil
	}
	cachedPolicy = p
	return p
}

// fetchPublicSystemFeaturePolicy reads the system feature policy from the DB.
// OPT-052: Formerly fetched /api/public/system-feature-policy/ from Django (retired 2026-07-30).
// Now reads from the taskAuth-owned auth_system_feature_policy table (OPT-049).
func fetchPublicSystemFeaturePolicy() (*cachedSystemFeaturePolicy, error) {
	row, err := loadFeaturePolicy()
	if err != nil {
		return nil, err
	}
	var codes []string
	if row.AllowedPhoneCountryCodesJSON != "" && row.AllowedPhoneCountryCodesJSON != "[]" {
		if err := json.Unmarshal([]byte(row.AllowedPhoneCountryCodesJSON), &codes); err != nil {
			codes = []string{}
		}
	}
	return &cachedSystemFeaturePolicy{
		AllowedPhoneCountryCodes: codes,
		FetchedAt:                time.Now(),
	}, nil
}
