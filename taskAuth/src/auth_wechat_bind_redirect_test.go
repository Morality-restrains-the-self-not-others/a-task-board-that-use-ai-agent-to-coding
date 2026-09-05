package main

import (
	"net/http"
	"testing"
)

func TestWechatBindSuccessPathEmptyFallsBackToLogin(t *testing.T) {
	got := wechatBindSuccessPath("")
	want := "/auth/login/?wechat_bound=1"
	if got != want {
		t.Fatalf("empty next: got %q want %q", got, want)
	}
}

func TestWechatBindSuccessPathReferral(t *testing.T) {
	got := wechatBindSuccessPath("/profile/referral/")
	want := "/profile/referral/?wechat_bound=1"
	if got != want {
		t.Fatalf("referral next: got %q want %q", got, want)
	}
}

func TestWechatBindSuccessPathKeepsExistingQuery(t *testing.T) {
	got := wechatBindSuccessPath("/profile/referral/?from=gate")
	want := "/profile/referral/?from=gate&wechat_bound=1"
	if got != want {
		t.Fatalf("query next: got %q want %q", got, want)
	}
}

func TestWechatBindSuccessPathRejectsOpenRedirect(t *testing.T) {
	cases := []string{
		"https://evil.example/phish",
		"//evil.example/phish",
		"javascript:alert(1)",
		"profile/referral/",
	}
	for _, raw := range cases {
		got := wechatBindSuccessPath(raw)
		want := "/auth/login/?wechat_bound=1"
		if got != want {
			t.Fatalf("unsafe %q: got %q want %q", raw, got, want)
		}
	}
}

func TestWechatBindNextFromRequest(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/api/auth/wechat/bind/?app=web&next=/profile/referral/", nil)
	if err != nil {
		t.Fatal(err)
	}
	got := wechatBindNextFromRequest(r)
	if got != "/profile/referral/" {
		t.Fatalf("got %q", got)
	}
}
