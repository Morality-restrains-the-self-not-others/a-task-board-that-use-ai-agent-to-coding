package main

import "testing"

func TestFormatResourceChangeLineTaskPost(t *testing.T) {
	got := formatResourceChangeLine(ResourceTypeTaskPost, 10, "")
	want := "任务帖 +10 帖"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestFormatResourceChangeLineGitlabDiskIncludesRegion(t *testing.T) {
	got := formatResourceChangeLine(ResourceTypeGitlabDisk, 5, "tencent-sh-1")
	want := "GitLab 磁盘 +5 GB（tencent-sh-1）"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestFormatResourceChangeLineGitlabTraffic(t *testing.T) {
	got := formatResourceChangeLine(ResourceTypeGitlabTraffic, 3, "huawei-bj-1")
	want := "GitLab 流量 +3 GB（huawei-bj-1）"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestFormatAdminGrantDescriptionSingleWithoutReason(t *testing.T) {
	got := formatAdminGrantDescription([]ResourceGrantInput{
		{ResourceType: ResourceTypeTaskPost, Quantity: 10},
	})
	want := "管理员后台赠送：任务帖 +10 帖"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestFormatAdminGrantDescriptionMultiple(t *testing.T) {
	got := formatAdminGrantDescription([]ResourceGrantInput{
		{ResourceType: ResourceTypeTaskPost, Quantity: 10},
		{ResourceType: ResourceTypeGitlabDisk, Quantity: 5, Region: "tencent-sh-1"},
	})
	want := "管理员后台赠送：任务帖 +10 帖；GitLab 磁盘 +5 GB（tencent-sh-1）"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestFormatAdminGrantDescriptionAppendsDistinctReason(t *testing.T) {
	got := formatAdminGrantDescription([]ResourceGrantInput{
		{ResourceType: ResourceTypeTaskPost, Quantity: 3, Reason: "活动补偿"},
	})
	want := "管理员后台赠送：任务帖 +3 帖（活动补偿）"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAdminGrantTransactionDescriptionMembershipOnly(t *testing.T) {
	got := adminGrantTransactionDescription(nil, MembershipTierVIP1, "")
	want := "管理员设置会员等级为 " + MembershipTierVIP1
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAdminGrantTransactionDescriptionMembershipReasonWinsWhenNoGrants(t *testing.T) {
	got := adminGrantTransactionDescription(nil, MembershipTierVIP1, "开通购买")
	if got != "开通购买" {
		t.Fatalf("got %q", got)
	}
}

func TestPointsSourceTypeDisplay(t *testing.T) {
	cases := map[string]string{
		"admin_grant":          "后台赠送",
		"user_recharge_paypal": "PayPal 支付",
		"user_recharge_wechat": "微信支付",
		"user_recharge_admin":  "管理员直充",
		"user_recharge":        "用户支付（历史）",
		"promotion":            "活动赠送",
		"adjustment":           "人工调账",
		"quota_consumption":    "配额消耗",
		"consumption":          "消耗",
		"custom_source":        "custom_source",
		"":                     "",
	}
	for src, want := range cases {
		if got := pointsSourceTypeDisplay(src); got != want {
			t.Fatalf("src=%q got %q want %q", src, got, want)
		}
	}
}

func TestCreatedAtSecondTruncatesMicros(t *testing.T) {
	got := createdAtSecond("2026-08-18 11:19:04.123456")
	if got != "2026-08-18 11:19:04" {
		t.Fatalf("got %q", got)
	}
	if createdAtSecond("2026-08-18 11:19:04") != "2026-08-18 11:19:04" {
		t.Fatalf("already-second form changed")
	}
}

func TestJoinResourceChangeDisplays(t *testing.T) {
	got := joinResourceChangeDisplays([]resourceChange{
		{Display: "任务帖 +10 帖"},
		{Display: "GitLab 磁盘 +5 GB（tencent-sh-1）"},
	})
	want := "任务帖 +10 帖；GitLab 磁盘 +5 GB（tencent-sh-1）"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestIsGenericAdminGrantDescription(t *testing.T) {
	if !isGenericAdminGrantDescription("管理员后台赠送资源") {
		t.Fatal("plain generic")
	}
	if !isGenericAdminGrantDescription("管理员后台赠送资源: [补偿]") {
		t.Fatal("legacy reasons dump")
	}
	if isGenericAdminGrantDescription("管理员后台赠送：任务帖 +10 帖") {
		t.Fatal("concrete description must not be treated as generic")
	}
}

func TestFormatCashSnapshotLine(t *testing.T) {
	got := formatCashSnapshotLine(100, 70)
	if got != "余额 1.00 → 0.70 元" {
		t.Fatalf("got %q", got)
	}
	unchanged := formatCashSnapshotLine(1500, 1500)
	if unchanged != "余额 15.00 元" {
		t.Fatalf("unchanged %q", unchanged)
	}
}
