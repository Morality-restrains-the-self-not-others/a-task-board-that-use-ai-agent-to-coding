package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
)

const taskAPIEndpointPlaceholder = "__TASK2APP_TASK_API_ENDPOINT__"

func generateVerificationSecret() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func buildVerificationShellSnippet(companyID, workspaceID, taskID, secret string) string {
	// 44efa06 约定迁移后 taskCloudService 仅挂 /api/cloud/ 约定路径（kv-last scope），
	// legacy /api/tenant/{tid}/... 形态一律 404 —— verify 回调 URL 同步迁移
	// （OPT-20260809-024 同批；对应 taskGateway 恢复的 server-userdata-verify 路由）。
	url := fmt.Sprintf(
		"%s/api/cloud/server-userdata-verify/%s/tenant_id/%s/workspace_id/%s/task_id/%s/",
		taskAPIEndpointPlaceholder, secret, companyID, workspaceID, taskID,
	)
	return "\n# ==== Auto verification callback ====\n" +
		fmt.Sprintf("TASK2APP_UD_VERIFY_URL=\"%s\"\n", url) +
		"if command -v curl >/dev/null 2>&1; then\n" +
		"  curl -fsS \"$TASK2APP_UD_VERIFY_URL\" >/dev/null 2>&1 || true\n" +
		"elif command -v wget >/dev/null 2>&1; then\n" +
		"  wget -q -O /dev/null \"$TASK2APP_UD_VERIFY_URL\" || true\n" +
		"fi\n" +
		"# ==== End auto verification callback ====\n"
}

func buildUserdataExecutionWrapperScript(userContent, verifyScript string) string {
	userText := strings.TrimRight(userContent, " \t\n\r")
	verifyText := strings.TrimSpace(verifyScript)
	mergedInner := "#!/bin/bash\nset -e\n# >>> TASK2APP_USER_SCRIPT_BEGIN\n" +
		userText + "\n# <<< TASK2APP_USER_SCRIPT_END\n\n" + verifyText + "\n"
	return "#!/bin/bash\nset -e\n# >>> TASK2APP_USERDATA_EXEC_WRAPPER_BEGIN\n" +
		"cat > /root/init_from_task2app.sh <<'TASK2APP_INIT_EOF'\n" +
		mergedInner +
		"TASK2APP_INIT_EOF\n" +
		"chmod +x /root/init_from_task2app.sh\n" +
		"/root/init_from_task2app.sh > /root/init_from_task2app.sh.log 2>&1\n" +
		"# <<< TASK2APP_USERDATA_EXEC_WRAPPER_END\n"
}

func userdataVerifyBaseURL() string {
	// Prefer gateway publicBase so remote containers hit APISIX → taskAgentSupport.
	// OPT-052: DjangoInternalAPI fallback removed (saas-backend retired 2026-07-30).
	if base := strings.TrimRight(strings.TrimSpace(cfg.PublicTaskAPIBase), "/"); base != "" {
		return base
	}
	return taskAPIEndpointPlaceholder
}
