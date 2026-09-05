package autorunstartvm

import "testing"

func TestBuildAutoRunStartVmRequestFullResources(t *testing.T) {
	req, err := BuildAutoRunStartVmRequest("task_1", "img1", map[string]interface{}{
		"region":            "cn-hangzhou",
		"cloud_platform_id": "plat-1",
		"vpc_id":            "vpc-1",
		"vswitch_id":        "vsw-1",
		"security_group_id": "sg-1",
		"platform":          "aliyun",
		"hardware_config": map[string]interface{}{
			"cpu_cores":  "2",
			"memory_gb":  "4",
			"storage_gb": "40",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if req.APIPath != "start-vm" {
		t.Fatalf("apiPath=%s want start-vm", req.APIPath)
	}
	if req.Body["task_id"] != "task_1" || req.Body["container_image_id"] != "img1" {
		t.Fatalf("body=%v", req.Body)
	}
	if req.Body["runtime_source"] != RuntimeSourceCloudVMAutoRun {
		t.Fatalf("runtime_source=%v", req.Body["runtime_source"])
	}
}

func TestBuildAutoRunStartVmRequestMissingSG(t *testing.T) {
	req, err := BuildAutoRunStartVmRequest("task_1", "img1", map[string]interface{}{
		"region":            "cn-hangzhou",
		"cloud_platform_id": "plat-1",
		"vpc_id":            "vpc-1",
		"vswitch_id":        "vsw-1",
		"security_group_id": "",
		"hardware_config": map[string]interface{}{
			"cpu_cores":  "2",
			"memory_gb":  "4",
			"storage_gb": "40",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if req.APIPath != "start-vm-auto" {
		t.Fatalf("apiPath=%s want start-vm-auto", req.APIPath)
	}
}

func TestBuildAutoRunStartVmRequestPlaceholderHardware(t *testing.T) {
	req, err := BuildAutoRunStartVmRequest("t1", "img1", map[string]interface{}{
		"region": "cn-hangzhou", "cloud_platform_id": "p1",
		"vpc_id": "vpc-1", "vswitch_id": "vsw-1", "security_group_id": "sg-1",
		"hardware_config": map[string]interface{}{"cpu_cores": "1", "memory_gb": "1", "storage_gb": "40"},
		"filter_options":  map[string]interface{}{"cores": 8, "memory": 16},
	})
	if err != nil {
		t.Fatal(err)
	}
	if req.APIPath != "start-vm" {
		t.Fatalf("apiPath=%s", req.APIPath)
	}
	hw := req.Body["hardware_config"].(map[string]interface{})
	if hw["cpu_cores"] != float64(8) || hw["memory_gb"] != float64(16) {
		t.Fatalf("hw=%v", hw)
	}
}

func TestResolveProjectServerRunTemplateFromProjects(t *testing.T) {
	tpl := ResolveProjectServerRunTemplateFromProjects(
		[]map[string]interface{}{
			{"id": "a", "server_run_template": map[string]interface{}{}},
			{"id": "b", "server_run_template": map[string]interface{}{"region": "cn-beijing"}},
		},
		[]map[string]interface{}{{"project_id": "b"}},
	)
	if tpl == nil || tpl["region"] != "cn-beijing" {
		t.Fatalf("tpl=%v", tpl)
	}
}

func TestRunTemplateIsConfigured(t *testing.T) {
	if RunTemplateIsConfigured(nil) {
		t.Fatal("nil should be unconfigured")
	}
	if RunTemplateIsConfigured(map[string]interface{}{}) {
		t.Fatal("empty map should be unconfigured")
	}
	if RunTemplateIsConfigured(map[string]interface{}{"label": "", "hardware_config": map[string]interface{}{}}) {
		t.Fatal("blank fields should be unconfigured")
	}
	if !RunTemplateIsConfigured(map[string]interface{}{"region": "cn-hangzhou"}) {
		t.Fatal("region should count as configured")
	}
}
