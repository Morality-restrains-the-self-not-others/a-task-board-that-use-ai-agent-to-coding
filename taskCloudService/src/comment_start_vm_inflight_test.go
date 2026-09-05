package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLockCommentStartVMSerializesSameComment(t *testing.T) {
	held := make(chan struct{})
	var order []int
	go func() {
		unlock := lockCommentStartVM("co-lock", "task-lock", "cmt-lock")
		defer unlock()
		order = append(order, 1)
		close(held)
		time.Sleep(30 * time.Millisecond)
		order = append(order, 2)
	}()
	<-held
	unlock := lockCommentStartVM("co-lock", "task-lock", "cmt-lock")
	order = append(order, 3)
	unlock()
	if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Fatalf("order=%v want [1 2 3] (second waiter after first critical section)", order)
	}
}

func TestCommentStartVMBusyPeekDoesNotHold(t *testing.T) {
	if commentStartVMBusy("co-busy", "task-busy", "") {
		t.Fatal("empty comment_id must not be busy")
	}
	unlock := lockCommentStartVM("co-busy", "task-busy", "cmt-busy")
	if !commentStartVMBusy("co-busy", "task-busy", "cmt-busy") {
		unlock()
		t.Fatal("held lock must report busy")
	}
	if commentStartVMBusy("co-busy", "task-busy", "cmt-other") {
		unlock()
		t.Fatal("different comment must not share gate")
	}
	unlock()
	if commentStartVMBusy("co-busy", "task-busy", "cmt-busy") {
		t.Fatal("released lock must not stay busy")
	}
}

func TestBootstrapCommentCSCRuntime_SkipsWhenStartVMBusy(t *testing.T) {
	setupCloudTestDB(t)
	var hits int
	prev := startCommentCSCBootstrap
	startCommentCSCBootstrap = func(csc *CloudServerConfig) { hits++ }
	t.Cleanup(func() { startCommentCSCBootstrap = prev })

	csc := &CloudServerConfig{
		ID:          "csc-busy-skip",
		CompanyID:   "t1",
		WorkspaceID: "ws1",
		TaskID:      "task-boot-busy",
		CommentID:   "cmt-boot-busy",
		Platform:    "aliyun",
		Region:      "cn-qingdao",
	}
	unlock := lockCommentStartVM(csc.CompanyID, csc.TaskID, csc.CommentID)
	defer unlock()

	out, err := bootstrapCommentCSCRuntime(csc)
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if out == nil {
		t.Fatal("expected csc")
	}
	if hits != 0 {
		t.Fatalf("busy start-vm must skip bootstrap HTTP, hits=%d", hits)
	}
}

func TestBootstrapCommentCSCRuntime_SkipsWhenDBInstanceBound(t *testing.T) {
	setupCloudTestDB(t)
	var hits int
	prev := startCommentCSCBootstrap
	startCommentCSCBootstrap = func(csc *CloudServerConfig) { hits++ }
	t.Cleanup(func() { startCommentCSCBootstrap = prev })

	if err := upsertCloudServerConfig(CloudServerConfig{
		ID:                "csc-db-bound",
		CompanyID:         "t1",
		WorkspaceID:       "ws1",
		TaskID:            "task-boot-bound",
		CommentID:         "cmt-boot-bound",
		Platform:          "aliyun",
		InstanceID:        "i-mention-first",
		LastRuntimeStatus: "Starting",
	}); err != nil {
		t.Fatal(err)
	}
	// binding 侧内存 CSC 常尚未带回 instance_id；须按库内行跳过二次 start-vm。
	mem := &CloudServerConfig{
		ID:          "csc-db-bound",
		CompanyID:   "t1",
		WorkspaceID: "ws1",
		TaskID:      "task-boot-bound",
		CommentID:   "cmt-boot-bound",
		Platform:    "aliyun",
	}
	if _, err := bootstrapCommentCSCRuntime(mem); err != nil {
		t.Fatal(err)
	}
	if hits != 0 {
		t.Fatalf("DB-bound instance must skip bootstrap HTTP, hits=%d", hits)
	}
}

func TestBootstrapCommentCSCRuntime_SiblingStartEventStillTriggers(t *testing.T) {
	setupCloudTestDB(t)
	var hits int
	prev := startCommentCSCBootstrap
	startCommentCSCBootstrap = func(csc *CloudServerConfig) { hits++ }
	t.Cleanup(func() { startCommentCSCBootstrap = prev })

	companyID := "t1"
	taskID := "task-boot-sib"
	if _, _, err := insertPendingStartEvent(companyID, "ws1", taskID, "m1", "cmt-sibling", map[string]interface{}{
		"region_id": "cn-qingdao", "vpc_id": "vpc-1",
	}); err != nil {
		t.Fatal(err)
	}
	csc := &CloudServerConfig{
		ID:          "csc-sib-skip",
		CompanyID:   companyID,
		WorkspaceID: "ws1",
		TaskID:      taskID,
		CommentID:   "cmt-self",
		Platform:    "aliyun",
		Region:      "cn-qingdao",
	}
	if _, err := bootstrapCommentCSCRuntime(csc); err != nil {
		t.Fatal(err)
	}
	if hits != 1 {
		t.Fatalf("sibling start event must not skip this comment bootstrap, hits=%d", hits)
	}
}

func TestBootstrapCommentCSCRuntime_StalePendingOwnEventStillTriggers(t *testing.T) {
	setupCloudTestDB(t)
	var hits int
	prev := startCommentCSCBootstrap
	startCommentCSCBootstrap = func(csc *CloudServerConfig) { hits++ }
	t.Cleanup(func() { startCommentCSCBootstrap = prev })

	companyID := "t1"
	taskID := "task-boot-pending"
	commentID := "cmt-pending-self"
	if _, _, err := insertPendingStartEvent(companyID, "ws1", taskID, "m1", commentID, map[string]interface{}{
		"region_id": "cn-qingdao",
	}); err != nil {
		t.Fatal(err)
	}
	csc := &CloudServerConfig{
		ID:          "csc-pending-self",
		CompanyID:   companyID,
		WorkspaceID: "ws1",
		TaskID:      taskID,
		CommentID:   commentID,
		Platform:    "aliyun",
		Region:      "cn-qingdao",
	}
	if _, err := bootstrapCommentCSCRuntime(csc); err != nil {
		t.Fatal(err)
	}
	if hits != 1 {
		t.Fatalf("leftover pending start event must not block recovery bootstrap, hits=%d", hits)
	}
}

func TestHandleStartVmNative_SkipsWhenCommentInstanceBound(t *testing.T) {
	setupCloudTestDB(t)
	taskID := "task-dup-skip"
	commentID := "cmt-dup-skip"
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID:                "csc-dup-skip",
		CompanyID:         "t1",
		WorkspaceID:       "ws1",
		TaskID:            taskID,
		CommentID:         commentID,
		Platform:          "aliyun",
		InstanceID:        "i-already-bound",
		PublicIP:          "1.2.3.4",
		LastRuntimeStatus: "Starting",
		AuthorizationID:   "auth1",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := insertCommentContainerBinding("t1", taskID, commentID, ccbExecutionIndependent, ""); err != nil {
		t.Fatal(err)
	}
	persistCommentBindingStartTraceID(taskID, commentID, "trace-original-mention")

	body := map[string]interface{}{
		"task_id":            taskID,
		"comment_id":         commentID,
		"container_image_id": "img-1",
		"region_id":          "cn-qingdao",
		"vpc_id":             "vpc-1",
		"security_group_id":  "sg-1",
		"vswitch_id":         "vsw-1",
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/cloud/compute/start-vm/", strings.NewReader(string(raw)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "u-dup")
	req.Header.Set("X-Trace-Id", "trace-second-bootstrap")
	rec := httptest.NewRecorder()
	handleStartVmNative(rec, req, "t1", "ws1")

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["duplicate_skipped"] != true {
		t.Fatalf("want duplicate_skipped, got %v", rec.Body.String())
	}
	if fmt.Sprint(resp["instance_id"]) != "i-already-bound" {
		t.Fatalf("instance_id=%v", resp["instance_id"])
	}
	gotTrace := loadBindingStartTraceID(taskID, commentID)
	if gotTrace != "trace-original-mention" {
		t.Fatalf("start_trace_id=%q overwritten (must keep mention trace)", gotTrace)
	}
}

func TestHandleStartVmAutoNative_SkipsWhenCommentInstanceBound(t *testing.T) {
	setupCloudTestDB(t)
	taskID := "task-dup-auto"
	commentID := "cmt-dup-auto"
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID:                "csc-dup-auto",
		CompanyID:         "c1",
		WorkspaceID:       "w1",
		TaskID:            taskID,
		CommentID:         commentID,
		Platform:          "aliyun",
		InstanceID:        "i-auto-bound",
		LastRuntimeStatus: "Running",
		AuthorizationID:   "auth1",
	}); err != nil {
		t.Fatal(err)
	}
	body := map[string]interface{}{
		"task_id":                    taskID,
		"comment_id":                 commentID,
		"container_image_id":         "img-auto",
		"auto_create_security_group": true,
		"auto_create_vswitch":        true,
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/internal/cloud/compute/start-vm-auto/", strings.NewReader(string(raw)))
	req.Header.Set("X-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleStartVmAutoNative(rec, req, "c1", "w1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"duplicate_skipped":true`) {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestHandleStartVmNative_MockInstanceDoesNotSkip(t *testing.T) {
	setupCloudTestDB(t)
	taskID := "task-dup-mock"
	commentID := "cmt-dup-mock"
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID:                "csc-dup-mock",
		CompanyID:         "t1",
		WorkspaceID:       "ws1",
		TaskID:            taskID,
		CommentID:         commentID,
		Platform:          "aliyun",
		InstanceID:        "mock-not-real",
		LastRuntimeStatus: "Running",
	}); err != nil {
		t.Fatal(err)
	}
	body := map[string]interface{}{
		"task_id":            taskID,
		"comment_id":         commentID,
		"container_image_id": "img-1",
		"region_id":          "cn-qingdao",
		"vpc_id":             "vpc-1",
		"security_group_id":  "sg-1",
		"vswitch_id":         "vsw-1",
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/cloud/compute/start-vm/", strings.NewReader(string(raw)))
	req.Header.Set("X-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleStartVmNative(rec, req, "t1", "ws1")
	if rec.Code == http.StatusOK && strings.Contains(rec.Body.String(), `"duplicate_skipped":true`) {
		t.Fatalf("mock instance must not skip cold start, body=%s", rec.Body.String())
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected auth/cloud error after not skipping, status=%d body=%s", rec.Code, rec.Body.String())
	}
}
