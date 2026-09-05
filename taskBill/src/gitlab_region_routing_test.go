package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequireRegionSlug_EmptyRejected(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/x", nil)
	_, err := requireRegionSlug(r, "")
	if err == nil {
		t.Fatal("expected error when region missing")
	}
	if !strings.Contains(err.Error(), "region") {
		t.Fatalf("error should mention region, got %v", err)
	}
}

func TestRequireRegionSlug_QueryWins(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/x?region=tencent-sh-1", nil)
	got, err := requireRegionSlug(r, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "tencent-sh-1" {
		t.Fatalf("got %q", got)
	}
}

func TestRequireRegionSlug_BodyFallback(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/x", nil)
	got, err := requireRegionSlug(r, "tencent-shanghai-5")
	if err != nil {
		t.Fatal(err)
	}
	if got != "tencent-shanghai-5" {
		t.Fatalf("got %q", got)
	}
}

func TestEnsureTenantGitlabGroupForRegion_UsesRegionBaseAndToken(t *testing.T) {
	var sawToken, sawPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawToken = r.Header.Get("PRIVATE-TOKEN")
		sawPath = r.URL.Path
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/groups/") {
			http.NotFound(w, r)
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/api/v4/groups" {
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 9})
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	region := &GitlabRegion{
		Slug:              "tencent-sh-1",
		GitlabAPIBase:     srv.URL,
		AdminPrivateToken: "pat-sh-1",
	}
	err := ensureTenantGitlabGroupForRegion(context.Background(), 42, 1024, region)
	if err != nil {
		t.Fatal(err)
	}
	if sawToken != "pat-sh-1" {
		t.Fatalf("token=%q", sawToken)
	}
	if !strings.Contains(sawPath, "groups") {
		t.Fatalf("path=%q", sawPath)
	}
}

func TestEnsureTenantGitlabGroupForRegion_MissingToken(t *testing.T) {
	err := ensureTenantGitlabGroupForRegion(context.Background(), 1, 1, &GitlabRegion{Slug: "x", GitlabAPIBase: "http://127.0.0.1:1"})
	if err == nil {
		t.Fatal("expected missing token error")
	}
}

func TestCreateOrder_GitlabDiskRequiresRegion(t *testing.T) {
	_, _, err := createOrder(context.Background(), 1, []orderItemInput{
		{ResourceType: ResourceTypeGitlabDisk, Quantity: 1},
	})
	if err == nil || !strings.Contains(err.Error(), "region") {
		t.Fatalf("expected region required, got %v", err)
	}
}

func TestGetGitlabRegionBySlug_EmptyRejected(t *testing.T) {
	_, err := getGitlabRegionBySlug("")
	if err == nil {
		t.Fatal("expected error")
	}
}
