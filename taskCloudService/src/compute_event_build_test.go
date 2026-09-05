package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestBuildUserdataExecutionWrapperScript(t *testing.T) {
	out := buildUserdataExecutionWrapperScript("#!/bin/bash\necho hi", "# verify\n")
	if !strings.Contains(out, "TASK2APP_INIT_EOF") {
		t.Fatal("missing heredoc wrapper")
	}
	if !strings.Contains(out, "echo hi") {
		t.Fatal("missing user script")
	}
}

func TestBuildHardwareEventPayloadAllowsEmptyDataDiskCategory(t *testing.T) {
	body := map[string]interface{}{
		"hardware_config": map[string]interface{}{
			"cpu_cores": 2, "memory_gb": 4, "storage_gb": 40,
			"data_disk_category": "",
		},
		"filter_options": map[string]interface{}{
			"data_disk_category": "",
			"spot_strategy":      "NoSpot",
		},
		"selected_instance": "ecs.c6.large",
	}
	hw := buildHardwareEventPayload(body)
	// 空字符串表示不要数据盘：不应被默认盘型覆盖；缺失或空均可
	if cat, ok := hw["data_disk_category"]; ok {
		if s := strings.TrimSpace(fmt.Sprint(cat)); s != "" {
			t.Fatalf("data_disk_category should stay empty for no-data-disk, got %q", s)
		}
	}
}

func TestBuildHardwareEventPayloadPrefersSelectedInstanceObjectAndCache(t *testing.T) {
	resetInstanceTypeSpecCacheForTest()
	t.Cleanup(resetInstanceTypeSpecCacheForTest)
	setCachedInstanceTypeSpec("ecs.e-c1m4.xlarge", instanceTypeSpec{cpuCores: 4, memoryGB: 16})

	body := map[string]interface{}{
		"hardware_config": map[string]interface{}{"cpu_cores": 1, "memory_gb": 1, "storage_gb": 40},
		"selected_instance": map[string]interface{}{
			"instance_type": "ecs.e-c1m4.xlarge",
			"cpu_cores":     4,
			"memory_gb":     16,
		},
	}
	hw := buildHardwareEventPayload(body)
	if hw["instance_type"] != "ecs.e-c1m4.xlarge" {
		t.Fatalf("instance_type=%v", hw["instance_type"])
	}
	if hw["cpu_cores"] != 4 || hw["memory_gb"] != 16 {
		t.Fatalf("cpu/mem=%v/%v want 4/16", hw["cpu_cores"], hw["memory_gb"])
	}
	if hw["storage_gb"] != 40 {
		t.Fatalf("storage=%v", hw["storage_gb"])
	}
}

func TestBuildHardwareEventPayloadCacheOverridesPlaceholderHardware(t *testing.T) {
	resetInstanceTypeSpecCacheForTest()
	t.Cleanup(resetInstanceTypeSpecCacheForTest)
	setCachedInstanceTypeSpec("ecs.e-c1m4.xlarge", instanceTypeSpec{cpuCores: 4, memoryGB: 16})

	body := map[string]interface{}{
		"hardware_config":   map[string]interface{}{"cpu_cores": 1, "memory_gb": 1, "storage_gb": 80, "instance_type": "ecs.e-c1m4.xlarge"},
		"selected_instance": "ecs.e-c1m4.xlarge",
	}
	hw := buildHardwareEventPayload(body)
	if hw["cpu_cores"] != 4 || hw["memory_gb"] != 16 {
		t.Fatalf("cpu/mem=%v/%v want 4/16 from cache", hw["cpu_cores"], hw["memory_gb"])
	}
	if hw["storage_gb"] != 80 {
		t.Fatalf("storage should stay request value, got %v", hw["storage_gb"])
	}
}

func TestResolveStartVmRuntimeSourcePrefersBody(t *testing.T) {
	if got := resolveStartVmRuntimeSource(map[string]interface{}{"runtime_source": "cloud_vm_auto_run"}); got != "cloud_vm_auto_run" {
		t.Fatalf("got=%s", got)
	}
	if got := resolveStartVmRuntimeSource(map[string]interface{}{"start_reason": "cloud_vm_manual"}); got != "cloud_vm_manual" {
		t.Fatalf("got=%s", got)
	}
	if got := resolveStartVmRuntimeSource(map[string]interface{}{}); got != "cloud_vm" {
		t.Fatalf("default=%s", got)
	}
}

func TestAlignHardwareWithAuthInstanceTypeUsesCache(t *testing.T) {
	resetInstanceTypeSpecCacheForTest()
	t.Cleanup(resetInstanceTypeSpecCacheForTest)
	setCachedInstanceTypeSpec("ecs.e-c1m4.xlarge", instanceTypeSpec{cpuCores: 4, memoryGB: 16})
	hw := map[string]interface{}{"cpu_cores": 1, "memory_gb": 1, "storage_gb": 40, "instance_type": "ecs.e-c1m4.xlarge"}
	alignHardwareWithAuthInstanceType(&cloudAuthRecord{PlatformType: "aliyun"}, "cn-hongkong", hw)
	if hw["cpu_cores"] != 4 || hw["memory_gb"] != 16 {
		t.Fatalf("hw=%v", hw)
	}
	if hw["storage_gb"] != 40 {
		t.Fatalf("storage=%v", hw["storage_gb"])
	}
}
