package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const systemAutoGitIdentityLabel = "system-auto"
const systemGitEmailDomain = "daydaymoney.com"

// BuildSystemGitEmail returns deterministic Git email:
// {sha256(memberID)[:16]}.{sha256(tenantID)[:16]}@daydaymoney.com
func BuildSystemGitEmail(memberID, tenantID string) string {
	return fmt.Sprintf("%s.%s@%s",
		shortHash16(memberID),
		shortHash16(tenantID),
		systemGitEmailDomain,
	)
}

func shortHash16(s string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(s)))
	return hex.EncodeToString(sum[:])[:16]
}

// ResolveSystemGitUserName prefers company member display name.
func ResolveSystemGitUserName(memberName, userID string) string {
	name := strings.TrimSpace(memberName)
	if name != "" {
		return name
	}
	uid := strings.TrimSpace(userID)
	if uid != "" {
		return uid
	}
	return "user"
}
