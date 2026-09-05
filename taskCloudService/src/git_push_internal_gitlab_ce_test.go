package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLayerGitPushPrepare_TencentSh1DoesNotBorrowDaydaymoneyToken(t *testing.T) {
	cfg.InternalSecret = ""
	prevTask := cfg.TaskServiceURL
	prevProj := cfg.ProjectServiceURL
	prevOauth := cfg.GitOauthBaseURL
	t.Cleanup(func() {
		cfg.TaskServiceURL = prevTask
		cfg.ProjectServiceURL = prevProj
		cfg.GitOauthBaseURL = prevOauth
	})
	cfg.TaskServiceURL = startTaskCommentGrantServer(t, "877397583960502272", "gitlab-tencent-sh-1.daydaymoney.com", nil).URL
	cfg.ProjectServiceURL = ""

	oauth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		_ = json.Unmarshal(body, &reqBody)
		pk, _ := reqBody["provider_key"].(string)
		if !strings.HasSuffix(r.URL.Path, "/oauth/access-for-user/") {
			http.NotFound(w, r)
			return
		}
		switch pk {
		case "gitlab:daydaymoney-gitlab":
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok_daydaymoney_wrong_ce"})
		case "gitlab:tencent-sh-1":
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok_tencent_sh1"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer oauth.Close()
	cfg.GitOauthBaseURL = oauth.URL

	body := `{"tenant_id":"t1","task_id":"task1","layer_id":"L1","user_id":"877397583960502272","prefer_container_remote":true,"repo_url":"https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad","target_branch":"feature/x"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/layer-git-push/prepare", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalLayerGitPushPrepare(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	pb, _ := out["push_body"].(map[string]any)
	oauthAuth, _ := pb["oauth_auth_by_repo"].(map[string]any)
	entry, _ := oauthAuth["https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad"].(map[string]any)
	if entry["provider_key"] != "gitlab:tencent-sh-1" {
		t.Fatalf("provider_key=%v oauth=%v (must not borrow gitlab:daydaymoney-gitlab)", entry["provider_key"], oauthAuth)
	}
	if entry["access_token"] != "tok_tencent_sh1" {
		t.Fatalf("access_token=%v", entry["access_token"])
	}
}
