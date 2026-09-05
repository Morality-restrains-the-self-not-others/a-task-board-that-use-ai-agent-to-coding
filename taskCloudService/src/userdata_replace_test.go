package main

import (
	"strings"
	"testing"
)

func templateFixture() string {
	return `#!/bin/bash
export ACCESS_TOKEN='__TASK2APP_ACCESS_TOKEN__'
export TASK_API_ENDPOINT='__TASK2APP_TASK_CLOUD_PREFIX__'
REGISTRY_URL="{REGISTRY_URL}"
CONTAINER_IMAGE='__TASK2APP_CONTAINER_IMAGE__'
curl -X POST "${TASK_API_ENDPOINT%/}/server-container-token/boot-progress/" -d "{\"access_token\":\"${ACCESS_TOKEN}\"}"
TASK2APP_UD_VERIFY_URL="__TASK2APP_TASK_API_ENDPOINT__/api/cloud/server-userdata-verify/abc/tenant_id/874176608758427648/workspace_id/ws_1/task_id/task_1/"
curl -fsS "$TASK2APP_UD_VERIFY_URL" >/dev/null 2>&1 || true
`
}

func eventDataFixture() map[string]interface{} {
	return map[string]interface{}{
		"container_image_url":        "registry.cn-qingdao.aliyuncs.com/ruandao/task2app-trae:x86_64-latest",
		"userdata_access_token":      "tok-abc123",
		"userdata_task_api_endpoint": "https://api.daydaymoney.com",
		"company_id":                 "874176608758427648",
		"workspace_id":               "ws_1",
		"task_id":                    "task_1",
		"trace_id":                   "trace-1",
		"comment_id":                 "cmt_1",
		"container_name":             "task_task_1_cmt_1",
	}
}

func TestReplaceUserdataRuntimePlaceholders(t *testing.T) {
	out, err := replaceUserdataRuntimePlaceholders(templateFixture(), eventDataFixture())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checks := []string{
		"tok-abc123",
		"registry.cn-qingdao.aliyuncs.com/ruandao/task2app-trae:x86_64-latest",
		"https://api.daydaymoney.com/api/tenant/874176608758427648/workspace/ws_1/task/task_1/comment/cmt_1/cloud",
		"https://api.daydaymoney.com/api/cloud/server-userdata-verify/abc/tenant_id/874176608758427648/workspace_id/ws_1/task_id/task_1/",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Errorf("replaced output missing %q\n---\n%s", want, out)
		}
	}
	// 新生成器已不再输出 REGISTRY_URL；fixture 仍含旧模板残留以覆盖存量脚本。
	// replace 不负责剥离未使用的 {REGISTRY_URL}；__TASK2APP_* 必须全部替换。
	if strings.Contains(out, "__TASK2APP_") {
		t.Errorf("replaced output still contains __TASK2APP_\n---\n%s", out)
	}
	// 存量模板中的 {REGISTRY_URL} 可保留；断言仅保证业务占位符已替换。
}

func TestReplaceUserdataRuntimePlaceholdersMissingRequiredSlot(t *testing.T) {
	ev := eventDataFixture()
	delete(ev, "userdata_access_token")
	_, err := replaceUserdataRuntimePlaceholders(templateFixture(), ev)
	if err == nil {
		t.Fatalf("expected error when access_token slot missing")
	}
	if !strings.Contains(err.Error(), "access_token") {
		t.Errorf("error should mention slot name: %v", err)
	}
}

// TestReplaceUserdataShellContainerNamePreserved 回归：不得把 ${CONTAINER_NAME}/
// ${containerName} 值注入为平台名或 "-"（会破坏 boot-progress 与 docker inspect）。
func TestReplaceUserdataShellContainerNamePreserved(t *testing.T) {
	ev := eventDataFixture()
	tmpl := `export COMMENT_ID='__TASK2APP_COMMENT_ID__'
PLACE=__TASK2APP_CONTAINER_NAME__
if [ -n "${CONTAINER_NAME:-}" ] && [ "${CONTAINER_NAME}" != "-" ]; then
  _extra="\"container_name\":\"${CONTAINER_NAME}\""
fi
STATUS=$(docker inspect -f '{{.State.Status}}' "${containerName}" 2>/dev/null)`
	out, err := replaceUserdataRuntimePlaceholders(tmpl, ev)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, "${containerName}") {
		t.Errorf("camelCase not normalized: %s", out)
	}
	if !strings.Contains(out, `inspect -f '{{.State.Status}}' "${CONTAINER_NAME}"`) {
		t.Errorf("expected shell CONTAINER_NAME ref: %s", out)
	}
	if !strings.Contains(out, `[ "${CONTAINER_NAME}" != "-" ]`) {
		t.Errorf("shell CONTAINER_NAME was value-injected: %s", out)
	}
	if !strings.Contains(out, "COMMENT_ID='cmt_1'") || !strings.Contains(out, "PLACE=task_task_1_cmt_1") {
		t.Errorf("placeholders not replaced: %s", out)
	}
}

func TestReplaceUserdataShellContainerNameEmptyDoesNotDashInject(t *testing.T) {
	ev := eventDataFixture()
	delete(ev, "comment_id")
	delete(ev, "container_name")
	tmpl := `COMMENT=__TASK2APP_COMMENT_ID__ NAME=__TASK2APP_CONTAINER_NAME__ TRACE=__TASK2APP_TRACE_ID__
docker inspect "${CONTAINER_NAME}"
docker top "${containerName}"`
	out, err := replaceUserdataRuntimePlaceholders(tmpl, ev)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "COMMENT=-") {
		t.Errorf("optional comment_id should be '-': %s", out)
	}
	if !strings.Contains(out, "NAME=task2app-container") {
		t.Errorf("optional container_name should be task2app-container: %s", out)
	}
	if !strings.Contains(out, "TRACE=trace-1") {
		t.Errorf("trace_id from event should replace: %s", out)
	}
	if strings.Contains(out, `inspect "-"`) ||
		strings.Contains(out, "inspect - ") ||
		strings.Contains(out, "inspect -\n") ||
		strings.HasSuffix(strings.TrimSpace(out), "inspect -") {
		t.Errorf("must not dash-inject into inspect: %s", out)
	}
	if !strings.Contains(out, `docker top "${CONTAINER_NAME}"`) {
		t.Errorf("expected normalized top target: %s", out)
	}
}

func TestReplaceUserdataTraceIDOptionalWhenMissing(t *testing.T) {
	ev := eventDataFixture()
	delete(ev, "trace_id")
	tmpl := "TRACE=__TASK2APP_TRACE_ID__"
	out, err := replaceUserdataRuntimePlaceholders(tmpl, ev)
	if err != nil {
		t.Fatalf("trace_id optional should not fail: %v", err)
	}
	if !strings.Contains(out, "TRACE=-") {
		t.Errorf("missing trace_id should become '-': %s", out)
	}
}

func TestReplaceUserdataCommentIDFromParentCommentID(t *testing.T) {
	ev := eventDataFixture()
	delete(ev, "comment_id")
	delete(ev, "container_name")
	ev["parent_comment_id"] = "cmt_parent"
	tmpl := "COMMENT=__TASK2APP_COMMENT_ID__ NAME=__TASK2APP_CONTAINER_NAME__"
	out, err := replaceUserdataRuntimePlaceholders(tmpl, ev)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "COMMENT=cmt_parent") {
		t.Errorf("parent_comment_id should fill comment slot: %s", out)
	}
	if !strings.Contains(out, "NAME=task_1_cmt_parent") {
		t.Errorf("container_name should derive from task+parent comment: %s", out)
	}
}

func TestReplaceUserdataOptionalSlotsEmpty(t *testing.T) {
	tmpl := "IMAGE=__TASK2APP_CONTAINER_IMAGE__ COMMENT=__TASK2APP_COMMENT_ID__ NAME=__TASK2APP_CONTAINER_NAME__"
	ev := eventDataFixture()
	ev["container_image_url"] = "repo/img:tag"
	delete(ev, "comment_id")
	delete(ev, "container_name")
	out, err := replaceUserdataRuntimePlaceholders(tmpl, ev)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "COMMENT=-") || !strings.Contains(out, "NAME=task2app-container") {
		t.Errorf("optional comment=- and container_name=task2app-container when empty: %s", out)
	}
}

func TestNormalizeUserdataContainerImageURLForDocker(t *testing.T) {
	cases := map[string]string{
		"https://hub.docker.com/_/nginx":                      "nginx",
		"https://hub.docker.com/_/nginx/tags":                 "nginx",
		"https://hub.docker.com/r/library/nginx/tags/latest":  "nginx:latest",
		"https://hub.docker.com/r/ruandao/task2app-trae/tags": "ruandao/task2app-trae",
		"registry.cn-qingdao.aliyuncs.com/ruandao/img:v1":     "registry.cn-qingdao.aliyuncs.com/ruandao/img:v1",
		// r/_/alpine 形态与 SSOT（taskEvents replace.go）行为一致：ns="_" 保留为 _/alpine
		"https://www.hub.docker.com/r/_/alpine/tags/3.19": "_/alpine:3.19",
	}
	for in, want := range cases {
		if got := normalizeUserdataContainerImageURLForDocker(in); got != want {
			t.Errorf("normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestReplaceUserdataTaskCloudPrefixOnly(t *testing.T) {
	tmpl := "export TASK_API_ENDPOINT='__TASK2APP_TASK_CLOUD_PREFIX__'"
	ev := eventDataFixture()
	ev["container_image_url"] = "repo/img:tag"
	delete(ev, "userdata_access_token")
	out, err := replaceUserdataRuntimePlaceholders(tmpl, ev)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "https://api.daydaymoney.com/api/tenant/874176608758427648/workspace/ws_1/task/task_1/comment/cmt_1/cloud"
	if !strings.Contains(out, want) {
		t.Errorf("output missing task cloud prefix %q: %s", want, out)
	}
}

func TestHardenUserdataDockerCEInstallRewritesLegacyUbuntu(t *testing.T) {
	tmpl := `else
    log "使用 Docker 容器运行时..."
    if [ "$OS_FAMILY" = "centos" ]; then
        yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
        yum install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
    elif [ "$OS_FAMILY" = "ubuntu" ]; then
        mkdir -p /etc/apt/keyrings
        curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
        echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
        apt-get update
        apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
    elif [ "$OS_FAMILY" = "debian" ]; then
        mkdir -p /etc/apt/keyrings
        curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
        echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
        apt-get update
        apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
    fi
fi
TOKEN=__TASK2APP_ACCESS_TOKEN__
`
	out, err := replaceUserdataRuntimePlaceholders(tmpl, eventDataFixture())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "mirrors.aliyun.com/docker-ce/linux/ubuntu") {
		t.Fatalf("expected aliyun mirror rewrite, got:\n%s", out)
	}
	if !strings.Contains(out, "mirrors.tuna.tsinghua.edu.cn/docker-ce/linux/ubuntu") {
		t.Fatalf("expected tuna mirror rewrite, got:\n%s", out)
	}
	if strings.Contains(out, "curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor") {
		t.Fatalf("legacy single-source curl|gpg should be rewritten:\n%s", out)
	}
	if !strings.Contains(out, "tok-abc123") {
		t.Fatalf("access token still replaced: %s", out)
	}
}

func TestHardenUserdataDockerCEInstallIdempotent(t *testing.T) {
	already := `for DOCKER_CE_MIRROR in \
          "https://mirrors.aliyun.com/docker-ce/linux/ubuntu" \
          "https://download.docker.com/linux/ubuntu"; do
            true
        done`
	got := hardenUserdataDockerCEInstall(already)
	if got != already {
		t.Fatalf("already-hardened script must be unchanged")
	}
}

func TestBuildUserdataTaskCloudPrefixIncludesComment(t *testing.T) {
	got := buildUserdataTaskCloudPrefix("https://api.daydaymoney.com", "t1", "w1", "k9", "cmt_1")
	want := "https://api.daydaymoney.com/api/tenant/t1/workspace/w1/task/k9/comment/cmt_1/cloud"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestReplaceUserdataTaskCloudPrefixRequiresComment(t *testing.T) {
	tmpl := "export TASK_API_ENDPOINT='__TASK2APP_TASK_CLOUD_PREFIX__'"
	ev := eventDataFixture()
	delete(ev, "comment_id")
	_, err := replaceUserdataRuntimePlaceholders(tmpl, ev)
	if err == nil {
		t.Fatal("expected error when comment_id missing")
	}
	if !strings.Contains(err.Error(), "comment_id") {
		t.Fatalf("error should mention comment_id, got %v", err)
	}
}

func TestBuildUserdataTaskCloudPrefixRequiresComment(t *testing.T) {
	got := buildUserdataTaskCloudPrefix("https://api.daydaymoney.com", "t1", "w1", "k9", "-")
	if got != "" {
		t.Fatalf("got %q want empty (no legacy …/task/{id}/cloud)", got)
	}
	got = buildUserdataTaskCloudPrefix("https://api.daydaymoney.com", "t1", "w1", "k9", "")
	if got != "" {
		t.Fatalf("got %q want empty", got)
	}
}
