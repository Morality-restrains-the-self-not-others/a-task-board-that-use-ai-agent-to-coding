package main

import "testing"

func TestOptionalServerRunTemplateComplete(t *testing.T) {
	tpl := map[string]interface{}{
		"cloud_platform_id": "plat-1",
		"region":            "cn-hangzhou",
		"zone_id":           "b",
	}
	got := optionalServerRunTemplate(map[string]interface{}{
		"content":             "@镜像",
		"server_run_template": tpl,
	})
	if got == nil {
		t.Fatal("expected complete template")
	}
	if got["region"] != "cn-hangzhou" || got["cloud_platform_id"] != "plat-1" {
		t.Fatalf("got %#v", got)
	}
}

func TestOptionalServerRunTemplateMissingOrIncomplete(t *testing.T) {
	if optionalServerRunTemplate(nil) != nil {
		t.Fatal("nil body")
	}
	if optionalServerRunTemplate(map[string]interface{}{"content": "hi"}) != nil {
		t.Fatal("absent field")
	}
	if optionalServerRunTemplate(map[string]interface{}{
		"server_run_template": map[string]interface{}{"cloud_platform_id": "plat-1"},
	}) != nil {
		t.Fatal("missing region must be ignored")
	}
	if optionalServerRunTemplate(map[string]interface{}{
		"server_run_template": "not-an-object",
	}) != nil {
		t.Fatal("non-object must be ignored")
	}
}
