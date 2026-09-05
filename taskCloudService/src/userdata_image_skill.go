package main

import (
	"regexp"
	"strings"
)

var dockerRunPrefixRe = regexp.MustCompile(`(?m)^(.*\bdocker\s+run\b)(\s+)`)

// inboundSkillVersionRE 仅允许字母数字与 ._-（契约版本形如 1 / v1 / 1.2.0），
// 防止 UserData 注入时把不可信目录字段写进 shell。
var inboundSkillVersionRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

func sanitizeImageSkillName(name string) string {
	name = strings.TrimSpace(name)
	if !imageSkillNameRE.MatchString(name) {
		return ""
	}
	return name
}

// normalizeInboundSkillVersion 归一化 SaaS inbound 契约版本：去首字母 v/V 并去空格
// （对齐 taskAiProvider domain.NormalizeSkillVersion 语义：镜像声明 v1 → env 为 1）。
func normalizeInboundSkillVersion(v string) string {
	s := strings.TrimSpace(v)
	s = strings.TrimPrefix(s, "v")
	s = strings.TrimPrefix(s, "V")
	return strings.TrimSpace(s)
}

func sanitizeInboundSkillVersion(v string) string {
	v = strings.TrimSpace(v)
	if !inboundSkillVersionRE.MatchString(v) {
		return ""
	}
	return v
}

// injectImageSkillEnv 把所选（或默认）技能写入 UserData：export + docker run -e。
// 模板无需预埋占位符；已有 IMAGE_SKILL 则不重复注入。
func injectImageSkillEnv(script, skill string) string {
	skill = sanitizeImageSkillName(skill)
	if skill == "" || strings.TrimSpace(script) == "" {
		return script
	}
	needExport := !strings.Contains(script, "export IMAGE_SKILL=")
	needDocker := strings.Contains(script, "docker run") &&
		!strings.Contains(script, "-e IMAGE_SKILL=") &&
		!strings.Contains(script, "--env IMAGE_SKILL=")
	if !needExport && !needDocker {
		return script
	}
	out := script
	if needExport {
		out = insertImageSkillExport(out, "export IMAGE_SKILL='"+skill+"'\n")
	}
	if needDocker {
		if dockerRunPrefixRe.MatchString(out) {
			out = dockerRunPrefixRe.ReplaceAllString(out, "${1} -e IMAGE_SKILL='"+skill+"'${2}")
		} else {
			out = strings.Replace(out, "docker run", "docker run -e IMAGE_SKILL='"+skill+"'", 1)
		}
	}
	return out
}

// injectSaasInboundSkillVersionEnv 把镜像声明的 SaaS inbound 契约版本写入 UserData：
// export + docker run -e（复用 IMAGE_SKILL 的 export 插入位）。模板无需预埋占位符；
// 已有 SAAS_INBOUND_SKILL_VERSION 则不重复注入。
func injectSaasInboundSkillVersionEnv(script, version string) string {
	version = sanitizeInboundSkillVersion(version)
	if version == "" || strings.TrimSpace(script) == "" {
		return script
	}
	needExport := !strings.Contains(script, "export SAAS_INBOUND_SKILL_VERSION=")
	needDocker := strings.Contains(script, "docker run") &&
		!strings.Contains(script, "-e SAAS_INBOUND_SKILL_VERSION=") &&
		!strings.Contains(script, "--env SAAS_INBOUND_SKILL_VERSION=")
	if !needExport && !needDocker {
		return script
	}
	out := script
	if needExport {
		out = insertImageSkillExport(out, "export SAAS_INBOUND_SKILL_VERSION='"+version+"'\n")
	}
	if needDocker {
		if dockerRunPrefixRe.MatchString(out) {
			out = dockerRunPrefixRe.ReplaceAllString(out, "${1} -e SAAS_INBOUND_SKILL_VERSION='"+version+"'${2}")
		} else {
			out = strings.Replace(out, "docker run", "docker run -e SAAS_INBOUND_SKILL_VERSION='"+version+"'", 1)
		}
	}
	return out
}

// injectUserdataImageEnvs 统一注入镜像相关环境变量：IMAGE_SKILL（所选/默认技能）与
// SAAS_INBOUND_SKILL_VERSION（ADR-0024 契约版本）。空值不注入。
func injectUserdataImageEnvs(script string, eventData map[string]interface{}) string {
	out := injectImageSkillEnv(script, strField(eventData, "image_skill"))
	return injectSaasInboundSkillVersionEnv(out, strField(eventData, "saas_inbound_skill_version"))
}

// resolveStartVmSaasInboundSkillVersion 从已安装镜像记录读取 SaaS inbound 契约版本
// （安装时已从公开目录/厂商镜像字段归一化拷贝）；读不到返回空（不注入）。
func resolveStartVmSaasInboundSkillVersion(tenantID, installedImageID string) string {
	tenantID = strings.TrimSpace(tenantID)
	installedImageID = strings.TrimSpace(installedImageID)
	if tenantID == "" || installedImageID == "" || db == nil {
		return ""
	}
	img, err := getInstalledImage(tenantID, installedImageID)
	if err != nil || img == nil {
		return ""
	}
	return normalizeInboundSkillVersion(img.SaasInboundSkillVersion)
}

func insertImageSkillExport(script, exportLine string) string {
	lines := strings.SplitN(script, "\n", 2)
	if strings.HasPrefix(strings.TrimSpace(lines[0]), "#!") {
		rest := ""
		if len(lines) == 2 {
			rest = lines[1]
		}
		if rest == "" {
			return lines[0] + "\n" + exportLine
		}
		return lines[0] + "\n" + exportLine + rest
	}
	return exportLine + script
}

func resolveStartVmImageSkill(tenantID, installedImageID, explicit string) string {
	if s := sanitizeImageSkillName(explicit); s != "" {
		return s
	}
	tenantID = strings.TrimSpace(tenantID)
	installedImageID = strings.TrimSpace(installedImageID)
	if tenantID == "" || installedImageID == "" || db == nil {
		return ""
	}
	img, err := getInstalledImage(tenantID, installedImageID)
	if err != nil || img == nil {
		return ""
	}
	return sanitizeImageSkillName(decodeImageSkillsJSON(img.ImageSkillsJSON).DefaultName())
}
