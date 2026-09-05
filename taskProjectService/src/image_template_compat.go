package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

func parseJSONObject(raw string) map[string]interface{} {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]interface{}{}
	}
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil || out == nil {
		return map[string]interface{}{}
	}
	return out
}

func templateInstanceType(template map[string]interface{}) string {
	if template == nil {
		return ""
	}
	if v := strings.TrimSpace(fmt.Sprintf("%v", template["selected_instance"])); v != "" && v != "<nil>" {
		return v
	}
	hw, _ := template["hardware_config"].(map[string]interface{})
	if hw == nil {
		return ""
	}
	v := strings.TrimSpace(fmt.Sprintf("%v", hw["instance_type"]))
	if v == "<nil>" {
		return ""
	}
	return v
}

func runTemplateIsConfigured(template map[string]interface{}) bool {
	if template == nil || len(template) == 0 {
		return false
	}
	platform := strings.TrimSpace(fmt.Sprintf("%v", template["platform"]))
	if platform == "<nil>" {
		platform = ""
	}
	region := strings.TrimSpace(fmt.Sprintf("%v", template["region"]))
	if region == "<nil>" {
		region = ""
	}
	cloudID := strings.TrimSpace(fmt.Sprintf("%v", template["cloud_platform_id"]))
	if cloudID == "<nil>" {
		cloudID = ""
	}
	if platform == "" && cloudID == "" {
		return false
	}
	if region == "" {
		return false
	}
	return templateInstanceType(template) != ""
}

func instanceCompatibleWithImageArchitectures(instanceType string, imageArches []string) bool {
	arches := knownCPUArchitectures(imageArches)
	if len(arches) == 0 {
		return false
	}
	got := inferInstanceArchitecture(instanceType)
	if got == "" {
		return false
	}
	for _, a := range arches {
		if a == got {
			return true
		}
	}
	return false
}

func parseTargetArchitectures(v interface{}) []string {
	switch t := v.(type) {
	case []string:
		return normalizeCPUArchitectureList(t)
	case []interface{}:
		out := make([]string, 0, len(t))
		for _, item := range t {
			out = append(out, fmt.Sprintf("%v", item))
		}
		return normalizeCPUArchitectureList(out)
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return nil
		}
		var arr []interface{}
		if err := json.Unmarshal([]byte(s), &arr); err == nil {
			return parseTargetArchitectures(arr)
		}
		return normalizeCPUArchitectureList([]string{s})
	default:
		return nil
	}
}

// errImageTemplateIncompatible is a user-facing validation error (HTTP 400).
type errImageTemplateIncompatible string

func (e errImageTemplateIncompatible) Error() string { return string(e) }

const (
	errMsgImageTemplateIncomplete = "更换镜像须同时提交完整运行模版（含地域与实例规格）"
	imageArchLabelUnqueryable     = "无法查询"
	imageArchLabelMissing         = "不存在或已卸载"
)

func errImageArchUnresolvable(imageLabel, instanceType string) error {
	return errImageTemplateIncompatible(fmt.Sprintf(
		"无法解析已安装镜像的 CPU 架构（镜像要求: %s；实例系统支持: %s），拒绝保存以免规格不匹配",
		imageLabel, formatInstanceSupportedArch(instanceType),
	))
}

func errImageArchMismatch(imageArches []string, instanceType string) error {
	return errImageTemplateIncompatible(fmt.Sprintf(
		"镜像要求的 CPU 架构（%s）与实例系统支持的 CPU 架构（%s）不匹配，请在同一次保存中提交完整且匹配的 server_run_template",
		strings.Join(knownCPUArchitectures(imageArches), ", "), formatInstanceSupportedArch(instanceType),
	))
}

func stringifyArchMeta(v interface{}) string {
	if v == nil {
		return ""
	}
	s := strings.TrimSpace(fmt.Sprintf("%v", v))
	if s == "<nil>" {
		return ""
	}
	return s
}

func resolveImageCPUArchitectures(img map[string]interface{}) (canonical []string, raw []string) {
	if img == nil {
		return nil, nil
	}
	raw = parseTargetArchitectures(img["target_architectures"])
	canonical = knownCPUArchitectures(raw)
	if len(canonical) > 0 {
		return canonical, raw
	}
	return extractCPUArchitecturesFromText(
		stringifyArchMeta(img["version"]),
		stringifyArchMeta(img["name"]),
		stringifyArchMeta(img["image_url"]),
	), raw
}

func coerceTemplateMap(v interface{}) map[string]interface{} {
	if v == nil {
		return map[string]interface{}{}
	}
	if m, ok := v.(map[string]interface{}); ok {
		if m == nil {
			return map[string]interface{}{}
		}
		return m
	}
	b, err := json.Marshal(v)
	if err != nil {
		return map[string]interface{}{}
	}
	return parseJSONObject(string(b))
}

// resolveNextRunTemplate prefers a complete body draft; incomplete drafts must not
// override a complete stored template (FE hardware panel often sends region without instance).
func resolveNextRunTemplate(tmplRaw string, body map[string]interface{}) map[string]interface{} {
	stored := parseJSONObject(tmplRaw)
	v, ok := body["server_run_template"]
	if !ok {
		return stored
	}
	bodyTpl := coerceTemplateMap(v)
	if runTemplateIsConfigured(bodyTpl) {
		return bodyTpl
	}
	if runTemplateIsConfigured(stored) {
		delete(body, "server_run_template")
		return stored
	}
	return bodyTpl
}

func validateProjectUpdateImageTemplate(tenantID, currentImage, currentImageName, tmplRaw string, body map[string]interface{}) (healedImageID string, err error) {
	_, imageInBody := body["container_image_id"]
	nextImage := strings.TrimSpace(currentImage)
	if imageInBody {
		nextImage = strings.TrimSpace(fmt.Sprintf("%v", body["container_image_id"]))
		if nextImage == "<nil>" {
			nextImage = ""
		}
	}
	nextTemplate := resolveNextRunTemplate(tmplRaw, body)
	_, templateInBody := body["server_run_template"]
	if !imageInBody && !templateInBody {
		return "", nil
	}
	imageChanged := imageInBody && nextImage != strings.TrimSpace(currentImage)
	if nextImage == "" {
		return "", nil
	}
	needLookup := imageChanged || runTemplateIsConfigured(nextTemplate)
	lookupOK := true
	unresolvableLabel := imageArchLabelUnqueryable
	var arches []string
	if needLookup {
		imageName := strings.TrimSpace(currentImageName)
		if v := strField(body, "container_image"); v != "" {
			imageName = v
		}
		img, lookupErr := lookupInstalledImage(tenantID, nextImage, imageName)
		if lookupErr != nil {
			lookupOK = false
		} else if img == nil {
			lookupOK = false
			unresolvableLabel = imageArchLabelMissing
		} else {
			canonical, raw := resolveImageCPUArchitectures(img)
			arches = canonical
			if len(arches) == 0 {
				// Keep unrecognized declared tokens so 400 names the image-required side.
				arches = raw
			}
			resolvedID := strings.TrimSpace(fmt.Sprintf("%v", img["id"]))
			if resolvedID != "" && resolvedID != "<nil>" && resolvedID != nextImage {
				healedImageID = resolvedID
			}
		}
	}
	return healedImageID, validateProjectImageTemplatePair(nextImage, arches, lookupOK, unresolvableLabel, nextTemplate, imageChanged)
}

func validateProjectImageTemplatePair(nextImageID string, imageArches []string, lookupOK bool, unresolvableLabel string, nextTemplate map[string]interface{}, imageChanged bool) error {
	nextImageID = strings.TrimSpace(nextImageID)
	if nextImageID == "" {
		return nil
	}
	if !imageChanged && !runTemplateIsConfigured(nextTemplate) {
		return nil
	}
	if imageChanged && !runTemplateIsConfigured(nextTemplate) {
		return errImageTemplateIncompatible(errMsgImageTemplateIncomplete)
	}
	if !runTemplateIsConfigured(nextTemplate) {
		return nil
	}
	inst := templateInstanceType(nextTemplate)
	if !lookupOK {
		label := strings.TrimSpace(unresolvableLabel)
		if label == "" {
			label = imageArchLabelUnqueryable
		}
		return errImageArchUnresolvable(label, inst)
	}
	canonical := knownCPUArchitectures(imageArches)
	if len(canonical) == 0 {
		return errImageArchUnresolvable(formatImageRequiredArch(nil, imageArches), inst)
	}
	if !instanceCompatibleWithImageArchitectures(inst, canonical) {
		return errImageArchMismatch(canonical, inst)
	}
	return nil
}
