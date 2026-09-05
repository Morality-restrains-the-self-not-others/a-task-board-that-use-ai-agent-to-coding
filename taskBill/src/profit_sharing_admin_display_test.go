package main

import (
	"context"
	"testing"
)

// OPT-20260822-024: 管理端分账接收方展示用户显示名（username → email → user_id 兜底），
// 不因上游 taskAuth 抖动阻塞账单数据，禁止下发 openid。

func TestEnrichProfitSharingReceiverDisplaysUsernamePriority(t *testing.T) {
	ctx := context.Background()
	orig := fetchUserBatchDetailsFn
	t.Cleanup(func() { fetchUserBatchDetailsFn = orig })
	fetchUserBatchDetailsFn = func(_ context.Context, ids []interface{}) (map[string]userDisplayInfo, error) {
		return map[string]userDisplayInfo{
			"u1": {Username: "软刀", Email: "ljy@example.com"},
			"u2": {Email: "bob@example.com"},
			"u3": {},
		}, nil
	}
	rows := []map[string]interface{}{
		{"receiver_user_id": "u1"},
		{"receiver_user_id": "u2"},
		{"receiver_user_id": "u3"},
	}
	enrichProfitSharingReceiverDisplays(ctx, rows)
	if got := rows[0]["receiver_display"]; got != "软刀" {
		t.Fatalf("u1 receiver_display=%v want 软刀", got)
	}
	if got := rows[1]["receiver_display"]; got != "bob@example.com" {
		t.Fatalf("u2 receiver_display=%v want bob@example.com", got)
	}
	if got := rows[2]["receiver_display"]; got != "u3" {
		t.Fatalf("u3 receiver_display=%v want u3 (fallback to id)", got)
	}
}

func TestEnrichProfitSharingReceiverDisplaysUpstreamFailureDegrades(t *testing.T) {
	ctx := context.Background()
	orig := fetchUserBatchDetailsFn
	t.Cleanup(func() { fetchUserBatchDetailsFn = orig })
	fetchUserBatchDetailsFn = func(_ context.Context, ids []interface{}) (map[string]userDisplayInfo, error) {
		return nil, errForTest("taskAuth down")
	}
	rows := []map[string]interface{}{{"receiver_user_id": "u1"}}
	enrichProfitSharingReceiverDisplays(ctx, rows)
	if _, ok := rows[0]["receiver_display"]; ok {
		t.Fatalf("receiver_display should be absent on upstream failure: %v", rows[0])
	}
}

func TestEnrichProfitSharingReceiverDisplaysEmptyRowsNoop(t *testing.T) {
	ctx := context.Background()
	orig := fetchUserBatchDetailsFn
	t.Cleanup(func() { fetchUserBatchDetailsFn = orig })
	fetchUserBatchDetailsFn = func(_ context.Context, ids []interface{}) (map[string]userDisplayInfo, error) {
		t.Fatal("must not call lookup for empty rows")
		return nil, nil
	}
	enrichProfitSharingReceiverDisplays(ctx, nil)
	enrichProfitSharingReceiverDisplays(ctx, []map[string]interface{}{{}})
}

type testSentinelError struct{ msg string }

func (e testSentinelError) Error() string { return e.msg }

func errForTest(msg string) error { return testSentinelError{msg: msg} }
