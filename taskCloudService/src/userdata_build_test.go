package main

import "testing"

func TestUserdataVerifyBaseURLPrefersPublicTaskAPIBase(t *testing.T) {
	prevPublic := cfg.PublicTaskAPIBase
	t.Cleanup(func() {
		cfg.PublicTaskAPIBase = prevPublic
	})

	cfg.PublicTaskAPIBase = "http://127.0.0.1:18081"
	got := userdataVerifyBaseURL()
	if got != "http://127.0.0.1:18081" {
		t.Fatalf("prefer gateway publicBase: got %q", got)
	}

	cfg.PublicTaskAPIBase = ""
	got = userdataVerifyBaseURL()
	if got != taskAPIEndpointPlaceholder {
		t.Fatalf("empty fallback placeholder: got %q", got)
	}
}
