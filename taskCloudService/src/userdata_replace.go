package main

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// userdata 运行时占位符替换 — 与 taskEvents/internal/cloud/userdata/replace.go 对齐的移植版。
// SSOT：taskEvents/internal/cloud/userdata/replace.go（ReplaceRuntimePlaceholders）。
// taskEvents/internal 包受 Go internal 规则限制无法跨模块导入，故此处按相同 token 表移植；
// 修改 token 表/替换规则时两处须同步。
//
// 背景（OPT-20260809-024）：cloud_server_events 持久化的 userdata_content 是镜像市场模板，
// 含 __TASK2APP_* 占位符；RunInstances 前未替换导致 VM 上 docker pull 拉取字面量
// "__TASK2APP_CONTAINER_IMAGE__" 失败，cloud-init scripts-user 失败、容器永不启动。

const (
	userdataTaskAPIEndpointPlaceholder  = "__TASK2APP_TASK_API_ENDPOINT__"
	userdataTaskCloudPrefixPlaceholder  = "__TASK2APP_TASK_CLOUD_PREFIX__"
	userdataContainerImagePlaceholder   = "__TASK2APP_CONTAINER_IMAGE__"
	userdataAccessTokenPlaceholder      = "__TASK2APP_ACCESS_TOKEN__"
	userdataInstalledImageIDPlaceholder = "__TASK2APP_INSTALLED_IMAGE_ID__"
	userdataCommentIDPlaceholder        = "__TASK2APP_COMMENT_ID__"
	userdataContainerNamePlaceholder    = "__TASK2APP_CONTAINER_NAME__"
)

var userdataContainerImageTokens = []string{
	"__TASK2APP_CONTAINER_IMAGE__",
	"__TASK2APP_CONTAINER_IMAGE_URL__",
	"{{CONTAINER_IMAGE}}",
	"{{CONTAINER_IMAGE_URL}}",
	"${CONTAINER_IMAGE}",
	"${CONTAINER_IMAGE_URL}",
	"__CONTAINER_IMAGE__",
	"__CONTAINER_IMAGE_URL__",
	"TASK2APP_CONTAINER_IMAGE",
	"TASK2APP_CONTAINER_IMAGE_URL",
}

type userdataRuntimeSlot struct {
	name   string
	tokens []string
}

var userdataRuntimeSlots = []userdataRuntimeSlot{
	{name: "container_image_url", tokens: userdataContainerImageTokens},
	{name: "access_token", tokens: []string{"__TASK2APP_ACCESS_TOKEN__"}},
	{name: "task_api_endpoint", tokens: []string{userdataTaskAPIEndpointPlaceholder}},
	{name: "tenant_id", tokens: []string{"__TASK2APP_TENANT_ID__"}},
	{name: "workspace_id", tokens: []string{"__TASK2APP_WORKSPACE_ID__"}},
	{name: "task_id", tokens: []string{"__TASK2APP_TASK_ID__"}},
	{name: "trace_id", tokens: []string{"__TASK2APP_TRACE_ID__"}},
	{name: "ssh_public_key", tokens: []string{"__TASK2APP_SSH_PUBLIC_KEY__"}},
	{name: "ssh_match_address", tokens: []string{"__TASK2APP_SSH_MATCH_ADDRESS__"}},
	// Optional log-correlation slots: empty → "-"（绝不阻断 RunInstances）。
	{name: "installed_image_id", tokens: []string{userdataInstalledImageIDPlaceholder}},
	{name: "comment_id", tokens: []string{userdataCommentIDPlaceholder}},
	// 仅替换 __TASK2APP_CONTAINER_NAME__。禁止值注入 ${CONTAINER_NAME}/${containerName}
	//（会破坏 report_progress / docker inspect 的 shell 变量；空值→"-" 导致启动卡死）。
	// 存量 camelCase 残留见 normalizeUserdataShellContainerNameRefs。
	{name: "container_name", tokens: []string{userdataContainerNamePlaceholder}},
}

var userdataOptionalRuntimeSlots = map[string]bool{
	"installed_image_id": true,
	"comment_id":         true,
	"container_name":     true,
	"trace_id":           true, // 缺失时写 "-"，不阻断 RunInstances
}

// replaceUserdataRuntimePlaceholders 在 RunInstances 前将 userdata 模板中的
// __TASK2APP_* 占位符替换为 eventData 中的真实运行时值（镜像 URL、访问令牌、
// 平台 API 端点、租户/工作空间/任务标识等）。
func replaceUserdataRuntimePlaceholders(plain string, eventData map[string]interface{}) (string, error) {
	if strings.TrimSpace(plain) == "" {
		return plain, nil
	}
	p := userdataReplaceParamsFromEvent(eventData)

	slotsNeeded := matchedUserdataRuntimeSlots(plain)
	if len(slotsNeeded) == 0 {
		out := applyUserdataCloudPrefixRewrite(plain, p)
		out = rewriteUserdataLocalhostVerifyHosts(out, strings.TrimSpace(p.taskAPIEndpoint))
		return injectUserdataImageEnvs(hardenUserdataDockerCEInstall(out), eventData), nil
	}

	values := map[string]string{}
	paramBySlot := map[string]string{
		"container_image_url": strings.TrimSpace(p.containerImageURL),
		"access_token":        strings.TrimSpace(p.accessToken),
		"task_api_endpoint":   strings.TrimSpace(p.taskAPIEndpoint),
		"tenant_id":           strings.TrimSpace(p.tenantID),
		"workspace_id":        strings.TrimSpace(p.workspaceID),
		"task_id":             strings.TrimSpace(p.taskID),
		"trace_id":            strings.TrimSpace(p.traceID),
		"ssh_public_key":      strings.TrimSpace(p.sshPublicKey),
		"ssh_match_address":   strings.TrimSpace(p.sshMatchAddress),
		"installed_image_id":  strings.TrimSpace(p.installedImageID),
		"comment_id":          strings.TrimSpace(p.commentID),
		"container_name":      strings.TrimSpace(p.containerName),
	}
	for _, slot := range slotsNeeded {
		val := paramBySlot[slot]
		if val == "" {
			if userdataOptionalRuntimeSlots[slot] {
				// container_name 空值用稳定默认名，避免 docker 操作落到 "-"
				if slot == "container_name" {
					val = "task2app-container"
				} else {
					val = "-"
				}
			} else {
				return "", fmt.Errorf("UserData 包含运行时占位符（slot=%s），但未提供对应替换值", slot)
			}
		}
		if slot == "container_image_url" {
			val = normalizeUserdataContainerImageURLForDocker(val)
		}
		values[slot] = val
	}

	replaced := plain
	for _, entry := range userdataReplacementEntriesLongestFirst() {
		val, ok := values[entry.slot]
		if !ok {
			continue
		}
		if strings.Contains(replaced, entry.token) {
			replaced = strings.ReplaceAll(replaced, entry.token, val)
		}
	}

	if strings.Contains(replaced, userdataTaskCloudPrefixPlaceholder) {
		cp := buildUserdataTaskCloudPrefix(values["task_api_endpoint"], values["tenant_id"], values["workspace_id"], values["task_id"], p.commentID)
		if cp == "" {
			return "", fmt.Errorf("UserData 包含 __TASK2APP_TASK_CLOUD_PREFIX__，但未提供完整的 task_api_endpoint、tenant_id、workspace_id、task_id、comment_id 替换值")
		}
		replaced = strings.ReplaceAll(replaced, userdataTaskCloudPrefixPlaceholder, cp)
	}

	replaced = applyUserdataCloudPrefixRewrite(replaced, p)
	replaced = rewriteUserdataLocalhostVerifyHosts(replaced, strings.TrimSpace(p.taskAPIEndpoint))
	replaced = normalizeUserdataShellContainerNameRefs(replaced)
	replaced = hardenUserdataDockerCEInstall(replaced)
	return injectUserdataImageEnvs(replaced, eventData), nil
}

// hardenUserdataDockerCEInstall 将存量 UserData 中「仅 download.docker.com」的 Docker CE
// 安装段改写为阿里云/清华镜像优先 + 官方兜底，避免国内云主机 TLS reset 导致
// gpg/apt 失败、容器永不启动（与 taskAiProvider userDataScriptLinux.js 对齐）。
func hardenUserdataDockerCEInstall(plain string) string {
	if plain == "" {
		return plain
	}
	if strings.Contains(plain, "mirrors.aliyun.com/docker-ce") {
		return plain
	}
	out := plain
	for _, pair := range [][2]string{
		{userdataLegacyDockerUbuntuInstall, userdataHardenedDockerUbuntuInstall},
		{userdataLegacyDockerDebianInstall, userdataHardenedDockerDebianInstall},
		{userdataLegacyDockerCentosInstall, userdataHardenedDockerCentosInstall},
	} {
		if strings.Contains(out, pair[0]) {
			out = strings.Replace(out, pair[0], pair[1], 1)
		}
	}
	return out
}

const userdataLegacyDockerUbuntuInstall = `elif [ "$OS_FAMILY" = "ubuntu" ]; then
        mkdir -p /etc/apt/keyrings
        curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
        echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
        apt-get update
        apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin`

const userdataLegacyDockerDebianInstall = `elif [ "$OS_FAMILY" = "debian" ]; then
        mkdir -p /etc/apt/keyrings
        curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
        echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
        apt-get update
        apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin`

const userdataLegacyDockerCentosInstall = `if [ "$OS_FAMILY" = "centos" ]; then
        yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
        yum install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin`

const userdataHardenedDockerUbuntuInstall = `elif [ "$OS_FAMILY" = "ubuntu" ]; then
        mkdir -p /etc/apt/keyrings
        DOCKER_CE_OK=0
        for DOCKER_CE_MIRROR in \
          "https://mirrors.aliyun.com/docker-ce/linux/ubuntu" \
          "https://mirrors.tuna.tsinghua.edu.cn/docker-ce/linux/ubuntu" \
          "https://download.docker.com/linux/ubuntu"; do
            rm -f /etc/apt/keyrings/docker.gpg
            if curl -fsSL "${DOCKER_CE_MIRROR}/gpg" 2>/dev/null | gpg --dearmor -o /etc/apt/keyrings/docker.gpg \
              && [ -s /etc/apt/keyrings/docker.gpg ]; then
                echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] ${DOCKER_CE_MIRROR} $(lsb_release -cs) stable" \
                  | tee /etc/apt/sources.list.d/docker.list > /dev/null
                if apt-get update \
                  && apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin; then
                    DOCKER_CE_OK=1
                    break
                fi
            fi
        done
        if [ "$DOCKER_CE_OK" -ne 1 ]; then
            echo "错误: Docker CE 安装失败（GPG/仓库均不可达，含 download.docker.com）"
            exit 1
        fi`

const userdataHardenedDockerDebianInstall = `elif [ "$OS_FAMILY" = "debian" ]; then
        mkdir -p /etc/apt/keyrings
        DOCKER_CE_OK=0
        for DOCKER_CE_MIRROR in \
          "https://mirrors.aliyun.com/docker-ce/linux/debian" \
          "https://mirrors.tuna.tsinghua.edu.cn/docker-ce/linux/debian" \
          "https://download.docker.com/linux/debian"; do
            rm -f /etc/apt/keyrings/docker.gpg
            if curl -fsSL "${DOCKER_CE_MIRROR}/gpg" 2>/dev/null | gpg --dearmor -o /etc/apt/keyrings/docker.gpg \
              && [ -s /etc/apt/keyrings/docker.gpg ]; then
                echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] ${DOCKER_CE_MIRROR} $(lsb_release -cs) stable" \
                  | tee /etc/apt/sources.list.d/docker.list > /dev/null
                if apt-get update \
                  && apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin; then
                    DOCKER_CE_OK=1
                    break
                fi
            fi
        done
        if [ "$DOCKER_CE_OK" -ne 1 ]; then
            echo "错误: Docker CE 安装失败（GPG/仓库均不可达，含 download.docker.com）"
            exit 1
        fi`

const userdataHardenedDockerCentosInstall = `if [ "$OS_FAMILY" = "centos" ]; then
        DOCKER_CE_OK=0
        for _repo in \
          "https://mirrors.aliyun.com/docker-ce/linux/centos/docker-ce.repo" \
          "https://mirrors.tuna.tsinghua.edu.cn/docker-ce/linux/centos/docker-ce.repo" \
          "https://download.docker.com/linux/centos/docker-ce.repo"; do
            if yum-config-manager --add-repo "$_repo" 2>/dev/null; then
                if yum install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin; then
                    DOCKER_CE_OK=1
                    break
                fi
            fi
        done
        if [ "$DOCKER_CE_OK" -ne 1 ]; then
            echo "错误: Docker CE 安装失败（centos 镜像源均不可用）"
            exit 1
        fi`

// normalizeUserdataShellContainerNameRefs 将残留 ${containerName} 改写为 ${CONTAINER_NAME}。
func normalizeUserdataShellContainerNameRefs(plain string) string {
	if plain == "" || !strings.Contains(plain, "${containerName}") {
		return plain
	}
	return strings.ReplaceAll(plain, "${containerName}", "${CONTAINER_NAME}")
}

type userdataReplaceParams struct {
	containerImageURL string
	accessToken       string
	taskAPIEndpoint   string
	tenantID          string
	workspaceID       string
	taskID            string
	traceID           string
	sshPublicKey      string
	sshMatchAddress   string
	installedImageID  string
	commentID         string
	containerName     string
}

func userdataReplaceParamsFromEvent(eventData map[string]interface{}) userdataReplaceParams {
	// 与 taskEvents/internal/cloud/userdata/event_params.go ParamsFromEventData 对齐：
	// parent_comment_id 优先；container_name 缺省时由 task_id+comment_id 推导。
	commentID := firstNonEmpty(strField(eventData, "parent_comment_id"), strField(eventData, "comment_id"))
	taskID := strField(eventData, "task_id")
	containerName := firstNonEmpty(strField(eventData, "container_name"), strField(eventData, "mock_container_name"))
	if containerName == "" && taskID != "" && commentID != "" {
		containerName = buildCommentMockContainerName(taskID, commentID)
	}
	return userdataReplaceParams{
		containerImageURL: strField(eventData, "container_image_url"),
		accessToken:       strField(eventData, "userdata_access_token"),
		taskAPIEndpoint:   strField(eventData, "userdata_task_api_endpoint"),
		tenantID:          strField(eventData, "company_id"),
		workspaceID:       strField(eventData, "workspace_id"),
		taskID:            taskID,
		traceID:           strField(eventData, "trace_id"),
		sshPublicKey:      strField(eventData, "ssh_public_key"),
		sshMatchAddress:   strField(eventData, "ssh_match_address"),
		installedImageID:  firstNonEmpty(strField(eventData, "installed_image_id"), strField(eventData, "container_image_id")),
		commentID:         commentID,
		containerName:     containerName,
	}
}

type userdataTokenEntry struct {
	token string
	slot  string
	len   int
}

func userdataReplacementEntriesLongestFirst() []userdataTokenEntry {
	var entries []userdataTokenEntry
	for _, slot := range userdataRuntimeSlots {
		for _, t := range slot.tokens {
			entries = append(entries, userdataTokenEntry{token: t, slot: slot.name, len: len(t)})
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].len > entries[j].len })
	return entries
}

func matchedUserdataRuntimeSlots(plain string) []string {
	var out []string
	seen := map[string]bool{}
	for _, slot := range userdataRuntimeSlots {
		for _, t := range slot.tokens {
			if strings.Contains(plain, t) {
				if !seen[slot.name] {
					seen[slot.name] = true
					out = append(out, slot.name)
				}
				break
			}
		}
	}
	if strings.Contains(plain, userdataTaskCloudPrefixPlaceholder) {
		for _, req := range []string{"task_api_endpoint", "tenant_id", "workspace_id", "task_id"} {
			if !seen[req] {
				seen[req] = true
				out = append(out, req)
			}
		}
	}
	return out
}

// normalizeUserdataContainerImageURLForDocker 将 Docker Hub 页面 URL 转为可 pull 的镜像引用。
func normalizeUserdataContainerImageURLForDocker(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" || !strings.Contains(strings.ToLower(s), "hub.docker.com") || !strings.Contains(s, "://") {
		return s
	}
	u, err := url.Parse(s)
	if err != nil {
		return s
	}
	host := strings.ToLower(u.Hostname())
	if host != "hub.docker.com" && host != "www.hub.docker.com" {
		return s
	}
	segments := strings.FieldsFunc(strings.Trim(u.Path, "/"), func(r rune) bool { return r == '/' })
	if len(segments) == 0 {
		return s
	}
	if segments[0] == "_" {
		if len(segments) < 2 {
			return s
		}
		if len(segments) >= 4 && segments[len(segments)-2] == "tags" && segments[len(segments)-1] != "" {
			return segments[1] + ":" + segments[len(segments)-1]
		}
		return segments[1]
	}
	if segments[0] == "r" && len(segments) >= 3 {
		ns, repo := segments[1], segments[2]
		base := repo
		if ns != "library" {
			base = ns + "/" + repo
		}
		for i := 0; i < len(segments)-1; i++ {
			if segments[i] == "tags" && i+1 < len(segments) && segments[i+1] != "" {
				return base + ":" + segments[i+1]
			}
		}
		return base
	}
	return s
}
