package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestTenantGitLabOidcClientID(t *testing.T) {
	got, err := TenantGitLabOidcClientID(" 877397588196749312 ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "gitlab-tenant-877397588196749312" {
		t.Fatalf("got %q", got)
	}
	if _, err := TenantGitLabOidcClientID(""); !errors.Is(err, ErrEmptyCompanyID) {
		t.Fatalf("empty: %v", err)
	}
}

func TestOwnerCompanyIDFromClientID(t *testing.T) {
	cid, err := OwnerCompanyIDFromClientID("gitlab-tenant-abc")
	if err != nil || cid != "abc" {
		t.Fatalf("got %q %v", cid, err)
	}
	if _, err := OwnerCompanyIDFromClientID("gitlab-git-service"); !errors.Is(err, ErrNotTenantGitLabOidcClient) {
		t.Fatalf("platform prefix: %v", err)
	}
}

func TestIsTenantGitLabOidcClientAndRegionGate(t *testing.T) {
	if !IsTenantGitLabOidcClient("gitlab-tenant-1", "") {
		t.Fatal("tenant id")
	}
	if !IsTenantGitLabOidcClient("other", "tenant") {
		t.Fatal("managed_by")
	}
	if IsTenantGitLabOidcClient("gitlab-git-service", "bootstrap") {
		t.Fatal("must not treat platform client as tenant")
	}
	if !RegionGateApplies("gitlab-git-service-tencent-sh-1") {
		t.Fatal("region gate should apply to platform prefix")
	}
	if RegionGateApplies("gitlab-tenant-1") {
		t.Fatal("region gate must skip tenant clients")
	}
}

func TestRedirectURIAndHTTPS(t *testing.T) {
	uri, err := RedirectURIFromBaseURL("https://gitlab.daydaymoney.com/")
	if err != nil {
		t.Fatal(err)
	}
	if uri != "https://gitlab.daydaymoney.com/users/auth/openid_connect/callback" {
		t.Fatalf("got %q", uri)
	}
	if err := ValidateProductionRedirectURI(uri); err != nil {
		t.Fatal(err)
	}
	publicHTTP := "http://gitlab.daydaymoney.com/users/auth/openid_connect/callback"
	if err := ValidateProductionRedirectURI(publicHTTP); err != nil {
		t.Fatalf("http public should be allowed: %v", err)
	}
	if !RedirectURIIsHTTP(publicHTTP) {
		t.Fatal("expected HTTP callback")
	}
	if RedirectURIIsHTTP(uri) {
		t.Fatal("https callback must not be reported as HTTP")
	}
	loop := "http://127.0.0.1:8929/users/auth/openid_connect/callback"
	if err := ValidateProductionRedirectURI(loop); err != nil {
		t.Fatalf("loopback http should be allowed: %v", err)
	}
	if err := ValidateProductionRedirectURI("ftp://gitlab.daydaymoney.com/users/auth/openid_connect/callback"); !errors.Is(err, ErrInvalidRedirectScheme) {
		t.Fatalf("ftp: %v", err)
	}
}

func TestDecideAuthorizeMembership(t *testing.T) {
	ok := AuthorizeMembershipInput{UserID: "u1", OwnerCompanyID: "t1", IsMember: true}
	if err := DecideAuthorizeMembership(ok); err != nil {
		t.Fatal(err)
	}
	if err := DecideAuthorizeMembership(AuthorizeMembershipInput{UserID: "u1", OwnerCompanyID: "t1", IsMember: false}); !errors.Is(err, ErrAuthorizeMembershipDenied) {
		t.Fatalf("non-member: %v", err)
	}
	if err := DecideAuthorizeMembership(AuthorizeMembershipInput{UserID: "super", OwnerCompanyID: "t1", IsMember: false}); !errors.Is(err, ErrAuthorizeMembershipDenied) {
		t.Fatalf("superuser non-member: %v", err)
	}
	if err := DecideAuthorizeMembership(AuthorizeMembershipInput{UserID: "u1", OwnerCompanyID: "t1", IsMember: true, MembershipError: errors.New("down")}); !errors.Is(err, ErrAuthorizeMembershipDenied) {
		t.Fatalf("lookup fail: %v", err)
	}
}

func TestOmniAuthSnippet(t *testing.T) {
	tid := "877397588196749312"
	s := OmniAuthSnippet("https://api.daydaymoney.com", "gitlab-tenant-"+tid, "https://g.example/users/auth/openid_connect/callback", tid)
	for _, want := range []string{
		"discovery: false",
		"openid",
		"Daydaymoney SSO",
		"https://api.daydaymoney.com/api/oidc/" + tid + "/authorize",
		"https://api.daydaymoney.com/api/oidc/" + tid + "/token",
		"https://api.daydaymoney.com/api/oidc/" + tid + "/userinfo",
		"https://api.daydaymoney.com/api/oidc/" + tid + "/jwks",
		"gitlab-tenant-" + tid,
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("snippet missing %q:\n%s", want, s)
		}
	}
	// OPT-20260826-013: snippet carries a secret placeholder (never a real secret).
	if !strings.Contains(s, "secret: "+OmniAuthSecretPlaceholder) {
		t.Fatalf("snippet missing secret placeholder line:\n%s", s)
	}
	if strings.Count(s, "secret:") != 1 {
		t.Fatalf("snippet must contain exactly one secret line (the placeholder):\n%s", s)
	}
	if strings.Contains(s, `"https://api.daydaymoney.com/api/oidc/authorize"`) {
		t.Fatal("snippet must not use global authorize path")
	}
}

func TestCheckOidcPathTenant(t *testing.T) {
	if err := CheckOidcPathTenant("", "t1"); err != nil {
		t.Fatalf("global path: %v", err)
	}
	if err := CheckOidcPathTenant("t1", "t1"); err != nil {
		t.Fatalf("match: %v", err)
	}
	if err := CheckOidcPathTenant("t1", "t2"); !errors.Is(err, ErrOidcPathTenantMismatch) {
		t.Fatalf("mismatch: %v", err)
	}
	if err := CheckOidcPathTenant("t1", ""); !errors.Is(err, ErrOidcPathTenantMismatch) {
		t.Fatalf("empty owner: %v", err)
	}
}
