package aliyun

import (
	"fmt"
	"strings"

	ecs "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/dara"
)

// StopVMInput is the minimal DeleteInstance payload for CLOUD_SERVER_STOPPED.
type StopVMInput struct {
	RegionID   string
	InstanceID string
}

// StopVMResult holds release outcome metadata.
type StopVMResult struct {
	InstanceID string
	RegionID   string
	RequestIDs []string
}

// StopVM releases an ECS instance (DeleteInstance + best-effort disk cleanup).
func StopVM(accessKey, secretKey string, in StopVMInput) (StopVMResult, error) {
	regionID := strings.TrimSpace(in.RegionID)
	instanceID := strings.TrimSpace(in.InstanceID)
	if instanceID == "" {
		return StopVMResult{}, fmt.Errorf("未提供实例ID")
	}
	if regionID == "" {
		regionID = "cn-hongkong"
	}

	client, err := newECSClient(accessKey, secretKey, regionID)
	if err != nil {
		return StopVMResult{}, err
	}

	requestIDs := []string{}
	diskIDs, listErr := listDiskIDsForInstance(client, regionID, instanceID)
	if listErr != nil {
		_ = listErr
	}

	force := true
	forceStop := true
	delResp, err := client.DeleteInstance(&ecs.DeleteInstanceRequest{
		InstanceId: dara.String(instanceID),
		Force:      &force,
		ForceStop:  &forceStop,
	})
	if err != nil {
		return StopVMResult{}, fmt.Errorf("停止虚拟机失败: %w", err)
	}
	if rid := extractStopRequestID(delResp); rid != "" {
		requestIDs = append(requestIDs, rid)
	}
	for _, diskID := range diskIDs {
		if diskID == "" {
			continue
		}
		_ = deleteDiskBestEffort(client, diskID, &requestIDs)
	}

	return StopVMResult{
		InstanceID: instanceID,
		RegionID:   regionID,
		RequestIDs: requestIDs,
	}, nil
}

func listDiskIDsForInstance(client *ecs.Client, regionID, instanceID string) ([]string, error) {
	out := []string{}
	page := int32(1)
	pageSize := int32(50)
	for {
		resp, err := client.DescribeDisks(&ecs.DescribeDisksRequest{
			RegionId:   dara.String(regionID),
			InstanceId: dara.String(instanceID),
			PageNumber: &page,
			PageSize:   &pageSize,
		})
		if err != nil {
			return out, err
		}
		body := resp.Body
		if body == nil || body.Disks == nil || body.Disks.Disk == nil {
			break
		}
		disks := body.Disks.Disk
		for _, d := range disks {
			if d != nil && d.DiskId != nil {
				id := strings.TrimSpace(*d.DiskId)
				if id != "" {
					out = append(out, id)
				}
			}
		}
		if len(disks) < int(pageSize) {
			break
		}
		page++
	}
	return out, nil
}

func deleteDiskBestEffort(client *ecs.Client, diskID string, requestIDs *[]string) error {
	resp, err := client.DeleteDisk(&ecs.DeleteDiskRequest{DiskId: dara.String(diskID)})
	if err != nil {
		raw := err.Error()
		if strings.Contains(raw, "InvalidDiskId.NotFound") ||
			strings.Contains(raw, "IncorrectDiskStatus") ||
			strings.Contains(raw, "InvalidOperation.Conflict") ||
			strings.Contains(raw, "does not exist") {
			return nil
		}
		return err
	}
	if requestIDs != nil {
		if rid := extractStopRequestID(resp); rid != "" {
			*requestIDs = append(*requestIDs, rid)
		}
	}
	return nil
}

func extractStopRequestID(resp interface{}) string {
	switch r := resp.(type) {
	case *ecs.DeleteInstanceResponse:
		if r != nil && r.Body != nil && r.Body.RequestId != nil {
			return strings.TrimSpace(*r.Body.RequestId)
		}
	case *ecs.DeleteDiskResponse:
		if r != nil && r.Body != nil && r.Body.RequestId != nil {
			return strings.TrimSpace(*r.Body.RequestId)
		}
	}
	return ""
}
