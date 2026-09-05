package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"taskGitOauth/domain"
	"taskGitOauth/infrastructure"
)

func TestTenantGitlabReachabilityUnconfigured(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	var probes atomic.Int32
	app.GitAPIDoFn = func(req *http.Request) (*http.Response, error) {
		probes.Add(1)
		return nil, http.ErrServerClosed
	}
	path := "/api/git-oauth/tenant-connection/tenant_id/company-empty/reachability/"
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-User-Id", "u1")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != domain.ReachabilityUnconfigured {
		t.Fatalf("status=%v", body["status"])
	}
	if probes.Load() != 0 {
		t.Fatalf("must not probe when unconfigured, probes=%d", probes.Load())
	}
}

func TestTenantGitlabReachabilityIntranetSkipsProbe(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	tid := "company-intranet"
	_, err := app.DB.UpsertTenantGitLabConnection(&infrastructure.TenantGitLabOAuthConnectionRow{
		CompanyID:       tid,
		BaseURL:         "http://10.0.0.8",
		ClientID:        "cid",
		ClientSecretEnc: "enc",
		RedirectURI:     "http://localhost/cb",
		Scope:           domain.DefaultTenantGitLabScope,
		Active:          true,
		Intranet:        true,
	})
	if err != nil {
		t.Fatal(err)
	}
	var probes atomic.Int32
	app.GitAPIDoFn = func(req *http.Request) (*http.Response, error) {
		probes.Add(1)
		return nil, http.ErrServerClosed
	}
	path := "/api/git-oauth/tenant-connection/tenant_id/" + tid + "/reachability/?url=http://127.0.0.1/"
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-User-Id", "u1")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["status"] != domain.ReachabilitySkippedIntranet {
		t.Fatalf("status=%v body=%v", body["status"], body)
	}
	if body["intranet"] != true {
		t.Fatalf("intranet=%v", body["intranet"])
	}
	if probes.Load() != 0 {
		t.Fatalf("intranet must skip probe, probes=%d", probes.Load())
	}
}

func TestTenantGitlabReachabilityPublicUnreachable(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	tid := "company-down"
	_, err := app.DB.UpsertTenantGitLabConnection(&infrastructure.TenantGitLabOAuthConnectionRow{
		CompanyID:       tid,
		BaseURL:         "http://115.29.110.74",
		ClientID:        "cid",
		ClientSecretEnc: "enc",
		RedirectURI:     "http://localhost/cb",
		Scope:           domain.DefaultTenantGitLabScope,
		Active:          true,
		Intranet:        false,
	})
	if err != nil {
		t.Fatal(err)
	}
	var probedURL string
	app.GitAPIDoFn = func(req *http.Request) (*http.Response, error) {
		probedURL = req.URL.String()
		return nil, &timeoutError{}
	}
	path := "/api/git-oauth/tenant-connection/tenant_id/" + tid + "/reachability/?url=http://127.0.0.1/"
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-User-Id", "u1")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["status"] != domain.ReachabilityUnreachable {
		t.Fatalf("status=%v body=%v", body["status"], body)
	}
	if body["reachable"] != false {
		t.Fatalf("reachable=%v", body["reachable"])
	}
	if body["reason"] != "timeout" {
		t.Fatalf("reason=%v", body["reason"])
	}
	if probedURL != "http://115.29.110.74/" {
		t.Fatalf("must probe stored base_url, got %q", probedURL)
	}
}

func TestTenantGitlabReachabilityHTTPAnyStatusIsReachable(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	tid := "company-up"
	_, err := app.DB.UpsertTenantGitLabConnection(&infrastructure.TenantGitLabOAuthConnectionRow{
		CompanyID:       tid,
		BaseURL:         "http://gitlab.daydaymoney.com",
		ClientID:        "cid",
		ClientSecretEnc: "enc",
		RedirectURI:     "http://localhost/cb",
		Scope:           domain.DefaultTenantGitLabScope,
		Active:          true,
	})
	if err != nil {
		t.Fatal(err)
	}
	app.GitAPIDoFn = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	}
	path := "/api/git-oauth/tenant-connection/tenant_id/" + tid + "/reachability/"
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-User-Id", "u1")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["status"] != domain.ReachabilityReachable {
		t.Fatalf("status=%v body=%v", body["status"], body)
	}
	if body["http_status"] != float64(401) {
		t.Fatalf("http_status=%v", body["http_status"])
	}
}

func TestTenantGitlabConnectionPutIntranetRoundTrip(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	tid := "company-flag"
	path := "/api/git-oauth/tenant-connection/tenant_id/" + tid + "/"
	putBody := `{"base_url":"https://gitlab.daydaymoney.com","client_id":"cid","client_secret":"sekrit","intranet":true}`
	req := httptest.NewRequest(http.MethodPut, path, strings.NewReader(putBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "admin1")
	req.Header.Set("X-Is-Admin", "1")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("PUT status %d body %s", rr.Code, rr.Body.String())
	}
	var saved map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &saved)
	if saved["intranet"] != true {
		t.Fatalf("PUT intranet=%v", saved["intranet"])
	}
	req = httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-User-Id", "u1")
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	var got map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	if got["intranet"] != true {
		t.Fatalf("GET intranet=%v", got["intranet"])
	}
}

type timeoutError struct{}

func (timeoutError) Error() string   { return "i/o timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }
