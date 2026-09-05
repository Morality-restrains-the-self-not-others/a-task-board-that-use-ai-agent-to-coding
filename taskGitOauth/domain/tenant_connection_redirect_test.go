package domain

import (
	"strings"
	"testing"
)

func TestDefaultRedirectURI_IncludesTenantID(t *testing.T) {
	tidA := "875304088135299072"
	tidB := "874599492341493760"
	uriA := DefaultRedirectURI("https://daydaymoney.com", tidA)
	uriB := DefaultRedirectURI("https://daydaymoney.com", tidB)

	wantA := "https://daydaymoney.com/api/accounts/tenant-" + tidA + "/oauth/callback/"
	wantB := "https://daydaymoney.com/api/accounts/tenant-" + tidB + "/oauth/callback/"
	if uriA != wantA {
		t.Fatalf("uriA=%q want %q", uriA, wantA)
	}
	if uriB != wantB {
		t.Fatalf("uriB=%q want %q", uriB, wantB)
	}
	if uriA == uriB {
		t.Fatal("different tenants must get different redirect URIs")
	}
	if strings.Contains(uriA, "/api/accounts/tenant-gitlab/") {
		t.Fatalf("shared tenant-gitlab callback must not be used: %s", uriA)
	}
}

func TestDefaultRedirectURI_TrimsBaseAndEmptyFallback(t *testing.T) {
	got := DefaultRedirectURI("https://example.com/", "42")
	if got != "https://example.com/api/accounts/tenant-42/oauth/callback/" {
		t.Fatalf("got=%q", got)
	}
	// Empty company id keeps legacy shared callback for safety/compat.
	legacy := DefaultRedirectURI("https://example.com", "")
	if legacy != "https://example.com/api/accounts/tenant-gitlab/oauth/callback/" {
		t.Fatalf("legacy empty id=%q", legacy)
	}
	emptyBase := DefaultRedirectURI("", "99")
	if emptyBase != "http://127.0.0.1:8002/api/accounts/tenant-99/oauth/callback/" {
		t.Fatalf("emptyBase=%q", emptyBase)
	}
}

func TestCanonicalTenantRedirectURI_RewritesSharedCallback(t *testing.T) {
	tid := "875304088135299072"
	stored := "https://daydaymoney.com/api/accounts/tenant-gitlab/oauth/callback/"
	got := CanonicalTenantRedirectURI(stored, tid)
	want := "https://daydaymoney.com/api/accounts/tenant-" + tid + "/oauth/callback/"
	if got != want {
		t.Fatalf("got=%q want %q", got, want)
	}
	// Already canonical stays stable.
	if CanonicalTenantRedirectURI(want, tid) != want {
		t.Fatalf("idempotent rewrite failed: %q", CanonicalTenantRedirectURI(want, tid))
	}
}
