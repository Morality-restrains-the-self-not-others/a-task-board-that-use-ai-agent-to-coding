package aliyun

import "testing"

func TestApplyInstanceTypeSpecToHardwareOverridesCPUMemoryKeepsStorage(t *testing.T) {
	hw := map[string]interface{}{
		"cpu_cores":     1,
		"memory_gb":     1,
		"storage_gb":    80,
		"instance_type": "ecs.e-c1m4.xlarge",
	}
	ApplyInstanceTypeSpecToHardware(hw, InstanceTypeSpec{CPUCores: 4, MemoryGB: 16})
	if hw["cpu_cores"] != 4 || hw["memory_gb"] != 16 {
		t.Fatalf("cpu/mem=%v/%v", hw["cpu_cores"], hw["memory_gb"])
	}
	if hw["storage_gb"] != 80 {
		t.Fatalf("storage=%v", hw["storage_gb"])
	}
}

func TestApplyInstanceTypeSpecToHardwareIgnoresInvalidSpec(t *testing.T) {
	hw := map[string]interface{}{"cpu_cores": 2, "memory_gb": 4}
	ApplyInstanceTypeSpecToHardware(hw, InstanceTypeSpec{})
	if hw["cpu_cores"] != 2 || hw["memory_gb"] != 4 {
		t.Fatalf("should keep original, got %v/%v", hw["cpu_cores"], hw["memory_gb"])
	}
}
