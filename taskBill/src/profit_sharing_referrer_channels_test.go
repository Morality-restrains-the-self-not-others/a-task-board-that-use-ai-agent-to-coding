package main

import (
	"testing"
	"time"
)

func TestAggregateReferrerChannelsBucketsAndHidesOrderIdentity(t *testing.T) {
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	frozenPaid := now.Add(-5 * 24 * time.Hour).Format(time.RFC3339)
	sharePaid := now.Add(-20 * 24 * time.Hour).Format(time.RFC3339)
	sharedPaid := now.Add(-18 * 24 * time.Hour).Format(time.RFC3339)

	got := aggregateReferrerChannels([]referrerPSListRow{
		{ChannelCode: "CH-B", Commission: 30, Total: 600, Status: psStatusFinished, PaidAt: sharedPaid},
		{ChannelCode: "CH-A", Commission: 50, Total: 1000, Status: psStatusPending, PaidAt: frozenPaid},
		{ChannelCode: "CH-A", Commission: 125, Total: 2500, Status: psStatusPending, PaidAt: sharePaid},
	}, now)
	if len(got) != 2 {
		t.Fatalf("channels=%d want 2: %+v", len(got), got)
	}
	if got[0].ChannelCode != "CH-A" || got[1].ChannelCode != "CH-B" {
		t.Fatalf("sort=%q,%q", got[0].ChannelCode, got[1].ChannelCode)
	}
	a := got[0]
	if a.OrderAmountYuanCents != 3500 || a.FrozenAmountYuanCents != 50 || a.ShareableAmountYuanCents != 125 || !a.Shareable {
		t.Fatalf("CH-A amounts=%+v", a)
	}
	if a.PeriodFrom != sharePaid || a.PeriodTo != frozenPaid {
		t.Fatalf("CH-A period from=%s to=%s", a.PeriodFrom, a.PeriodTo)
	}
	b := got[1]
	if b.OrderAmountYuanCents != 600 || b.FrozenAmountYuanCents != 0 || b.ShareableAmountYuanCents != 0 || b.Shareable {
		t.Fatalf("CH-B amounts=%+v", b)
	}
}

func TestAggregateReferrerChannelsFailedInFreezeNotFrozen(t *testing.T) {
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	frozenPaid := now.Add(-5 * 24 * time.Hour).Format(time.RFC3339)
	got := aggregateReferrerChannels([]referrerPSListRow{
		{ChannelCode: "CH-A", Commission: 50, Total: 1000, Status: psStatusPending, PaidAt: frozenPaid},
		{ChannelCode: "CH-A", Commission: 80, Total: 1600, Status: psStatusFailed, PaidAt: frozenPaid},
	}, now)
	if len(got) != 1 {
		t.Fatalf("channels=%d want 1: %+v", len(got), got)
	}
	a := got[0]
	if a.FrozenAmountYuanCents != 50 {
		t.Fatalf("frozen=%d want 50 (failed-in-freeze must not count as frozen)", a.FrozenAmountYuanCents)
	}
	// OPT-20260824-086: 失败金额单独成列（failed-in-freeze 计入失败而非冻结）。
	if a.FailedAmountYuanCents != 80 {
		t.Fatalf("failed=%d want 80 (failed-in-freeze should count as failed)", a.FailedAmountYuanCents)
	}
	if a.ShareableAmountYuanCents != 0 || a.Shareable {
		t.Fatalf("shareable=%d shareable=%v want 0/false", a.ShareableAmountYuanCents, a.Shareable)
	}
	if a.OrderAmountYuanCents != 2600 {
		t.Fatalf("order=%d want 2600", a.OrderAmountYuanCents)
	}
}

func TestAggregateReferrerChannelsEmptyChannelCode(t *testing.T) {
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	got := aggregateReferrerChannels([]referrerPSListRow{
		{ChannelCode: "", Commission: 10, Total: 200, Status: psStatusPending, PaidAt: now.Add(-20 * 24 * time.Hour).Format(time.RFC3339)},
	}, now)
	if len(got) != 1 || got[0].ChannelCode != "" || !got[0].Shareable || got[0].ShareableAmountYuanCents != 10 {
		t.Fatalf("empty channel=%+v", got)
	}
}
