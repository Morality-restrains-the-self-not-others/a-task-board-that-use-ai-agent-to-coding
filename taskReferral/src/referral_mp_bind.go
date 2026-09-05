package main

import "strings"

// userHasServiceAccountBound 默认 true，避免单测无 wechat_identity 表时申请全失败。
// main() 在 openAuthDB 之后接到 live 实现。
var userHasServiceAccountBound = func(string) bool { return true }

func userHasServiceAccountBoundLive(userID string) bool {
	if authDB == nil || strings.TrimSpace(userID) == "" {
		return false
	}
	var n int
	err := authDB.QueryRow(`
		SELECT COUNT(*) FROM wechat_identity
		WHERE user_id = ? AND app_key = 'mp' AND openid != ''`,
		userID).Scan(&n)
	return err == nil && n > 0
}
