package main

import (
	"fmt"
	"strings"
)

const (
	inviteLinkKindSingle = "single"
	inviteLinkKindOpen   = "open"
	maxOpenInviteUsesCap = 10000
)

func parseInviteLinkKind(body map[string]interface{}, inviteMethod string) (linkKind string, maxUses int, err error) {
	linkKind = strings.ToLower(strField(body, "link_kind"))
	if linkKind == "" {
		linkKind = inviteLinkKindSingle
	}
	if linkKind != inviteLinkKindSingle && linkKind != inviteLinkKindOpen {
		return "", 0, fmt.Errorf("link_kind 仅支持 single、open")
	}
	if linkKind == inviteLinkKindOpen && inviteMethod != "link" {
		return "", 0, fmt.Errorf("open_invite_link_only")
	}
	if linkKind == inviteLinkKindSingle {
		return inviteLinkKindSingle, 1, nil
	}
	if _, ok := body["max_uses"]; !ok {
		return inviteLinkKindOpen, 0, nil
	}
	n := intField(body, "max_uses", 0)
	if n < 0 {
		return "", 0, fmt.Errorf("max_uses 不能为负数")
	}
	if n > maxOpenInviteUsesCap {
		return "", 0, fmt.Errorf("max_uses 超过上限")
	}
	return inviteLinkKindOpen, n, nil
}

func remainingInviteUses(maxUses, useCount int) *int {
	if maxUses <= 0 {
		return nil
	}
	left := maxUses - useCount
	if left < 0 {
		left = 0
	}
	return &left
}

func inviteHasRemainingUses(maxUses, useCount int) bool {
	if maxUses <= 0 {
		return true
	}
	return useCount < maxUses
}

func inviteTokenFingerprint(token string) string {
	if token == "" {
		return ""
	}
	if len(token) <= 8 {
		return "…"
	}
	return token[:4] + "…" + token[len(token)-4:]
}

func resolveJoinMemberName(body map[string]interface{}, inviteName string) string {
	if n := strField(body, "member_name"); n != "" {
		return n
	}
	if strings.TrimSpace(inviteName) != "" {
		return strings.TrimSpace(inviteName)
	}
	return "成员"
}
