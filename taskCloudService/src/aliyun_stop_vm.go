package main

import (
	"fmt"
	"strings"

	ecsclient "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/dara"
)

// aliyunStopVM releases an ECS instance and best-effort deletes leftover disks.
func aliyunStopVM(accessKey, secretKey, regionID, instanceID string) (map[string]interface{}, []string, error) {
	regionID = strings.TrimSpace(regionID)
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return nil, nil, fmt.Errorf("未提供实例ID")
	}
	if regionID == "" {
		regionID = "cn-hongkong"
	}

	client, err := newECSClient(accessKey, secretKey, regionID)
	if err != nil {
		return nil, nil, err
	}

	requestIDs := []string{}
	diskIDs, listErr := aliyunListDiskIDsForInstance(client, regionID, instanceID)
	if listErr != nil {
		logInfo("DescribeDisks before stop skipped: "+listErr.Error(), "")
	}

	force := true
	forceStop := true
	delReq := &ecsclient.DeleteInstanceRequest{
		InstanceId: dara.String(instanceID),
		Force:      &force,
		ForceStop:  &forceStop,
	}
	delResp, err := client.DeleteInstance(delReq)
	if err != nil {
		return nil, requestIDs, fmt.Errorf("停止虚拟机失败: %w", err)
	}
	if rid := extractECSRequestID(delResp); rid != "" {
		requestIDs = append(requestIDs, rid)
	}

	for _, diskID := range diskIDs {
		if diskID == "" {
			continue
		}
		_ = aliyunDeleteDiskBestEffort(client, diskID, &requestIDs)
	}

	return map[string]interface{}{
		"instance_id": instanceID,
		"region":      regionID,
		"status":      "released",
	}, requestIDs, nil
}

func aliyunListDiskIDsForInstance(client *ecsclient.Client, regionID, instanceID string) ([]string, error) {
	if client == nil {
		return nil, fmt.Errorf("ecs client required")
	}
	out := []string{}
	page := int32(1)
	pageSize := int32(50)
	for {
		resp, err := client.DescribeDisks(&ecsclient.DescribeDisksRequest{
			RegionId:   dara.String(regionID),
			InstanceId: dara.String(instanceID),
			PageNumber: &page,
			PageSize:   &pageSize,
		})
		if err != nil {
			return out, err
		}
		if rid := extractECSRequestID(resp); rid != "" {
			_ = rid
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

func aliyunDeleteDiskBestEffort(client *ecsclient.Client, diskID string, requestIDs *[]string) error {
	resp, err := client.DeleteDisk(&ecsclient.DeleteDiskRequest{DiskId: dara.String(diskID)})
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
		if rid := extractECSRequestID(resp); rid != "" {
			*requestIDs = append(*requestIDs, rid)
		}
	}
	return nil
}

func extractECSRequestID(resp interface{}) string {
	type requestIDCarrier interface {
		GetBody() interface{}
	}
	if resp == nil {
		return ""
	}
	switch r := resp.(type) {
	case *ecsclient.DeleteInstanceResponse:
		if r != nil && r.Body != nil && r.Body.RequestId != nil {
			return strings.TrimSpace(*r.Body.RequestId)
		}
	case *ecsclient.DeleteDiskResponse:
		if r != nil && r.Body != nil && r.Body.RequestId != nil {
			return strings.TrimSpace(*r.Body.RequestId)
		}
	case *ecsclient.DescribeDisksResponse:
		if r != nil && r.Body != nil && r.Body.RequestId != nil {
			return strings.TrimSpace(*r.Body.RequestId)
		}
	}
	return ""
}

// aliyunStopVMFn is swappable in tests.
var aliyunStopVMFn = aliyunStopVM
