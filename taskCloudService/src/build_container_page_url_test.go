package main

import "testing"

func TestBuildContainerPageURLScoped(t *testing.T) {
	cfg := &CloudServerConfig{
		CompanyID:            "t1",
		WorkspaceID:          "w1",
		TaskID:               "task1",
		BusinessAPIEndpoint:  "https://biz.example.com/api",
		ServerURL:            "https://runtime.example.com/ui/tenant/t1/workspace/w1/task/task1/tok_x",
	}
	got := buildContainerPageURL(cfg)
	want := "https://biz.example.com/ui/tenant/t1/workspace/w1/task/task1/tok_x"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestNormalizeURLOriginScopedToken(t *testing.T) {
	origin, token := normalizeURLOrigin("http://10.0.0.1:8765/ui/tenant/t/workspace/w/task/task1/tok_z")
	if origin != "http://10.0.0.1:8765" {
		t.Fatalf("origin=%q", origin)
	}
	if token != "tok_z" {
		t.Fatalf("token=%q", token)
	}
}
