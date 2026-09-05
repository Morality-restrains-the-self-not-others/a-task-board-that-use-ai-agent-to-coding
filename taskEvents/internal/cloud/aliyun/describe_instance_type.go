package aliyun

import (
	"fmt"
	"strings"

	ecs "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/dara"
)

// InstanceTypeSpec is the authoritative CPU/memory for an ECS instance type.
type InstanceTypeSpec struct {
	CPUCores int
	MemoryGB int
}

// DescribeInstanceTypeSpec looks up CPU/memory for a single instance type ID.
func DescribeInstanceTypeSpec(accessKey, secretKey, region, instanceTypeID string) (InstanceTypeSpec, error) {
	instanceTypeID = strings.TrimSpace(instanceTypeID)
	if instanceTypeID == "" {
		return InstanceTypeSpec{}, fmt.Errorf("instance_type required")
	}
	region = strings.TrimSpace(region)
	if region == "" {
		region = "cn-hangzhou"
	}
	client, err := newECSClient(accessKey, secretKey, region)
	if err != nil {
		return InstanceTypeSpec{}, err
	}
	resp, err := client.DescribeInstanceTypes(&ecs.DescribeInstanceTypesRequest{
		InstanceTypes: []*string{dara.String(instanceTypeID)},
	})
	if err != nil {
		return InstanceTypeSpec{}, err
	}
	if resp == nil || resp.Body == nil || resp.Body.InstanceTypes == nil {
		return InstanceTypeSpec{}, fmt.Errorf("empty DescribeInstanceTypes response for %s", instanceTypeID)
	}
	for _, inst := range resp.Body.InstanceTypes.InstanceType {
		if inst == nil || inst.InstanceTypeId == nil {
			continue
		}
		if strings.TrimSpace(*inst.InstanceTypeId) != instanceTypeID {
			continue
		}
		cpu := 0
		if inst.CpuCoreCount != nil {
			cpu = int(*inst.CpuCoreCount)
		}
		mem := 0
		if inst.MemorySize != nil {
			mem = int(*inst.MemorySize + 0.5)
		}
		if cpu <= 0 {
			return InstanceTypeSpec{}, fmt.Errorf("invalid cpu for %s", instanceTypeID)
		}
		return InstanceTypeSpec{CPUCores: cpu, MemoryGB: mem}, nil
	}
	return InstanceTypeSpec{}, fmt.Errorf("instance type %s not found", instanceTypeID)
}

// ApplyInstanceTypeSpecToHardware overwrites cpu_cores/memory_gb from spec.
// storage_gb is left unchanged.
func ApplyInstanceTypeSpecToHardware(hw map[string]interface{}, spec InstanceTypeSpec) {
	if hw == nil || spec.CPUCores <= 0 {
		return
	}
	hw["cpu_cores"] = spec.CPUCores
	if spec.MemoryGB > 0 {
		hw["memory_gb"] = spec.MemoryGB
	}
}

// AlignHardwareCPUMemoryWithInstanceType overwrites cpu_cores/memory_gb from the
// instance type when available. storage_gb is left unchanged.
func AlignHardwareCPUMemoryWithInstanceType(accessKey, secretKey, region string, hw map[string]interface{}) {
	if hw == nil {
		return
	}
	it := strings.TrimSpace(fmt.Sprint(hw["instance_type"]))
	if it == "" || it == "<nil>" {
		return
	}
	spec, err := DescribeInstanceTypeSpec(accessKey, secretKey, region, it)
	if err != nil || spec.CPUCores <= 0 {
		return
	}
	ApplyInstanceTypeSpecToHardware(hw, spec)
}
