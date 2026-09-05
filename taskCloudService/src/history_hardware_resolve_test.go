package main

import "testing"

func TestResolveHistoryCPUMemoryUsesInstanceTypeCache(t *testing.T) {
	resetInstanceTypeSpecCacheForTest()
	t.Cleanup(resetInstanceTypeSpecCacheForTest)
	setCachedInstanceTypeSpec("ecs.e-c1m4.xlarge", instanceTypeSpec{cpuCores: 4, memoryGB: 16})

	h := &CloudServerConfigHistory{
		InstanceTypeID: "ecs.e-c1m4.xlarge",
		CpuCores:       1,
		MemoryGB:       1,
		StorageGB:      40,
	}
	cpu, mem := resolveHistoryCPUMemory(h)
	if cpu != 4 || mem != 16 {
		t.Fatalf("got %d/%d want 4/16", cpu, mem)
	}
	out := cloudServerConfigHistoryToJSON(h)
	hw := out["hardware_config"].(map[string]interface{})
	if hw["cpu_cores"] != 4 || hw["memory_gb"] != 16 || hw["storage_gb"] != 40 {
		t.Fatalf("json hw=%v", hw)
	}
}

func TestResolveHistoryCPUMemoryKeepsStoredWhenNoCache(t *testing.T) {
	resetInstanceTypeSpecCacheForTest()
	t.Cleanup(resetInstanceTypeSpecCacheForTest)
	h := &CloudServerConfigHistory{
		InstanceTypeID: "ecs.unknown.xlarge",
		CpuCores:       2,
		MemoryGB:       8,
	}
	cpu, mem := resolveHistoryCPUMemory(h)
	if cpu != 2 || mem != 8 {
		t.Fatalf("got %d/%d", cpu, mem)
	}
}
