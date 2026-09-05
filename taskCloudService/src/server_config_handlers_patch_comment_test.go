package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// OPT-20260814-017: PATCH server-config 若带 comment_id（评论级写入意图），
// 落库前须用 commentCSCCloudMetaIllegal 校验 —— mock/空 platform、空 region/auth 一律 400 不落库。

func TestPatchServerConfigWithCommentIDRejectsIllegalMeta(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudConfig(t, "t1", "ws1", "task1", "http://203.0.113.10:8080/")

	cases := []struct {
		name string
		body string
		want int
	}{
		{
			name: "comment_id + mock platform rejected",
			body: `{"comment_id":"cmt-rt","platform":"mock"}`,
			want: http.StatusBadRequest,
		},
		{
			name: "comment_id + empty region rejected",
			body: `{"comment_id":"cmt-rt","region":""}`,
			want: http.StatusBadRequest,
		},
		{
			name: "comment_id + empty authorization_id rejected",
			body: `{"comment_id":"cmt-rt","authorization_id":""}`,
			want: http.StatusBadRequest,
		},
		{
			name: "comment_id + legal override accepted",
			body: `{"comment_id":"cmt-rt","platform":"aliyun","region":"cn-hangzhou","authorization_id":"auth-2"}`,
			want: http.StatusOK,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPatch, "/api/tenant/t1/workspace/ws1/server-config/", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Auth-Tenant-Id", "t1")
			req.Header.Set("X-Workspace-Id", "ws1")
			req.Header.Set("X-Task-Id", "task1")
			rec := httptest.NewRecorder()
			handleServerConfigRoutes(rec, req, "t1", "ws1", "task1")
			if rec.Code != tc.want {
				t.Fatalf("status=%d body=%s, want %d", rec.Code, rec.Body.String(), tc.want)
			}
		})
	}
}

// 不带 comment_id 的任务级模板 PATCH 到 mock 仍应放行（任务模板可合法为 mock），
// 确认门控只作用于评论级写入意图。
func TestPatchServerConfigWithoutCommentIDAllowsMock(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudConfig(t, "t1", "ws1", "task1", "http://203.0.113.10:8080/")

	req := httptest.NewRequest(http.MethodPatch, "/api/tenant/t1/workspace/ws1/server-config/", strings.NewReader(`{"platform":"mock"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	req.Header.Set("X-Task-Id", "task1")
	rec := httptest.NewRecorder()
	handleServerConfigRoutes(rec, req, "t1", "ws1", "task1")
	if rec.Code != http.StatusOK {
		t.Fatalf("task-level mock patch should succeed, status=%d body=%s", rec.Code, rec.Body.String())
	}
}
