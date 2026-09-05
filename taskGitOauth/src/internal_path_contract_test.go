package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// OPT-20260807-071 回归：internal access-for-user 新路径契约
// /api/internal/{provider}/oauth/access-for-user/（调用方 taskProjectService /
// taskCredentialService 使用；此前 taskGitOauth 只注册旧 /api/internal/git-oauth/ 路径
// 致 ServeMux 404 → token_error）。断言新路径命中 handler（空 user_id → 400 bad user_id，
// 而非 404 page not found）。
func TestAccessForUserNewPathAliasReachesHandler(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	for _, path := range []string{
		"/api/internal/github/oauth/access-for-user/",
		"/api/internal/gitlab/oauth/access-for-user/",
	} {
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(`{"user_id":""}`))
		req.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code == 404 {
			t.Fatalf("path %s: 404 page not found — new path alias not registered", path)
		}
		if rec.Code != 400 {
			t.Fatalf("path %s: expected 400 bad user_id, got %d body=%s", path, rec.Code, rec.Body.String())
		}
	}
}

// OPT-20260807-071 回归：雪花 ID（>2^53）经默认 encoding/json 解码为 float64 丢精度，
// 导致按 user_id 精确匹配 SQL 0 行 → not_found → token_error。
// readJSONNumbered 必须以 json.Number 保留原文；对照 readJSON 复现丢精度缺陷。
func TestReadJSONNumberedSnowflakePrecision(t *testing.T) {
	const snowflake = "873438061961179136"
	body := `{"user_id":` + snowflake + `,"provider_key":"github:x"}`

	legacy, err := readJSON(httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body)))
	if err != nil {
		t.Fatal(err)
	}
	gotLegacy := strings.TrimSpace(fmt.Sprint(legacy["user_id"]))
	if gotLegacy == snowflake {
		t.Fatalf("readJSON unexpectedly preserved precision (got %q) — float64 lossy path no longer lossy", gotLegacy)
	}

	got, err := readJSONNumbered(httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body)))
	if err != nil {
		t.Fatal(err)
	}
	uid := strings.TrimSpace(fmt.Sprint(got["user_id"]))
	if uid != snowflake {
		t.Fatalf("readJSONNumbered user_id=%q want %s (json.Number 应保留雪花 ID 原文)", uid, snowflake)
	}
}
