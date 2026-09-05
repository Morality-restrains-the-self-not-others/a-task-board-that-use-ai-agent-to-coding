package main

import (
	"strings"
	"testing"
)

func TestInjectImageSkillEnvAddsExportAndDockerFlag(t *testing.T) {
	in := "#!/bin/bash\nset -e\ndocker run -e ACCESS_TOKEN=x example.com/img:v1\n"
	out := injectImageSkillEnv(in, "k8s-debug")
	if !strings.Contains(out, "export IMAGE_SKILL='k8s-debug'") {
		t.Fatalf("missing export: %q", out)
	}
	if !strings.Contains(out, "-e IMAGE_SKILL='k8s-debug'") {
		t.Fatalf("missing docker -e: %q", out)
	}
	if injectImageSkillEnv(out, "k8s-debug") != out {
		t.Fatal("must be idempotent when IMAGE_SKILL already present")
	}
}

func TestInjectImageSkillEnvIgnoresInvalid(t *testing.T) {
	in := "docker run img\n"
	if got := injectImageSkillEnv(in, "Bad_Name"); got != in {
		t.Fatalf("got=%q", got)
	}
	if got := injectImageSkillEnv(in, ""); got != in {
		t.Fatalf("empty got=%q", got)
	}
}

func TestResolveStartVmImageSkillPrefersExplicit(t *testing.T) {
	if got := resolveStartVmImageSkill("t1", "img", "k8s-debug"); got != "k8s-debug" {
		t.Fatalf("got=%q", got)
	}
	if got := resolveStartVmImageSkill("t1", "img", "Bad_Name"); got != "" {
		t.Fatalf("invalid explicit should not pass, got=%q", got)
	}
}

func TestNormalizeInboundSkillVersion(t *testing.T) {
	cases := []struct{ in, want string }{
		{"v1", "1"},
		{"V2", "2"},
		{" 1.2.0 ", "1.2.0"},
		{"1", "1"},
		{"", ""},
		{"v", ""},
	}
	for _, c := range cases {
		if got := normalizeInboundSkillVersion(c.in); got != c.want {
			t.Fatalf("normalizeInboundSkillVersion(%q)=%q want %q", c.in, got, c.want)
		}
	}
}

func TestInjectSaasInboundSkillVersionEnvAddsExportAndDockerFlag(t *testing.T) {
	in := "#!/bin/bash\nset -e\ndocker run -e ACCESS_TOKEN=x example.com/img:v1\n"
	out := injectSaasInboundSkillVersionEnv(in, "1")
	if !strings.Contains(out, "export SAAS_INBOUND_SKILL_VERSION='1'") {
		t.Fatalf("missing export: %q", out)
	}
	if !strings.Contains(out, "-e SAAS_INBOUND_SKILL_VERSION='1'") {
		t.Fatalf("missing docker -e: %q", out)
	}
	if injectSaasInboundSkillVersionEnv(out, "1") != out {
		t.Fatal("must be idempotent when SAAS_INBOUND_SKILL_VERSION already present")
	}
}

func TestInjectSaasInboundSkillVersionEnvIgnoresInvalid(t *testing.T) {
	in := "docker run img\n"
	if got := injectSaasInboundSkillVersionEnv(in, "v1; rm -rf /"); got != in {
		t.Fatalf("shell-injection version must be rejected, got=%q", got)
	}
	if got := injectSaasInboundSkillVersionEnv(in, ""); got != in {
		t.Fatalf("empty got=%q", got)
	}
}

func TestInjectUserdataImageEnvsInjectsBoth(t *testing.T) {
	in := "#!/bin/bash\nset -e\ndocker run example.com/img:v1\n"
	eventData := map[string]interface{}{
		"image_skill":                "k8s-debug",
		"saas_inbound_skill_version": "1",
	}
	out := injectUserdataImageEnvs(in, eventData)
	if !strings.Contains(out, "export IMAGE_SKILL='k8s-debug'") {
		t.Fatalf("missing IMAGE_SKILL export: %q", out)
	}
	if !strings.Contains(out, "export SAAS_INBOUND_SKILL_VERSION='1'") {
		t.Fatalf("missing SAAS_INBOUND_SKILL_VERSION export: %q", out)
	}
	if !strings.Contains(out, "-e IMAGE_SKILL='k8s-debug'") || !strings.Contains(out, "-e SAAS_INBOUND_SKILL_VERSION='1'") {
		t.Fatalf("missing docker -e flags: %q", out)
	}
}

func TestResolveStartVmSaasInboundSkillVersionReadsInstalledImage(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_tenant_installed_images
		(id,tenant_id,external_image_id,name,image_url,target_architectures,is_dev_mode,saas_inbound_skill_version)
		VALUES('img-sk','t1','ext-sk','skill-img','registry.example/skill','[]',0,'v1')`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if got := resolveStartVmSaasInboundSkillVersion("t1", "img-sk"); got != "1" {
		t.Fatalf("resolveStartVmSaasInboundSkillVersion='%s' want '1'", got)
	}
	if got := resolveStartVmSaasInboundSkillVersion("t1", "missing-img"); got != "" {
		t.Fatalf("missing image should return empty, got=%q", got)
	}
}
