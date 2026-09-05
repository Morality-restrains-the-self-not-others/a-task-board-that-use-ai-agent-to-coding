package main

import (
	"testing"

	ecsclient "github.com/alibabacloud-go/ecs-20140526/v7/client"
)

func TestApplyHardwareToRunInstancesSkipsEmptyDataDiskCategory(t *testing.T) {
	req := &ecsclient.RunInstancesRequest{}
	applyHardwareToRunInstances(req, map[string]interface{}{
		"data_disk_category": "",
		"data_disk_size":     40,
	})
	if req.DataDisk != nil {
		t.Fatalf("empty data_disk_category must not attach data disk, got %+v", req.DataDisk)
	}
}

func TestApplyHardwareToRunInstancesSkipsMissingDataDiskCategory(t *testing.T) {
	req := &ecsclient.RunInstancesRequest{}
	applyHardwareToRunInstances(req, map[string]interface{}{
		"data_disk_size": 40,
	})
	if req.DataDisk != nil {
		t.Fatalf("missing data_disk_category must not attach data disk, got %+v", req.DataDisk)
	}
}

func TestApplyHardwareToRunInstancesAttachesDataDiskWhenCategorySet(t *testing.T) {
	req := &ecsclient.RunInstancesRequest{}
	applyHardwareToRunInstances(req, map[string]interface{}{
		"data_disk_category": "cloud_essd",
		"data_disk_size":     40,
	})
	if len(req.DataDisk) != 1 {
		t.Fatalf("expected 1 data disk, got %d", len(req.DataDisk))
	}
	if cat := derefString(req.DataDisk[0].Category); cat != "cloud_essd" {
		t.Fatalf("category=%q want cloud_essd", cat)
	}
	if size := req.DataDisk[0].Size; size == nil || *size != 40 {
		t.Fatalf("size=%v want 40", size)
	}
}
