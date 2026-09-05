package infrastructure

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchAccessTokenAlwaysUsesGitsitePath(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "glpat-path-a", "expires_in": 3600})
	}))
	t.Cleanup(srv.Close)

	client := NewGitoauthHTTPClient(srv.URL, 5, nil, "bridge")
	tok, err := client.FetchAccessToken(42, "115.29.110.74")
	if err != nil {
		t.Fatalf("FetchAccessToken: %v", err)
	}
	if tok != "glpat-path-a" {
		t.Fatalf("token=%q", tok)
	}
	if gotPath != "/api/internal/gitsite/115.29.110.74/oauth/access-for-user/" {
		t.Fatalf("path=%q", gotPath)
	}
	if _, has := gotBody["provider_key"]; has {
		t.Fatalf("gitsite body must omit provider_key, got %v", gotBody)
	}
	if gotBody["user_id"] != float64(42) {
		t.Fatalf("user_id=%v want 42", gotBody["user_id"])
	}
}

func TestFetchAccessTokenGithubComGoesThroughGitsitePath(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "ghp-tok"})
	}))
	t.Cleanup(srv.Close)

	client := NewGitoauthHTTPClient(srv.URL, 5, nil, "")
	tok, err := client.FetchAccessToken(1, "github.com")
	if err != nil {
		t.Fatal(err)
	}
	if tok != "ghp-tok" {
		t.Fatalf("token=%q", tok)
	}
	if gotPath != "/api/internal/gitsite/github.com/oauth/access-for-user/" {
		t.Fatalf("path=%q", gotPath)
	}
}

func TestFetchAccessTokenSiteWithPortKeepsHostColonPort(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path // colon is a legal path char; server splits on "/" then PathUnescape
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok"})
	}))
	t.Cleanup(srv.Close)

	client := NewGitoauthHTTPClient(srv.URL, 5, nil, "")
	if _, err := client.FetchAccessToken(1, "localhost:8012"); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/api/internal/gitsite/localhost:8012/oauth/access-for-user/" {
		t.Fatalf("path=%q want host:port site preserved", gotPath)
	}
}

func TestFetchAccessTokenEmptySiteRejected(t *testing.T) {
	client := NewGitoauthHTTPClient("http://unused", 5, nil, "")
	if _, err := client.FetchAccessToken(1, "   "); err == nil {
		t.Fatal("expected error for empty site")
	}
}
