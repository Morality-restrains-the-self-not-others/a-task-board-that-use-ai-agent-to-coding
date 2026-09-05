package userdata

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
)

const (
	TaskAPIEndpointPlaceholder  = "__TASK2APP_TASK_API_ENDPOINT__"
	TaskCloudPrefixPlaceholder  = "__TASK2APP_TASK_CLOUD_PREFIX__"
	ContainerImagePlaceholder   = "__TASK2APP_CONTAINER_IMAGE__"
	AccessTokenPlaceholder      = "__TASK2APP_ACCESS_TOKEN__"
	InstalledImageIDPlaceholder = "__TASK2APP_INSTALLED_IMAGE_ID__"
	CommentIDPlaceholder        = "__TASK2APP_COMMENT_ID__"
	ContainerNamePlaceholder    = "__TASK2APP_CONTAINER_NAME__"
)

var containerImageTokens = []string{
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

type runtimeSlot struct {
	name   string
	tokens []string
}

var runtimeSlots = []runtimeSlot{
	{name: "container_image_url", tokens: containerImageTokens},
	{name: "access_token", tokens: []string{"__TASK2APP_ACCESS_TOKEN__"}},
	{name: "task_api_endpoint", tokens: []string{TaskAPIEndpointPlaceholder}},
	{name: "tenant_id", tokens: []string{"__TASK2APP_TENANT_ID__"}},
	{name: "workspace_id", tokens: []string{"__TASK2APP_WORKSPACE_ID__"}},
	{name: "task_id", tokens: []string{"__TASK2APP_TASK_ID__"}},
	{name: "trace_id", tokens: []string{"__TASK2APP_TRACE_ID__"}},
	{name: "ssh_public_key", tokens: []string{"__TASK2APP_SSH_PUBLIC_KEY__"}},
	{name: "ssh_match_address", tokens: []string{"__TASK2APP_SSH_MATCH_ADDRESS__"}},
	// Optional log-correlation slots: empty → "-" (never fail RunInstances).
	{name: "installed_image_id", tokens: []string{InstalledImageIDPlaceholder}},
	{name: "comment_id", tokens: []string{CommentIDPlaceholder}},
	// 仅替换 __TASK2APP_CONTAINER_NAME__ 占位符。
	// 禁止把 ${CONTAINER_NAME}/${containerName} 当作值注入 token：正式 Linux 模板在
	// report_progress / docker inspect 中把它们当 shell 变量；空值时注入 "-" 会变成
	// `[ "-" != "-" ]` 与 `docker inspect -`，启动日志不上报且状态卡在「启动中」。
	// 存量 camelCase 残留见 normalizeShellContainerNameRefs（改写为 ${CONTAINER_NAME}）。
	{name: "container_name", tokens: []string{ContainerNamePlaceholder}},
}

var optionalRuntimeSlots = map[string]bool{
	"installed_image_id": true,
	"comment_id":         true,
	"container_name":     true,
	"trace_id":           true, // 缺失时写 "-"，不阻断 RunInstances
}

type ReplaceParams struct {
	ContainerImageURL string
	AccessToken       string
	TaskAPIEndpoint   string
	TenantID          string
	WorkspaceID       string
	TaskID            string
	TraceID           string
	SSHPublicKey      string
	SSHMatchAddress   string
	InstalledImageID  string
	CommentID         string
	ContainerName     string
}

// ReplaceRuntimePlaceholders mirrors Django replace_userdata_runtime_placeholders +
// _rewrite_localhost_userdata_verify_hosts for RunInstances UserData.
func ReplaceRuntimePlaceholders(plain string, p ReplaceParams) (string, error) {
	if strings.TrimSpace(plain) == "" {
		return plain, nil
	}

	slotsNeeded := matchedRuntimeSlots(plain)
	if len(slotsNeeded) == 0 {
		out := applyCloudPrefixRewrite(plain, p)
		out = rewriteLocalhostVerifyHosts(out, strings.TrimSpace(p.TaskAPIEndpoint))
		return HardenDockerCEInstall(out), nil
	}

	values := map[string]string{}
	paramBySlot := map[string]string{
		"container_image_url": strings.TrimSpace(p.ContainerImageURL),
		"access_token":        strings.TrimSpace(p.AccessToken),
		"task_api_endpoint":   strings.TrimSpace(p.TaskAPIEndpoint),
		"tenant_id":           strings.TrimSpace(p.TenantID),
		"workspace_id":        strings.TrimSpace(p.WorkspaceID),
		"task_id":             strings.TrimSpace(p.TaskID),
		"trace_id":            strings.TrimSpace(p.TraceID),
		"ssh_public_key":      strings.TrimSpace(p.SSHPublicKey),
		"ssh_match_address":   strings.TrimSpace(p.SSHMatchAddress),
		"installed_image_id":  strings.TrimSpace(p.InstalledImageID),
		"comment_id":          strings.TrimSpace(p.CommentID),
		"container_name":      strings.TrimSpace(p.ContainerName),
	}
	for _, slot := range slotsNeeded {
		val := paramBySlot[slot]
		if val == "" {
			if optionalRuntimeSlots[slot] {
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
			val = NormalizeContainerImageURLForDocker(val)
		}
		values[slot] = val
	}

	replaced := plain
	for _, entry := range replacementEntriesLongestFirst() {
		val, ok := values[entry.slot]
		if !ok {
			continue
		}
		if strings.Contains(replaced, entry.token) {
			replaced = strings.ReplaceAll(replaced, entry.token, val)
		}
	}

	if strings.Contains(replaced, TaskCloudPrefixPlaceholder) {
		cp := buildTaskCloudPrefix(values["task_api_endpoint"], values["tenant_id"], values["workspace_id"], values["task_id"], p.CommentID)
		if cp == "" {
			return "", fmt.Errorf("UserData 包含 __TASK2APP_TASK_CLOUD_PREFIX__，但未提供完整的 task_api_endpoint、tenant_id、workspace_id、task_id、comment_id 替换值")
		}
		replaced = strings.ReplaceAll(replaced, TaskCloudPrefixPlaceholder, cp)
	}

	replaced = applyCloudPrefixRewrite(replaced, p)
	replaced = rewriteLocalhostVerifyHosts(replaced, strings.TrimSpace(p.TaskAPIEndpoint))
	// 存量模板健康检查残留 ${containerName}（JS 生成器 camelCase）→ 统一为已 export 的 shell 变量
	replaced = normalizeShellContainerNameRefs(replaced)
	replaced = HardenDockerCEInstall(replaced)
	return replaced, nil
}

// HardenDockerCEInstall 将存量 UserData 中「仅 download.docker.com」的 Docker CE
// 安装段改写为阿里云/清华镜像优先 + 官方兜底（与 taskAiProvider userDataScriptLinux.js /
// taskCloudService hardenUserdataDockerCEInstall 对齐）。
func HardenDockerCEInstall(plain string) string {
	if plain == "" {
		return plain
	}
	if strings.Contains(plain, "mirrors.aliyun.com/docker-ce") {
		return plain
	}
	out := plain
	for _, pair := range [][2]string{
		{legacyDockerUbuntuInstall, hardenedDockerUbuntuInstall},
		{legacyDockerDebianInstall, hardenedDockerDebianInstall},
		{legacyDockerCentosInstall, hardenedDockerCentosInstall},
	} {
		if strings.Contains(out, pair[0]) {
			out = strings.Replace(out, pair[0], pair[1], 1)
		}
	}
	return out
}

const legacyDockerUbuntuInstall = `elif [ "$OS_FAMILY" = "ubuntu" ]; then
        mkdir -p /etc/apt/keyrings
        curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
        echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
        apt-get update
        apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin`

const legacyDockerDebianInstall = `elif [ "$OS_FAMILY" = "debian" ]; then
        mkdir -p /etc/apt/keyrings
        curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
        echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
        apt-get update
        apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin`

const legacyDockerCentosInstall = `if [ "$OS_FAMILY" = "centos" ]; then
        yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
        yum install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin`

const hardenedDockerUbuntuInstall = `elif [ "$OS_FAMILY" = "ubuntu" ]; then
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

const hardenedDockerDebianInstall = `elif [ "$OS_FAMILY" = "debian" ]; then
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

const hardenedDockerCentosInstall = `if [ "$OS_FAMILY" = "centos" ]; then
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

// normalizeShellContainerNameRefs 将残留的 ${containerName} 改写为 ${CONTAINER_NAME}，
// 供 shell 在运行时展开（export CONTAINER_NAME=docker 名）。不注入平台路由名或 "-"。
func normalizeShellContainerNameRefs(plain string) string {
	if plain == "" || !strings.Contains(plain, "${containerName}") {
		return plain
	}
	return strings.ReplaceAll(plain, "${containerName}", "${CONTAINER_NAME}")
}

type tokenEntry struct {
	token string
	slot  string
	len   int
}

func replacementEntriesLongestFirst() []tokenEntry {
	var entries []tokenEntry
	for _, slot := range runtimeSlots {
		for _, t := range slot.tokens {
			entries = append(entries, tokenEntry{token: t, slot: slot.name, len: len(t)})
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].len > entries[j].len })
	return entries
}

func matchedRuntimeSlots(plain string) []string {
	var out []string
	seen := map[string]bool{}
	for _, slot := range runtimeSlots {
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
	if strings.Contains(plain, TaskCloudPrefixPlaceholder) {
		for _, req := range []string{"task_api_endpoint", "tenant_id", "workspace_id", "task_id"} {
			if !seen[req] {
				seen[req] = true
				out = append(out, req)
			}
		}
	}
	return out
}

// NormalizeContainerImageURLForDocker converts Docker Hub page URLs to pull references.
func NormalizeContainerImageURLForDocker(raw string) string {
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
