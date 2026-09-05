package infrastructure

import "testing"

func TestExpandProviderKeys(t *testing.T) {
	got := ExpandProviderKeys("github:github-official")
	wantBare := false
	wantCompound := false
	wantDayday := false
	for _, k := range got {
		if k == "github" {
			wantBare = true
		}
		if k == "github:github-official" {
			wantCompound = true
		}
		if k == "github:github-official-daydaymoney" {
			wantDayday = true
		}
	}
	if !wantBare || !wantCompound || !wantDayday {
		t.Fatalf("got %#v", got)
	}
	got2 := ExpandProviderKeys("github")
	hasOfficial := false
	hasDayday := false
	for _, k := range got2 {
		if k == "github:github-official" {
			hasOfficial = true
		}
		if k == "github:github-official-daydaymoney" {
			hasDayday = true
		}
	}
	if !hasOfficial || !hasDayday {
		t.Fatalf("bare github aliases missing: %#v", got2)
	}
	got3 := ExpandProviderKeys("github:github-official-daydaymoney")
	hasLegacy := false
	for _, k := range got3 {
		if k == "github:github-official" {
			hasLegacy = true
		}
	}
	if !hasLegacy {
		t.Fatalf("daydaymoney should alias legacy official: %#v", got3)
	}
}

// TestExpandProviderKeysAll — 多键去重展开（user-app-connection 状态检查
// 用配置存储键 + 调用方键合并查询的基础）。注意：github:default 的展开
// 不含 legacy github:github-official*（这正是仅传 repo_url 时漏查配置键
// 凭据的根因），配置键须显式加入候选键。
func TestExpandProviderKeysAll(t *testing.T) {
	got := ExpandProviderKeysAll([]string{"github:default", "github:github-official-daydaymoney"})
	expected := map[string]bool{
		"github:default":                   false,
		"github":                           false,
		"github:github-official-daydaymoney": false,
	}
	for _, k := range got {
		if _, ok := expected[k]; ok {
			expected[k] = true
		}
	}
	for k, found := range expected {
		if !found {
			t.Fatalf("missing key %q in %#v", k, got)
		}
	}
	if len(got) != len(expected) {
		t.Fatalf("unexpected extra keys: %#v", got)
	}

	// 去重：相同键多次出现只保留一次
	dup := ExpandProviderKeysAll([]string{"github", "github:github-official", "github"})
	seen := map[string]int{}
	for _, k := range dup {
		seen[k]++
	}
	for k, n := range seen {
		if n != 1 {
			t.Fatalf("key %q duplicated %d times in %#v", k, n, dup)
		}
	}

	// 空输入 → 空结果
	if got := ExpandProviderKeysAll(nil); got != nil && len(got) != 0 {
		t.Fatalf("expected empty for nil input, got %#v", got)
	}
}
