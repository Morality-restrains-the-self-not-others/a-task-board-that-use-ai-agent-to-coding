package infrastructure

import (
	"testing"

	"taskGitOauth/domain"
)

// 单一实现：domain.CanonicalTenantRedirectURI（OPT-20260812-036 去重后 infrastructure
// 调用 domain，不再维护第二份 canonicalizeTenantRedirectURI）。
func TestCanonicalizeTenantRedirectURI(t *testing.T) {
	tid := "875304088135299072"
	got := domain.CanonicalTenantRedirectURI(
		"https://daydaymoney.com/api/accounts/tenant-gitlab/oauth/callback/",
		tid,
	)
	want := "https://daydaymoney.com/api/accounts/tenant-" + tid + "/oauth/callback/"
	if got != want {
		t.Fatalf("got=%q want %q", got, want)
	}
	if domain.CanonicalTenantRedirectURI(want, tid) != want {
		t.Fatal("canonical rewrite must be idempotent")
	}
}

func TestTenantRowToProviderConfig_RewritesLegacyRedirectURI(t *testing.T) {
	tid := "co-legacy"
	pc, err := TenantRowToProviderConfig(&TenantGitLabOAuthConnectionRow{
		CompanyID:   tid,
		BaseURL:     "https://git.corp.example",
		ClientID:    "cid",
		RedirectURI: "https://daydaymoney.com/api/accounts/tenant-gitlab/oauth/callback/",
		Scope:       "api",
		Active:      true,
	}, nil)
	if err != nil || pc == nil {
		t.Fatalf("pc err=%v pc=%v", err, pc)
	}
	want := "https://daydaymoney.com/api/accounts/tenant-" + tid + "/oauth/callback/"
	if pc.RedirectURI != want {
		t.Fatalf("RedirectURI=%q want %q", pc.RedirectURI, want)
	}
}
