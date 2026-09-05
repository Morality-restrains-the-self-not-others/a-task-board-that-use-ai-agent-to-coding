package main

import "testing"

func TestStripTenantIDKV_ConventionGitlabRegions(t *testing.T) {
	got := stripTenantIDKV("/api/billing/gitlab-regions/tenant_id/877397588196749312/")
	want := "/api/billing/gitlab-regions/"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestStripTenantIDKV_ConventionGitlabResourcesPurchase(t *testing.T) {
	got := stripTenantIDKV("/api/billing/gitlab-resources/tenant_id/1/purchase/")
	want := "/api/billing/gitlab-resources/purchase/"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestStripTenantIDKV_LegacyTenantPathUnchanged(t *testing.T) {
	in := "/api/tenant/877397588196749312/billing/gitlab-regions/"
	got := stripTenantIDKV(in)
	if got != in {
		t.Fatalf("legacy path mutated: got %q", got)
	}
}

func TestParseTenantID_ConventionKV(t *testing.T) {
	tid, ok := parseTenantID("/api/billing/gitlab-resources/tenant_id/877397588196749312/")
	if !ok || tid != 877397588196749312 {
		t.Fatalf("tid=%d ok=%v", tid, ok)
	}
}

func TestParseTenantID_LegacyPositional(t *testing.T) {
	tid, ok := parseTenantID("/api/tenant/877397588196749312/billing/gitlab-resources/")
	if !ok || tid != 877397588196749312 {
		t.Fatalf("tid=%d ok=%v", tid, ok)
	}
}

func TestParseTenantID_Missing(t *testing.T) {
	if _, ok := parseTenantID("/api/billing/gitlab-regions/"); ok {
		t.Fatal("expected no tenant")
	}
}
