//go:build !redis

package main

// incrMembershipRev bumps the membership revision counter for userID so that
// any outstanding membership JWT carrying a lower rev is invalidated.
//
// This is the default no-op implementation.  To enable Redis-backed revision
// tracking, build with `-tags redis` (see membership_rev_redis.go).
func incrMembershipRev(userID string) {
	// no-op: Redis not configured.
	// Set TENANT_REDIS_HOST env var and build with -tags redis to enable.
}

// incrGroupMembersRev — no-op 变体（与 redis 版签名一致，编译期占位）。
func incrGroupMembersRev(groupID string) {
	// no-op
}
