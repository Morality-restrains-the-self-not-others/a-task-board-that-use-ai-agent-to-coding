package main

import "testing"

func TestAssertDjangoInternalAPIIsLoopback_AcceptsLocal(t *testing.T) {
	cases := []string{
		"",
		"http://127.0.0.1:8001",
		"http://localhost:8001",
		"http://[::1]:8001",
	}
	for _, raw := range cases {
		if err := assertDjangoInternalAPIIsLoopback(raw); err != nil {
			t.Fatalf("assertDjangoInternalAPIIsLoopback(%q) = %v, want nil", raw, err)
		}
	}
}

func TestAssertDjangoInternalAPIIsLoopback_RejectsPublicGateway(t *testing.T) {
	cases := []string{
		"https://api.daydaymoney.com",
		"https://api.daydaymoney.com/",
		"http://api.daydaymoney.com:443",
		"https://www.daydaymoney.com",
	}
	for _, raw := range cases {
		err := assertDjangoInternalAPIIsLoopback(raw)
		if err == nil {
			t.Fatalf("assertDjangoInternalAPIIsLoopback(%q) = nil, want error", raw)
		}
	}
}
