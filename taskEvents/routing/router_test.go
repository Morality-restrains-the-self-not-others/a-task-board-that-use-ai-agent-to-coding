package routing

import (
	"testing"

	"taskEvents/config"
)

func TestDispatchRouterPaths(t *testing.T) {
	router := DispatchRouter("accounts", []string{"USER_CREATED", "COMPANY_CREATED"})
	if router.Domain != "accounts" {
		t.Fatalf("domain=%q", router.Domain)
	}
	want := config.DispatchPath("accounts")
	for _, et := range []string{"USER_CREATED", "COMPANY_CREATED"} {
		r, ok := router.Routes[et]
		if !ok {
			t.Fatalf("missing route %s", et)
		}
		if r.Path != want {
			t.Fatalf("%s path=%q want %q", et, r.Path, want)
		}
	}
}
