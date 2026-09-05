package aliyun

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func TestBuildRunInstancesRequestCommentScopedInstanceName(t *testing.T) {
	taskID := "task_876810593758113792"
	java := "cmt_876810842430009344"
	lisp := "cmt_876810929306628096"
	hw := map[string]interface{}{"instance_type": "ecs.t5-lc1m1.small"}
	javaReq, javaParams := buildRunInstancesRequest(StartVMInput{
		RegionID: "cn-hangzhou", ImageID: "m-img", SecurityGroupID: "sg-1",
		VSwitchID: "vsw-1", TaskID: taskID, CommentID: java,
	}, hw, "tok")
	lispReq, _ := buildRunInstancesRequest(StartVMInput{
		RegionID: "cn-hangzhou", ImageID: "m-img", SecurityGroupID: "sg-1",
		VSwitchID: "vsw-1", TaskID: taskID, CommentID: lisp,
	}, hw, "tok")
	if javaReq == nil || javaReq.InstanceName == nil {
		t.Fatal("InstanceName required")
	}
	want := ECSInstanceName(taskID, java)
	if got := *javaReq.InstanceName; got != want {
		t.Fatalf("InstanceName=%q want %q", got, want)
	}
	if lispReq == nil || lispReq.InstanceName == nil {
		t.Fatal("second InstanceName required")
	}
	if *lispReq.InstanceName == *javaReq.InstanceName {
		t.Fatalf("two comments must not share InstanceName %q", *javaReq.InstanceName)
	}
	if fmtName, _ := javaParams["InstanceName"].(string); fmtName != want {
		t.Fatalf("requestParams InstanceName=%q want %q", fmtName, want)
	}

	taskLevel, params := buildRunInstancesRequest(StartVMInput{
		RegionID: "cn-hangzhou", ImageID: "m-img", SecurityGroupID: "sg-1",
		VSwitchID: "vsw-1", TaskID: taskID,
	}, hw, "tok")
	if taskLevel == nil {
		t.Fatal("task-level request required")
	}
	if taskLevel.InstanceName != nil && strings.TrimSpace(*taskLevel.InstanceName) != "" {
		t.Fatalf("empty comment_id must not set task-level InstanceName, got %q", *taskLevel.InstanceName)
	}
	if name, _ := params["InstanceName"].(string); name != "" {
		t.Fatalf("empty comment_id requestParams InstanceName=%q", name)
	}
}

func TestEncodeUserDataBase64(t *testing.T) {
	plain := "#!/bin/bash\necho hello\n"
	got := encodeUserDataBase64(plain)
	want := base64.StdEncoding.EncodeToString([]byte(plain))
	if got != want {
		t.Fatalf("encodeUserDataBase64() = %q, want %q", got, want)
	}
	decoded, err := base64.StdEncoding.DecodeString(got)
	if err != nil {
		t.Fatal(err)
	}
	if string(decoded) != plain {
		t.Fatalf("decoded = %q, want %q", string(decoded), plain)
	}
}

func TestFormatStartVMErrorZoneNotOnSale(t *testing.T) {
	sdkErr := errors.New("SDKError: StatusCode: 403 Code: Zone.NotOnSale Message: zone not on sale")
	got := FormatStartVMError(sdkErr, "cn-hongkong-d")
	if got == nil {
		t.Fatal("expected formatted error")
	}
	msg := got.Error()
	if !strings.Contains(msg, "cn-hongkong-d") {
		t.Fatalf("zone id missing: %q", msg)
	}
	if strings.Contains(msg, "SDKError") {
		t.Fatalf("should not expose raw SDK: %q", msg)
	}
	if !strings.Contains(msg, "已停售或无可售资源") {
		t.Fatalf("unexpected message: %q", msg)
	}
}

func TestIsPermanentStartVMError(t *testing.T) {
	if !IsPermanentStartVMError(errors.New("InvalidAccountStatus.NotEnoughBalance")) {
		t.Fatal("balance error should be permanent")
	}
	if !IsPermanentStartVMError(errors.New("IdempotentParameterMismatch code: 403")) {
		t.Fatal("idempotent mismatch after internal retries should be permanent")
	}
	if !IsPermanentStartVMError(&NoStockError{ZoneID: "cn-hongkong-b", InstanceType: "ecs.hfg7.large"}) {
		t.Fatal("no stock error should be permanent")
	}
	if IsPermanentStartVMError(errors.New("throttling")) {
		t.Fatal("throttling should stay retryable")
	}
}

func TestFormatStartVMErrorNoStock(t *testing.T) {
	sdkErr := errors.New("SDKError: StatusCode: 403 Code: OperationDenied.NoStock Message: out of stock")
	got := FormatStartVMError(&NoStockError{ZoneID: "cn-hongkong-b", InstanceType: "ecs.hfg7.large"}, "cn-hongkong-b")
	if got == nil {
		t.Fatal("expected formatted error")
	}
	msg := got.Error()
	if !strings.Contains(msg, "cn-hongkong-b") || !strings.Contains(msg, "ecs.hfg7.large") {
		t.Fatalf("unexpected message: %q", msg)
	}
	if strings.Contains(msg, "SDKError") {
		t.Fatalf("should not expose raw SDK: %q", msg)
	}
	_ = sdkErr
}

func TestNoStockErrorMessage(t *testing.T) {
	err := &NoStockError{ZoneID: "cn-hongkong-b", InstanceType: "ecs.hfg7.large"}
	msg := err.Error()
	if !strings.Contains(msg, "cn-hongkong-b") || !strings.Contains(msg, "ecs.hfg7.large") {
		t.Fatalf("unexpected message: %q", msg)
	}
	if strings.Contains(msg, "其他可用区") {
		t.Fatalf("should not mention other zones: %q", msg)
	}
	if _, ok := AsNoStockError(err); !ok {
		t.Fatal("AsNoStockError should match typed error")
	}
}

func TestIsNoStockError(t *testing.T) {
	if !IsNoStockError(errors.New("OperationDenied.NoStock")) {
		t.Fatal("expected true")
	}
	if IsNoStockError(errors.New("other")) {
		t.Fatal("expected false")
	}
}

func TestIsZoneNotOnSale(t *testing.T) {
	if !IsZoneNotOnSale(errors.New("Zone.NotOnSale")) {
		t.Fatal("expected true")
	}
	if IsZoneNotOnSale(errors.New("other")) {
		t.Fatal("expected false")
	}
}
