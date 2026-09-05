package main

import (
	"strings"
	"testing"
)

func TestReconcileWorkspaceMachineRuntimesClearsReleased(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_platform_authorizations
		(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active)
		VALUES('auth-rec','aliyun','access_key','sid','skey','', 't1',1)`)
	if err != nil {
		t.Fatal(err)
	}
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "cfg-rec", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-rec", CommentID: "cmt-rec",
		Platform: "aliyun", InstanceID: "i-gone", Region: "cn-hongkong",
		AuthorizationID: "auth-rec", LastRuntimeStatus: "Running",
	}); err != nil {
		t.Fatal(err)
	}

	old := describeInstanceForRuntime
	oldByName := describeInstancesByName
	t.Cleanup(func() {
		describeInstanceForRuntime = old
		describeInstancesByName = oldByName
	})
	describeInstanceForRuntime = func(accessKey, secretKey, regionID, instanceID string) (map[string]interface{}, bool, string, error) {
		return nil, false, "req", nil
	}
	describeInstancesByName = func(accessKey, secretKey, regionID, instanceName string) ([]map[string]interface{}, string, error) {
		return nil, "req", nil
	}

	reconcileWorkspaceMachineRuntimes("t1", "ws1")

	cfg, err := loadCloudServerConfigForComment("t1", "ws1", "task-rec", "cmt-rec")
	if err != nil || cfg == nil {
		t.Fatalf("load: %v", err)
	}
	if strings.TrimSpace(cfg.InstanceID) != "" {
		t.Fatalf("expected cleared instance_id, got %q", cfg.InstanceID)
	}

	counts, err := computeWorkspaceMachineCounts("t1", "ws1")
	if err != nil {
		t.Fatal(err)
	}
	if counts.StartedCount != 0 {
		t.Fatalf("started=%d want 0 after released reconcile", counts.StartedCount)
	}
}
