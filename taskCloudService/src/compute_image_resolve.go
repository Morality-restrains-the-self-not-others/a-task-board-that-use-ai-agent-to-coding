package main

import (
	"fmt"
	"strings"
)

func runtimeEnvString(env map[string]interface{}, key string) string {
	if env == nil {
		return ""
	}
	v, ok := env[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		return fmt.Sprintf("%.0f", t)
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

// resolveCloudServerImageID resolves marketplace runtime environment to cloud image id (Phase 3i Go cutover).
// desiredArch is optional (x86_64 / arm64); when set, runtime env architecture must match instance type.
func resolveCloudServerImageID(
	tenantID string,
	cloudServerImageID string,
	containerImageID string,
	platformType string,
	regionID string,
	desiredArch string,
) (imageID string, resolvedRegion string, errMsg string) {
	cloudServerImageID = strings.TrimSpace(cloudServerImageID)
	containerImageID = strings.TrimSpace(containerImageID)
	regionID = strings.TrimSpace(regionID)
	platformType = strings.TrimSpace(platformType)

	desiredArch = normalizeRuntimeArchitecture(desiredArch)

	if cloudServerImageID != "" {
		if containerImageID != "" && tenantID != "" {
			img, err := getInstalledImage(tenantID, containerImageID)
			if err == nil && img != nil && strings.TrimSpace(img.ExternalImageID) != "" {
				envs, fetchErr := fetchAIPublicImageRuntimeEnvironments(img.ExternalImageID)
				if fetchErr == nil && len(envs) > 0 {
					var candidates []map[string]interface{}
					for _, e := range envs {
						if runtimeEnvString(e, "platform_type") == platformType {
							candidates = append(candidates, e)
						}
					}
					if len(candidates) > 0 && regionID != "" {
						available := make([]string, 0, len(candidates))
						for _, e := range candidates {
							available = append(available, runtimeEnvString(e, "region"))
						}
						if !containsString(available, regionID) {
							return "", "", "直接传入的云服务器镜像在指定地域不可用"
						}
						regionCandidates := filterRuntimeEnvsByRegion(candidates, regionID)
						if picked, msg := pickRuntimeEnvByArchitecture(regionCandidates, desiredArch); msg != "" {
							return "", "", msg
						} else if picked != nil {
							marketImage := runtimeEnvString(picked, "image_id")
							if marketImage == cloudServerImageID {
								return cloudServerImageID, regionID, ""
							}
							return "", "", "传入的云服务器镜像 ID 与镜像市场配置不匹配"
						}
						return "", "", "未找到匹配地域与云服务器镜像 ID 的配置"
					}
				}
			}
		}
		if regionID != "" {
			return cloudServerImageID, regionID, ""
		}
		return cloudServerImageID, regionID, ""
	}

	if containerImageID == "" {
		return "", "", "缺少容器镜像或云服务器镜像"
	}
	if tenantID == "" {
		return "", "", "无法获取租户ID"
	}
	img, err := getInstalledImage(tenantID, containerImageID)
	if err != nil {
		return "", "", "查询已安装镜像失败"
	}
	if img == nil {
		return "", "", "未找到已安装镜像"
	}
	extID := strings.TrimSpace(img.ExternalImageID)
	if extID == "" {
		return "", "", "已安装镜像缺少 external_image_id"
	}
	envs, err := fetchAIPublicImageRuntimeEnvironments(extID)
	if err != nil {
		return "", "", "调用镜像市场 API 失败: " + err.Error()
	}
	if len(envs) == 0 {
		return "", "", "镜像市场返回空的运行环境列表"
	}
	var candidates []map[string]interface{}
	for _, e := range envs {
		if runtimeEnvString(e, "platform_type") == platformType {
			candidates = append(candidates, e)
		}
	}
	if len(candidates) == 0 {
		return "", "", "未找到匹配云平台的运行环境"
	}
	if regionID != "" {
		regionCandidates := filterRuntimeEnvsByRegion(candidates, regionID)
		if picked, msg := pickRuntimeEnvByArchitecture(regionCandidates, desiredArch); msg != "" {
			return "", "", msg
		} else if picked != nil {
			return runtimeEnvString(picked, "image_id"), regionID, ""
		}
		return "", "", "未找到匹配地域的运行环境"
	}
	if picked, msg := pickRuntimeEnvByArchitecture(candidates, desiredArch); msg != "" {
		return "", "", msg
	} else if picked != nil {
		return runtimeEnvString(picked, "image_id"), runtimeEnvString(picked, "region"), ""
	}
	return "", "", "未找到匹配架构的运行环境"
}

func normalizeRuntimeArchitecture(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "x86", "x86_64", "amd64":
		return "x86_64"
	case "arm", "arm64", "aarch64":
		return "arm64"
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}

// inferInstanceArchitecture maps Aliyun instance_type to CPU architecture (best-effort).
// SSOT heuristic also copied in taskProjectService/src/instance_architecture.go — keep tests in sync.
func inferInstanceArchitecture(instanceType string) string {
	t := strings.ToLower(strings.TrimSpace(instanceType))
	if t == "" {
		return ""
	}
	if strings.Contains(t, "arm") ||
		strings.Contains(t, "g8y") || strings.Contains(t, "c8y") || strings.Contains(t, "r8y") ||
		strings.Contains(t, "c6r") || strings.Contains(t, "g6r") || strings.Contains(t, "r6r") {
		return "arm64"
	}
	return "x86_64"
}

func filterRuntimeEnvsByRegion(candidates []map[string]interface{}, regionID string) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(candidates))
	for _, e := range candidates {
		if runtimeEnvString(e, "region") == regionID {
			out = append(out, e)
		}
	}
	return out
}

func pickRuntimeEnvByArchitecture(candidates []map[string]interface{}, desiredArch string) (map[string]interface{}, string) {
	if len(candidates) == 0 {
		return nil, "未找到匹配地域的运行环境"
	}
	if desiredArch == "" {
		return candidates[0], ""
	}
	var matched []map[string]interface{}
	for _, e := range candidates {
		if normalizeRuntimeArchitecture(runtimeEnvString(e, "architecture")) == desiredArch {
			matched = append(matched, e)
		}
	}
	if len(matched) == 1 {
		return matched[0], ""
	}
	if len(matched) > 1 {
		return matched[0], ""
	}
	avail := make([]string, 0, len(candidates))
	for _, e := range candidates {
		arch := normalizeRuntimeArchitecture(runtimeEnvString(e, "architecture"))
		if arch != "" {
			avail = append(avail, arch)
		}
	}
	if len(avail) == 0 {
		return candidates[0], ""
	}
	return nil, fmt.Sprintf("实例规格与宿主机镜像架构不匹配（需要 %s，可用: %s）", desiredArch, strings.Join(uniqueStrings(avail), ", "))
}

func uniqueStrings(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, s := range items {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func containsString(items []string, target string) bool {
	for _, s := range items {
		if s == target {
			return true
		}
	}
	return false
}
