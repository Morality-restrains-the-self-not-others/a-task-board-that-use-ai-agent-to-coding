package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestIsInternalRepoGitlabLocal(t *testing.T) {
	oldResolver := providerResolver
	providerResolver = &ProviderResolver{
		entries: []providerEntry{
			{
				Provider:        "gitlab",
				ServiceProvider: "daydaymoney-gitlab",
				ProviderKey:     "gitlab:daydaymoney-gitlab",
				Host:            "gitlab.daydaymoney.com",
				Netloc:          "gitlab.daydaymoney.com",
			},
			{
				Provider:        "gitlab",
				ServiceProvider: "gitlab-local",
				ProviderKey:     "gitlab:gitlab-local",
				Host:            "gitlab.daydaymoney.com",
				Netloc:          "gitlab.daydaymoney.com",
			},
			{
				Provider:        "github",
				ServiceProvider: "github-official",
				ProviderKey:     "github:github-official",
				Host:            "github.com",
				Netloc:          "github.com",
			},
		},
	}
	t.Cleanup(func() { providerResolver = oldResolver })

	if !providerResolver.IsInternalRepo("https://gitlab.daydaymoney.com/g/ram-work.git") {
		t.Fatal("expected internal for gitlab-local host even when another gitlab provider shares host")
	}
	if providerResolver.IsInternalRepo("https://github.com/o/r.git") {
		t.Fatal("expected external for github")
	}
	if providerResolver.IsInternalRepo("https://gitlab.daydaymoney.com/g/r.git") {
		t.Fatal("expected external for non-local gitlab (fallback key)")
	}
}

func TestParseGitLabRepositorySizeBytes(t *testing.T) {
	size, ok := parseGitLabRepositorySizeBytes([]byte(`{"statistics":{"repository_size":1288490188}}`))
	if !ok || size != 1288490188 {
		t.Fatalf("got size=%d ok=%v", size, ok)
	}
	if _, ok := parseGitLabRepositorySizeBytes([]byte(`{"id":1}`)); ok {
		t.Fatal("expected missing statistics to fail")
	}
	if _, ok := parseGitLabRepositorySizeBytes([]byte(`{`)); ok {
		t.Fatal("expected invalid json to fail")
	}
}

func TestEnrichProjectGitRepoDiskSizesInternalOnly(t *testing.T) {
	clearDiskSizeCache()
	t.Cleanup(clearDiskSizeCache)

	var gitlabHits int32
	gitlabSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&gitlabHits, 1)
		if !strings.Contains(r.URL.RawQuery, "statistics=true") {
			t.Fatalf("missing statistics query: %s", r.URL.String())
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"statistics": map[string]interface{}{"repository_size": 2048},
		})
	}))
	defer gitlabSrv.Close()

	gitlabHost := strings.TrimPrefix(gitlabSrv.URL, "http://")

	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "tok-disk"})
	}))
	defer gitoauthSrv.Close()
	oldGitoauth := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldGitoauth })

	oldResolver := providerResolver
	providerResolver = &ProviderResolver{
		entries: []providerEntry{
			{
				Provider:        "gitlab",
				ServiceProvider: "gitlab-local",
				ProviderKey:     "gitlab:gitlab-local",
				Host:            strings.Split(gitlabHost, ":")[0],
				Netloc:          gitlabHost,
				GitoauthBase:    gitoauthSrv.URL,
			},
		},
	}
	t.Cleanup(func() { providerResolver = oldResolver })

	internalURL := gitlabSrv.URL + "/group/internal.git"
	externalURL := "https://github.com/o/external.git"
	detail := map[string]interface{}{
		"git_repos": []string{internalURL, externalURL},
		"git_repo_entries": []map[string]interface{}{
			{"url": internalURL, "clone_alias": ""},
			{"url": externalURL, "clone_alias": "ext"},
		},
	}

	enrichProjectGitRepoDiskSizes(detail, "1001")

	entries, ok := detail["git_repo_entries"].([]map[string]interface{})
	if !ok || len(entries) != 2 {
		t.Fatalf("entries=%#v", detail["git_repo_entries"])
	}
	if entries[0]["is_internal"] != true {
		t.Fatalf("internal is_internal=%#v", entries[0]["is_internal"])
	}
	if entries[0]["disk_size_bytes"] != int64(2048) {
		// JSON numbers from map may be int64
		switch v := entries[0]["disk_size_bytes"].(type) {
		case int64:
			if v != 2048 {
				t.Fatalf("disk_size_bytes=%v", v)
			}
		case int:
			if v != 2048 {
				t.Fatalf("disk_size_bytes=%v", v)
			}
		default:
			t.Fatalf("disk_size_bytes type=%T value=%#v", entries[0]["disk_size_bytes"], entries[0]["disk_size_bytes"])
		}
	}
	if entries[1]["is_internal"] != false {
		t.Fatalf("external is_internal=%#v", entries[1]["is_internal"])
	}
	if _, has := entries[1]["disk_size_bytes"]; has {
		t.Fatalf("external must not have disk_size_bytes: %#v", entries[1])
	}
	if atomic.LoadInt32(&gitlabHits) != 1 {
		t.Fatalf("gitlabHits=%d want 1 (external must not call)", gitlabHits)
	}
}

func TestEnrichProjectGitRepoDiskSizesUsesShortTTLCache(t *testing.T) {
	clearDiskSizeCache()
	t.Cleanup(clearDiskSizeCache)

	var gitlabHits int32
	gitlabSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&gitlabHits, 1)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"statistics": map[string]interface{}{"repository_size": 4096},
		})
	}))
	defer gitlabSrv.Close()
	gitlabHost := strings.TrimPrefix(gitlabSrv.URL, "http://")

	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "tok-disk"})
	}))
	defer gitoauthSrv.Close()
	oldGitoauth := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldGitoauth })

	oldResolver := providerResolver
	providerResolver = &ProviderResolver{
		entries: []providerEntry{
			{
				ProviderKey:  "gitlab:gitlab-local",
				Host:         strings.Split(gitlabHost, ":")[0],
				Netloc:       gitlabHost,
				GitoauthBase: gitoauthSrv.URL,
			},
		},
	}
	t.Cleanup(func() { providerResolver = oldResolver })

	internalURL := gitlabSrv.URL + "/group/cached.git"
	detail := func() map[string]interface{} {
		return map[string]interface{}{
			"git_repo_entries": []map[string]interface{}{
				{"url": internalURL, "clone_alias": ""},
			},
		}
	}

	enrichProjectGitRepoDiskSizes(detail(), "1001")
	enrichProjectGitRepoDiskSizes(detail(), "1001")
	if atomic.LoadInt32(&gitlabHits) != 1 {
		t.Fatalf("gitlabHits=%d want 1 after second enrich (cache miss)", gitlabHits)
	}

	// Expire cache → next enrich must hit GitLab again.
	oldNow := diskSizeNow
	diskSizeNow = func() time.Time { return time.Now().Add(diskSizeCacheTTL + time.Second) }
	t.Cleanup(func() { diskSizeNow = oldNow })
	enrichProjectGitRepoDiskSizes(detail(), "1001")
	if atomic.LoadInt32(&gitlabHits) != 2 {
		t.Fatalf("gitlabHits=%d want 2 after TTL expiry", gitlabHits)
	}
}

func TestEnrichProjectGitRepoDiskSizesExternalNeverHitsRemote(t *testing.T) {
	clearDiskSizeCache()
	t.Cleanup(clearDiskSizeCache)

	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	oldResolver := providerResolver
	providerResolver = &ProviderResolver{
		entries: []providerEntry{
			{
				ProviderKey: "github:github-official",
				Host:        "github.com",
				Netloc:      "github.com",
			},
		},
	}
	t.Cleanup(func() { providerResolver = oldResolver })

	detail := map[string]interface{}{
		"git_repo_entries": []map[string]interface{}{
			{"url": "https://github.com/o/r.git"},
		},
	}
	enrichProjectGitRepoDiskSizes(detail, "1001")
	if atomic.LoadInt32(&hits) != 0 {
		t.Fatalf("hits=%d", hits)
	}
	entries := detail["git_repo_entries"].([]map[string]interface{})
	if entries[0]["is_internal"] != false {
		t.Fatalf("is_internal=%#v", entries[0]["is_internal"])
	}
	if _, has := entries[0]["disk_size_bytes"]; has {
		t.Fatal("unexpected disk_size_bytes on external")
	}
}

func TestHandleProjectGitRepoDiskSizesEnrichesInternal(t *testing.T) {
	setupTestDB(t)
	clearDiskSizeCache()
	t.Cleanup(clearDiskSizeCache)

	var gitlabHits int32
	gitlabSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&gitlabHits, 1)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"statistics": map[string]interface{}{"repository_size": 2048},
		})
	}))
	defer gitlabSrv.Close()
	gitlabHost := strings.TrimPrefix(gitlabSrv.URL, "http://")

	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "tok-disk"})
	}))
	defer gitoauthSrv.Close()
	oldGitoauth := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldGitoauth })

	oldResolver := providerResolver
	providerResolver = &ProviderResolver{
		entries: []providerEntry{
			{
				Provider:        "gitlab",
				ServiceProvider: "gitlab-local",
				ProviderKey:     "gitlab:gitlab-local",
				Host:            strings.Split(gitlabHost, ":")[0],
				Netloc:          gitlabHost,
				GitoauthBase:    gitoauthSrv.URL,
			},
		},
	}
	t.Cleanup(func() { providerResolver = oldResolver })

	internalURL := gitlabSrv.URL + "/group/internal.git"
	body := `{"name":"disk-split","git_repos":["` + internalURL + `"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "1001")
	rec := httptest.NewRecorder()
	handleCreateProject(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&created)
	pid, _ := created["id"].(string)

	reqGet := httptest.NewRequest(http.MethodGet, "/api/projects/"+pid+"/tenant_id/t1/", nil)
	reqGet.Header.Set("X-Auth-Tenant-Id", "t1")
	reqGet.Header.Set("X-Auth-User-Id", "1001")
	reqGet.Header.Set("X-Resource-Id", pid)
	recGet := httptest.NewRecorder()
	handleGetProject(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("get: %d %s", recGet.Code, recGet.Body.String())
	}
	var page map[string]interface{}
	_ = json.NewDecoder(recGet.Body).Decode(&page)
	rawEntries, _ := page["git_repo_entries"].([]interface{})
	if len(rawEntries) != 1 {
		t.Fatalf("GET project entries=%#v", page["git_repo_entries"])
	}
	entry0, _ := rawEntries[0].(map[string]interface{})
	if _, has := entry0["disk_size_bytes"]; has {
		t.Fatalf("GET project must not include disk_size_bytes: %#v", entry0)
	}
	if atomic.LoadInt32(&gitlabHits) != 0 {
		t.Fatalf("GET project gitlabHits=%d want 0", gitlabHits)
	}

	reqDisk := httptest.NewRequest(http.MethodGet, "/api/projects/"+pid+"/git-repo-disk-sizes/tenant_id/t1/", nil)
	reqDisk.Header.Set("X-Auth-Tenant-Id", "t1")
	reqDisk.Header.Set("X-Auth-User-Id", "1001")
	recDisk := httptest.NewRecorder()
	handleProjectGitRepoDiskSizes(recDisk, reqDisk, "t1", pid)
	if recDisk.Code != http.StatusOK {
		t.Fatalf("disk-size: %d %s", recDisk.Code, recDisk.Body.String())
	}
	var diskPayload map[string]interface{}
	_ = json.NewDecoder(recDisk.Body).Decode(&diskPayload)
	diskEntries, _ := diskPayload["git_repo_entries"].([]interface{})
	if len(diskEntries) != 1 {
		t.Fatalf("disk entries=%#v", diskPayload["git_repo_entries"])
	}
	disk0, _ := diskEntries[0].(map[string]interface{})
	if disk0["is_internal"] != true {
		t.Fatalf("is_internal=%#v", disk0["is_internal"])
	}
	switch v := disk0["disk_size_bytes"].(type) {
	case float64:
		if v != 2048 {
			t.Fatalf("disk_size_bytes=%v", v)
		}
	case int64:
		if v != 2048 {
			t.Fatalf("disk_size_bytes=%v", v)
		}
	case json.Number:
		n, _ := v.Int64()
		if n != 2048 {
			t.Fatalf("disk_size_bytes=%v", v)
		}
	default:
		t.Fatalf("disk_size_bytes type=%T value=%#v", disk0["disk_size_bytes"], disk0["disk_size_bytes"])
	}
	if atomic.LoadInt32(&gitlabHits) != 1 {
		t.Fatalf("disk-size gitlabHits=%d want 1", gitlabHits)
	}
}
