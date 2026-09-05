package application

import "testing"

// TestGenerateSnowflake_UniqueUnderRapidCalls 回归（OPT-20260818 nightly）：
// generateSnowflake 此前仅返回 Unix 毫秒时间戳，同一毫秒内的连续调用会碰撞，
// 导致两个不同 comment 的 exchange-refresh 拿到同一个 refresh token
// （TestTokenService_TwoCommentsIndependentExchange 抽测失败）。
// 修复后同一进程内即使落在同一毫秒也必须唯一。
func TestGenerateSnowflake_UniqueUnderRapidCalls(t *testing.T) {
	seen := make(map[string]bool, 5000)
	for i := 0; i < 5000; i++ {
		id := generateSnowflake()
		if seen[id] {
			t.Fatalf("generateSnowflake collision: %q (i=%d)", id, i)
		}
		seen[id] = true
	}
}

// TestGenerateToken_UniqueUnderRapidCalls 覆盖 token 层契约：access/refresh token
// 必须两两不同，否则两个 comment 的 refresh 会在 FindByRefreshToken 上串行（SCOPE_MISMATCH）。
func TestGenerateToken_UniqueUnderRapidCalls(t *testing.T) {
	seen := make(map[string]bool, 5000)
	for i := 0; i < 5000; i++ {
		tok := generateToken()
		if seen[tok] {
			t.Fatalf("generateToken collision: %q (i=%d)", tok, i)
		}
		seen[tok] = true
	}
}
