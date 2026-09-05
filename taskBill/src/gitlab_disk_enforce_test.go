package main

import (
	"fmt"
	"testing"
)

func TestRepoOwnerTenant_PersonalNamespaceResolvesToTenant(t *testing.T) {
	// BDD：个人命名空间仓 example-user/somanyad 经 gitlab 用户名 → 凭据 → 成员 → 区域资源
	// 链归集到租户（与流量闸门同链，OPT-20260824-080「个人命名空间命中租户 disk 配额」）。
	const wantTID int64 = 877397588196749312
	var gotUsername string
	stubUsernameResolver(t, func(username, region string) (int64, error) {
		gotUsername = username
		if region != "tencent-sh-1" {
			return 0, fmt.Errorf("unexpected region %s", region)
		}
		return wantTID, nil
	})

	got := repoOwnerTenant("example-user/somanyad", "tencent-sh-1")
	if got != wantTID {
		t.Fatalf("repoOwnerTenant=%d want %d", got, wantTID)
	}
	if gotUsername != "example-user" {
		t.Fatalf("resolver username=%q want example-user", gotUsername)
	}
}

func TestRepoOwnerTenant_TenantPrefixedProjectReturnsZero(t *testing.T) {
	// tenant-{id} 前缀仓由 path 直接可解，不触发用户名解析（调用方按 item.TenantID 下发）。
	var called bool
	stubUsernameResolver(t, func(_, _ string) (int64, error) {
		called = true
		return 0, fmt.Errorf("should not be called")
	})
	if got := repoOwnerTenant("tenant-877397588196749312/foo", "tencent-sh-1"); got != 0 {
		t.Fatalf("tenant-prefixed repoOwnerTenant=%d want 0", got)
	}
	if called {
		t.Fatal("resolver must not be called for tenant-{id} prefix")
	}
}

func TestRepoOwnerTenant_UnresolvedReturnsZero(t *testing.T) {
	// 解析失败保持 fail-safe（返回 0，调用方沿用现状继续下发）。
	stubUsernameResolver(t, func(_, _ string) (int64, error) {
		return 0, fmt.Errorf("no credential binding")
	})
	if got := repoOwnerTenant("someone/repo", "tencent-sh-1"); got != 0 {
		t.Fatalf("unresolved repoOwnerTenant=%d want 0", got)
	}
}

func TestShouldEnforceRepoForTenant_Attribution(t *testing.T) {
	const itemTID int64 = 877397588196749312
	const otherTID int64 = 877397588196749399

	t.Run("tenant-prefixed always enforced", func(t *testing.T) {
		if !shouldEnforceRepoForTenant("tenant-877397588196749312/foo", "tencent-sh-1", itemTID) {
			t.Fatal("tenant-{id} prefix must be enforced")
		}
	})

	t.Run("personal namespace owner matches item tenant enforced", func(t *testing.T) {
		stubUsernameResolver(t, func(_, _ string) (int64, error) { return itemTID, nil })
		if !shouldEnforceRepoForTenant("example-user/somanyad", "tencent-sh-1", itemTID) {
			t.Fatal("personal namespace resolving to same tenant must be enforced")
		}
	})

	t.Run("personal namespace owner other tenant skipped", func(t *testing.T) {
		stubUsernameResolver(t, func(_, _ string) (int64, error) { return otherTID, nil })
		if shouldEnforceRepoForTenant("otheruser/personal-repo", "tencent-sh-1", itemTID) {
			t.Fatal("personal namespace resolving to other tenant must be skipped")
		}
	})

	t.Run("unresolved kept fail-safe", func(t *testing.T) {
		stubUsernameResolver(t, func(_, _ string) (int64, error) { return 0, fmt.Errorf("unresolved") })
		if !shouldEnforceRepoForTenant("someone/repo", "tencent-sh-1", itemTID) {
			t.Fatal("unresolved personal namespace must stay enforced (fail-safe)")
		}
	})
}
