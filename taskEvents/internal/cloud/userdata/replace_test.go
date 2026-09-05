package userdata

import (
	"fmt"
	"strings"
	"testing"
)

func TestReplaceRuntimePlaceholdersContainerImage(t *testing.T) {
	out, err := ReplaceRuntimePlaceholders(
		`CONTAINER_IMAGE="__TASK2APP_CONTAINER_IMAGE__"`,
		ReplaceParams{ContainerImageURL: "registry.cn-hangzhou.aliyuncs.com/ruandao/task2app-trae:v1"},
	)
	if err != nil {
		t.Fatal(err)
	}
	want := `CONTAINER_IMAGE="registry.cn-hangzhou.aliyuncs.com/ruandao/task2app-trae:v1"`
	if out != want {
		t.Fatalf("got %q want %q", out, want)
	}
}

func TestReplaceRuntimePlaceholdersLongestTokenFirst(t *testing.T) {
	text := "A=__TASK2APP_CONTAINER_IMAGE__ B=TASK2APP_CONTAINER_IMAGE"
	out, err := ReplaceRuntimePlaceholders(text, ReplaceParams{ContainerImageURL: "img:1"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "A=img:1 B=img:1" {
		t.Fatalf("got %q", out)
	}
}

func TestReplaceRuntimePlaceholdersAllSlots(t *testing.T) {
	text := "I=__TASK2APP_CONTAINER_IMAGE__ T=__TASK2APP_ACCESS_TOKEN__ E=__TASK2APP_TASK_API_ENDPOINT__"
	out, err := ReplaceRuntimePlaceholders(text, ReplaceParams{
		ContainerImageURL: "img:1",
		AccessToken:       "tok",
		TaskAPIEndpoint:   "http://api",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != "I=img:1 T=tok E=http://api" {
		t.Fatalf("got %q", out)
	}
}

func TestReplaceRuntimePlaceholdersRequiresAccessToken(t *testing.T) {
	_, err := ReplaceRuntimePlaceholders("__TASK2APP_ACCESS_TOKEN__", ReplaceParams{})
	if err == nil || !strings.Contains(err.Error(), "access_token") {
		t.Fatalf("expected access_token error, got %v", err)
	}
}

func TestNormalizeContainerImageURLForDockerHub(t *testing.T) {
	if got := NormalizeContainerImageURLForDocker("https://hub.docker.com/_/nginx"); got != "nginx" {
		t.Fatalf("got %q", got)
	}
	out, err := ReplaceRuntimePlaceholders(
		"docker run __TASK2APP_CONTAINER_IMAGE__",
		ReplaceParams{ContainerImageURL: "https://hub.docker.com/_/nginx"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if out != "docker run nginx" {
		t.Fatalf("got %q", out)
	}
}

func TestReplaceRuntimePlaceholdersTaskCloudPrefixRequiresComment(t *testing.T) {
	plain := `TASK_API_ENDPOINT="__TASK2APP_TASK_CLOUD_PREFIX__"`
	_, err := ReplaceRuntimePlaceholders(plain, ReplaceParams{
		TaskAPIEndpoint: "https://api.daydaymoney.com",
		TenantID:        "t1",
		WorkspaceID:     "w1",
		TaskID:          "k9",
	})
	if err == nil {
		t.Fatal("expected error when comment_id missing")
	}
	if !strings.Contains(err.Error(), "comment_id") {
		t.Fatalf("error should mention comment_id, got %v", err)
	}
}

func TestReplaceRuntimePlaceholdersTaskCloudPrefixWithComment(t *testing.T) {
	plain := `TASK_API_ENDPOINT="__TASK2APP_TASK_CLOUD_PREFIX__"`
	out, err := ReplaceRuntimePlaceholders(plain, ReplaceParams{
		TaskAPIEndpoint: "https://api.daydaymoney.com",
		TenantID:        "t1",
		WorkspaceID:     "w1",
		TaskID:          "k9",
		CommentID:       "cmt_1",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `TASK_API_ENDPOINT="https://api.daydaymoney.com/api/tenant/t1/workspace/w1/task/k9/comment/cmt_1/cloud"`
	if out != want {
		t.Fatalf("got %q want %q", out, want)
	}
}

func TestRewriteLocalhostVerifyHosts(t *testing.T) {
	// 44efa06 约定迁移后 verify 回调走 /api/cloud/ 约定路径（kv-last scope）
	//（taskGateway server-userdata-verify 路由恢复，OPT-20260809-024 同批）。
	plain := `TASK2APP_UD_VERIFY_URL="http://127.0.0.1:9999/api/cloud/server-userdata-verify/tok/tenant_id/x/workspace_id/y/task_id/z/"`
	out, err := ReplaceRuntimePlaceholders(plain, ReplaceParams{
		TaskAPIEndpoint: "https://public.task2app.test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "127.0.0.1") {
		t.Fatalf("localhost not rewritten: %q", out)
	}
	if !strings.Contains(out, "https://public.task2app.test/api/cloud/server-userdata-verify/tok/tenant_id/x/workspace_id/y/task_id/z/") {
		t.Fatalf("unexpected: %q", out)
	}
}

// TestNormalizeShellContainerNameRefs 回归：存量模板残留 ${containerName} 须改写为
// ${CONTAINER_NAME}（shell 变量），不得注入平台路由名或 "-"（否则 docker inspect -
// / report_progress 条件恒假 → 启动日志缺失且状态卡在「启动中」）。
func TestNormalizeShellContainerNameRefs(t *testing.T) {
	plain := `export CONTAINER_NAME='task2app-container'
STATUS=$(docker inspect -f '{{.State.Status}}' "${containerName}" 2>/dev/null)
if [ -n "${CONTAINER_NAME:-}" ] && [ "${CONTAINER_NAME}" != "-" ]; then echo ok; fi
PLACE=__TASK2APP_CONTAINER_NAME__`
	out, err := ReplaceRuntimePlaceholders(plain, ReplaceParams{
		ContainerName: "task_task_x_cmt-9",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, "${containerName}") {
		t.Errorf("camelCase shell ref not normalized: %s", out)
	}
	if !strings.Contains(out, `inspect -f '{{.State.Status}}' "${CONTAINER_NAME}"`) {
		t.Errorf("expected shell CONTAINER_NAME ref, got: %s", out)
	}
	// 正式模板中的 ${CONTAINER_NAME} 必须保留给 shell，禁止被值替换成平台名
	if !strings.Contains(out, `[ "${CONTAINER_NAME}" != "-" ]`) {
		t.Errorf("shell CONTAINER_NAME was value-injected; got: %s", out)
	}
	if !strings.Contains(out, "PLACE=task_task_x_cmt-9") {
		t.Errorf("__TASK2APP_CONTAINER_NAME__ should still be value-replaced: %s", out)
	}
}

// TestReplaceOptionalContainerNameDashDoesNotBreakShellVars 空 container_name 时
// __TASK2APP_CONTAINER_NAME__ → task2app-container（稳定默认），不得改写 shell ${CONTAINER_NAME}。
func TestReplaceOptionalContainerNameDashDoesNotBreakShellVars(t *testing.T) {
	plain := `export COMMENT_ID='__TASK2APP_COMMENT_ID__'
export NAME='__TASK2APP_CONTAINER_NAME__'
if [ "${CONTAINER_NAME}" != "-" ]; then _extra="\"container_name\":\"${CONTAINER_NAME}\""; fi
docker inspect "${containerName}"`
	out, err := ReplaceRuntimePlaceholders(plain, ReplaceParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "COMMENT_ID='-'") {
		t.Errorf("optional comment_id should become '-': %s", out)
	}
	if !strings.Contains(out, "NAME='task2app-container'") {
		t.Errorf("optional container_name should become task2app-container: %s", out)
	}
	if strings.Contains(out, `[ "-" != "-" ]`) {
		t.Errorf("shell ${CONTAINER_NAME} must not be replaced with '-': %s", out)
	}
	if strings.Contains(out, `docker inspect "-"`) ||
		strings.Contains(out, "docker inspect - ") ||
		strings.Contains(out, "docker inspect -\n") {
		t.Errorf("inspect must not target literal '-': %s", out)
	}
	if !strings.Contains(out, `docker inspect "${CONTAINER_NAME}"`) {
		t.Errorf("expected normalized inspect target: %s", out)
	}
}

func TestApplyRunInstancesUserdataPlaceholdersFromEvent(t *testing.T) {
	data := map[string]interface{}{
		"container_image_url":        "registry.example.com/app:2.0",
		"userdata_access_token":      "access-tok",
		"userdata_task_api_endpoint": "https://api.daydaymoney.com",
		"company_id":                 "850256677331562496",
		"workspace_id":               "861623708318031872",
		"task_id":                    "task_12590983282794675865",
		"parent_comment_id":          "cmt-42",
	}
	plain := `pull __TASK2APP_CONTAINER_IMAGE__
token __TASK2APP_ACCESS_TOKEN__
api __TASK2APP_TASK_API_ENDPOINT__
name __TASK2APP_CONTAINER_NAME__`
	out, err := ApplyRunInstancesUserdataPlaceholders(plain, data, testStrField)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "__TASK2APP_") {
		t.Fatalf("placeholders remain: %q", out)
	}
	if !strings.Contains(out, "registry.example.com/app:2.0") {
		t.Fatalf("missing image ref: %q", out)
	}
	if !strings.Contains(out, "task_12590983282794675865_cmt-42") {
		t.Fatalf("missing derived container name: %q", out)
	}
}

func TestHardenDockerCEInstallRewritesLegacyUbuntu(t *testing.T) {
	tmpl := `elif [ "$OS_FAMILY" = "ubuntu" ]; then
        mkdir -p /etc/apt/keyrings
        curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
        echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
        apt-get update
        apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
TOKEN=__TASK2APP_ACCESS_TOKEN__`
	out, err := ReplaceRuntimePlaceholders(tmpl, ReplaceParams{
		AccessToken:     "tok-x",
		TaskAPIEndpoint: "https://api.daydaymoney.com",
		TenantID:        "1",
		WorkspaceID:     "2",
		TaskID:          "3",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "mirrors.aliyun.com/docker-ce/linux/ubuntu") {
		t.Fatalf("expected aliyun mirror: %s", out)
	}
	if strings.Contains(out, "curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor") {
		t.Fatalf("legacy curl|gpg remained: %s", out)
	}
}

func testStrField(body map[string]interface{}, key string) string {
	v, ok := body[key]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(strings.TrimSpace(fmt.Sprint(v)))
}
