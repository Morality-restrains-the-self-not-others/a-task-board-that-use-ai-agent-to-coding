package main

import "testing"

func TestIsMisSeededMemberName(t *testing.T) {
	cases := []struct {
		memberName  string
		companyName string
		want        bool
	}{
		{"", "Acme", true},
		{"我的公司", "Acme", true},
		{"我的公司", "我的公司", true},
		{"Acme", "Acme", true},
		{"Alice", "Acme", false},
		{"Alice", "Alice Corp", false},
		{"  ", "Acme", true},
	}
	for _, tc := range cases {
		got := isMisSeededMemberName(tc.memberName, tc.companyName)
		if got != tc.want {
			t.Fatalf("isMisSeededMemberName(%q,%q)=%v want %v", tc.memberName, tc.companyName, got, tc.want)
		}
	}
}

func TestResolveMemberDisplayNamePrefersPersonalNickname(t *testing.T) {
	// 误种子「我的公司」→ 显示个人昵称
	got := resolveMemberDisplayName("我的公司", "我的公司", "软刀", "875304088135299072")
	if got != "软刀" {
		t.Fatalf("got %q, want 软刀", got)
	}
	// 误种子等于公司名 → 个人昵称
	got = resolveMemberDisplayName("Acme", "Acme", "Bob", "u1")
	if got != "Bob" {
		t.Fatalf("got %q, want Bob", got)
	}
	// 正确的公司成员昵称不被覆盖
	got = resolveMemberDisplayName("Carol", "Acme", "CarolPersonal", "u1")
	if got != "Carol" {
		t.Fatalf("got %q, want Carol", got)
	}
	// 误种子且无个人昵称 → 不回显公司名，回退 userID 前缀
	got = resolveMemberDisplayName("我的公司", "我的公司", "", "875304088135299072")
	if got == "我的公司" {
		t.Fatalf("must not display default company name as member nickname")
	}
	if got != "87530408" {
		t.Fatalf("got %q, want userID prefix 87530408", got)
	}
}
