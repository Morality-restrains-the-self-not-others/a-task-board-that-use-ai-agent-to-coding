package main

import (
	"strings"
	"testing"
)

func TestInferInstanceArchitectureProjectCopy(t *testing.T) {
	if inferInstanceArchitecture("ecs.r6.xlarge") != "x86_64" {
		t.Fatalf("r6 should be x86_64")
	}
	if inferInstanceArchitecture("ecs.r6r.xlarge") != "arm64" {
		t.Fatalf("r6r should be arm64")
	}
}

func TestValidateProjectImageTemplatePair(t *testing.T) {
	x86Tmpl := map[string]interface{}{
		"platform":          "aliyun",
		"region":            "cn-hangzhou",
		"cloud_platform_id": "p1",
		"selected_instance": "ecs.g7.xlarge",
		"hardware_config":   map[string]interface{}{"instance_type": "ecs.g7.xlarge"},
	}
	armTmpl := map[string]interface{}{
		"platform":          "aliyun",
		"region":            "cn-hangzhou",
		"cloud_platform_id": "p1",
		"selected_instance": "ecs.g8y.xlarge",
	}

	if err := validateProjectImageTemplatePair("img-arm", []string{"arm64"}, true, "", x86Tmpl, true); err == nil {
		t.Fatal("expected mismatch")
	}
	if err := validateProjectImageTemplatePair("img-arm", []string{"arm64"}, true, "", armTmpl, true); err != nil {
		t.Fatalf("arm match: %v", err)
	}
	if err := validateProjectImageTemplatePair("img-x86", []string{"x86_64"}, true, "", x86Tmpl, true); err != nil {
		t.Fatalf("x86 match: %v", err)
	}
	if err := validateProjectImageTemplatePair("img-arm", []string{"arm64"}, true, "", map[string]interface{}{}, true); err == nil {
		t.Fatal("expected incomplete template")
	}
	if err := validateProjectImageTemplatePair("", []string{"arm64"}, true, "", x86Tmpl, true); err != nil {
		t.Fatalf("clear image should skip: %v", err)
	}
	if err := validateProjectImageTemplatePair("img-arm", nil, false, "", x86Tmpl, true); err == nil {
		t.Fatal("lookup fail should reject")
	}
	if err := validateProjectImageTemplatePair("img-x86", []string{"x86_64"}, true, "", x86Tmpl, false); err != nil {
		t.Fatalf("same image keep template: %v", err)
	}

	unknownErr := validateProjectImageTemplatePair("img-empty", nil, true, "", x86Tmpl, true)
	if unknownErr == nil {
		t.Fatal("empty arches should reject")
	}
	msg := unknownErr.Error()
	if !strings.Contains(msg, "镜像要求: 未声明") || !strings.Contains(msg, "实例系统支持: x86_64（实例规格 ecs.g7.xlarge）") {
		t.Fatalf("unknown error should name both arches, got %q", msg)
	}

	unrecognizedErr := validateProjectImageTemplatePair("img-empty", []string{"unknown"}, true, "", x86Tmpl, true)
	if unrecognizedErr == nil {
		t.Fatal("unrecognized declared arch should reject")
	}
	unrecognizedMsg := unrecognizedErr.Error()
	if !strings.Contains(unrecognizedMsg, "镜像要求: unknown（无法识别为 x86_64/arm64）") ||
		!strings.Contains(unrecognizedMsg, "实例系统支持: x86_64（实例规格 ecs.g7.xlarge）") {
		t.Fatalf("unrecognized error should name both arches, got %q", unrecognizedMsg)
	}

	mismatchErr := validateProjectImageTemplatePair("img-arm", []string{"arm64"}, true, "", x86Tmpl, true)
	if mismatchErr == nil {
		t.Fatal("expected mismatch")
	}
	mismatchMsg := mismatchErr.Error()
	if !strings.Contains(mismatchMsg, "镜像要求的 CPU 架构（arm64）") ||
		!strings.Contains(mismatchMsg, "实例系统支持的 CPU 架构（x86_64（实例规格 ecs.g7.xlarge））") {
		t.Fatalf("mismatch error should name both arches, got %q", mismatchMsg)
	}

	lookupErr := validateProjectImageTemplatePair("img-arm", nil, false, "", x86Tmpl, true)
	if lookupErr == nil {
		t.Fatal("lookup fail should reject")
	}
	if !strings.Contains(lookupErr.Error(), "镜像要求: 无法查询") ||
		!strings.Contains(lookupErr.Error(), "实例系统支持: x86_64（实例规格 ecs.g7.xlarge）") {
		t.Fatalf("lookup error should name both sides, got %q", lookupErr.Error())
	}

	missingErr := validateProjectImageTemplatePair("img-stale", nil, false, imageArchLabelMissing, x86Tmpl, true)
	if missingErr == nil {
		t.Fatal("missing image should reject")
	}
	if !strings.Contains(missingErr.Error(), "镜像要求: 不存在或已卸载") {
		t.Fatalf("missing image should name uninstall, got %q", missingErr.Error())
	}
}

func TestResolveImageCPUArchitecturesFromVersionWhenDeclaredEmpty(t *testing.T) {
	canonical, raw := resolveImageCPUArchitectures(map[string]interface{}{
		"target_architectures": []string{},
		"version":              "private_x86_64-latest",
		"name":                 "trae-agent",
	})
	if len(raw) != 0 {
		t.Fatalf("raw should stay empty, got %v", raw)
	}
	if len(canonical) != 1 || canonical[0] != "x86_64" {
		t.Fatalf("version hint should yield x86_64, got %v", canonical)
	}
	filtered, _ := resolveImageCPUArchitectures(map[string]interface{}{
		"target_architectures": []string{"x86_64", "unknown"},
	})
	if len(filtered) != 1 || filtered[0] != "x86_64" {
		t.Fatalf("unknown token should drop, got %v", filtered)
	}
}

func TestRunTemplateIsConfigured(t *testing.T) {
	if runTemplateIsConfigured(map[string]interface{}{}) {
		t.Fatal("empty")
	}
	if runTemplateIsConfigured(map[string]interface{}{"default_auto_run": true}) {
		t.Fatal("auto_run only")
	}
	if !runTemplateIsConfigured(map[string]interface{}{
		"platform": "aliyun", "region": "cn-hangzhou", "selected_instance": "ecs.g7.xlarge",
	}) {
		t.Fatal("should be configured")
	}
}

func TestResolveNextRunTemplateIgnoresIncompleteBodyDraft(t *testing.T) {
	storedJSON := `{"platform":"aliyun","region":"cn-hangzhou","cloud_platform_id":"p1","selected_instance":"ecs.g7.xlarge"}`
	body := map[string]interface{}{
		"server_run_template": map[string]interface{}{
			"platform": "aliyun",
			"region":   "cn-hangzhou",
			"label":    "半成品无实例",
		},
	}
	got := resolveNextRunTemplate(storedJSON, body)
	if !runTemplateIsConfigured(got) {
		t.Fatalf("expected stored complete template, got %#v", got)
	}
	if _, ok := body["server_run_template"]; ok {
		t.Fatal("incomplete body draft must be dropped so persistence keeps stored template")
	}
	if got["selected_instance"] != "ecs.g7.xlarge" {
		t.Fatalf("selected_instance: %v", got["selected_instance"])
	}
}

func TestResolveNextRunTemplateKeepsIncompleteWhenStoredIncomplete(t *testing.T) {
	body := map[string]interface{}{
		"server_run_template": map[string]interface{}{
			"platform": "aliyun",
			"region":   "cn-hangzhou",
		},
	}
	got := resolveNextRunTemplate(`{}`, body)
	if runTemplateIsConfigured(got) {
		t.Fatal("should stay incomplete")
	}
	if _, ok := body["server_run_template"]; !ok {
		t.Fatal("body key should remain when stored is incomplete")
	}
}
