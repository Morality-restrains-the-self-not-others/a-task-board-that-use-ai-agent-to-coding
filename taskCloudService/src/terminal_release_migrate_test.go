package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestListInstanceBindingsAndMigrateOff(t *testing.T) {
	setupCloudTestDB(t)
	startMigratedContainerFn = func(companyID, workspaceID, ownerTaskID, containerImageID, containerImageURL string) error {
		return nil
	}
	defer func() { startMigratedContainerFn = startMigratedContainer }()

	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "cfg-a", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-a",
		Platform: "aliyun", InstanceID: "mock-shared", PublicIP: "10.0.0.1",
		LastRuntimeStatus: "Running",
	}); err != nil {
		t.Fatal(err)
	}
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "cfg-b", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-b",
		Platform: "aliyun", InstanceID: "mock-shared", PublicIP: "10.0.0.1",
		ServerURL: "http://10.0.0.1:8080", LastRuntimeStatus: "Running",
	}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet,
		"/api/internal/cloud/compute/instance-bindings/?company_id=t1&workspace_id=ws1&instance_id=mock-shared", nil)
	rec := httptest.NewRecorder()
	handleInternalInstanceBindings(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var listBody map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &listBody); err != nil {
		t.Fatal(err)
	}
	bindings, _ := listBody["bindings"].([]interface{})
	if len(bindings) != 2 {
		t.Fatalf("bindings=%d want 2", len(bindings))
	}

	migBody := `{"tenant_id":"t1","workspace_id":"ws1","owner_task_id":"task-b","from_instance_id":"mock-shared"}`
	req = httptest.NewRequest(http.MethodPost, "/api/internal/cloud/compute/migrate-container-off-instance/",
		strings.NewReader(migBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handleInternalMigrateContainerOffInstance(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("migrate status=%d body=%s", rec.Code, rec.Body.String())
	}
	var migResp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &migResp); err != nil {
		t.Fatal(err)
	}
	newID, _ := migResp["instance_id"].(string)
	if newID == "" || newID == "mock-shared" {
		t.Fatalf("instance_id=%q want new dedicated id", newID)
	}
	if migResp["via"] != "new_mock" {
		t.Fatalf("via=%v want new_mock", migResp["via"])
	}
	if migResp["container_start_triggered"] != true {
		t.Fatalf("container_start_triggered=%v want true", migResp["container_start_triggered"])
	}
	owner, err := loadCloudServerConfig("t1", "ws1", "task-b")
	if err != nil {
		t.Fatal(err)
	}
	if owner.InstanceID != newID {
		t.Fatalf("owner instance_id=%q want %q", owner.InstanceID, newID)
	}
	if owner.ServerURL != "" {
		t.Fatalf("owner server_url should be cleared for recreate, got %q", owner.ServerURL)
	}
}

func TestMigrateRealECSUsesStartVmAutoHook(t *testing.T) {
	setupCloudTestDB(t)
	called := false
	var forbid string
	awaitCalls := 0
	provisionMigratedRealECSFn = func(companyID, workspaceID, ownerTaskID, forbidInstanceID, containerImageID string, owner *CloudServerConfig) (string, error) {
		called = true
		forbid = forbidInstanceID
		return "pending-start-evt1", nil
	}
	defer func() { provisionMigratedRealECSFn = provisionMigratedRealECS }()
	publishContainerMigrateAwaitReadyFn = func(companyID, workspaceID, taskID, instanceID, via, imageID, imageURL string) {
		awaitCalls++
		if via != "start_vm_auto" || taskID != "task-b" {
			t.Fatalf("await via=%s task=%s", via, taskID)
		}
	}
	defer func() { publishContainerMigrateAwaitReadyFn = publishContainerMigrateAwaitReady }()

	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "cfg-b", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-b",
		Platform: "aliyun", InstanceID: "i-real-ecs", PublicIP: "10.0.0.1",
		AuthorizationID: "auth1", ServerURL: "http://10.0.0.1:8080",
		LastRuntimeStatus: "Running", Region: "cn-hangzhou",
	}); err != nil {
		t.Fatal(err)
	}

	res, err := migrateContainerOffInstanceDetailed("t1", "ws1", "task-b", "i-real-ecs")
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected provisionMigratedRealECSFn")
	}
	if forbid != "i-real-ecs" {
		t.Fatalf("forbid=%q", forbid)
	}
	if res.Via != "start_vm_auto" {
		t.Fatalf("via=%s", res.Via)
	}
	if res.InstanceID != "pending-start-evt1" {
		t.Fatalf("instance_id=%s", res.InstanceID)
	}
	if !res.ContainerStartTriggered {
		t.Fatal("expected ContainerStartTriggered for userdata path")
	}
	if awaitCalls != 1 {
		t.Fatalf("await publish calls=%d want 1", awaitCalls)
	}
}

func TestMarkTerminalReleasedExcludesFromIdleReuse(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_platform_authorizations
		(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active)
		VALUES('auth1','aliyun','access_key','sid','skey','', 't1',1)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_workspace_machine_policies
		(company_id, workspace_id, idle_recycle_minutes, enabled_authorization_ids)
		VALUES ('t1','ws1',30,'[]')`)
	if err != nil {
		t.Fatal(err)
	}
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "cfg-term", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-done",
		Platform: "aliyun", InstanceID: "i-done", PublicIP: "10.0.0.2",
		AuthorizationID: "auth1", LastRuntimeStatus: "Running",
	}); err != nil {
		t.Fatal(err)
	}
	body := `{"tenant_id":"t1","workspace_id":"ws1","task_id":"task-done"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/cloud/compute/mark-terminal-released/",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalMarkTerminalReleased(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var instance, status string
	var released int
	if err := db.QueryRow(`SELECT COALESCE(instance_id,''), COALESCE(last_runtime_status,''), COALESCE(terminal_released,0) FROM cloud_server_configs WHERE id='cfg-term'`).Scan(&instance, &status, &released); err != nil {
		t.Fatal(err)
	}
	if instance != "" || status != "Released" || released != 1 {
		t.Fatalf("instance=%q status=%q released=%d", instance, status, released)
	}
}

func TestMarkTerminalReleasedSyncsBindingToReleased(t *testing.T) {
	setupCloudTestDB(t)
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "cfg-sync", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-sync",
		Platform: "aliyun", InstanceID: "i-sync", PublicIP: "10.0.0.3",
		LastRuntimeStatus: "Running",
	}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	for _, c := range []struct {
		id, cid, status string
	}{
		{"b-run", "cmt-run", ccbStatusRunning},
		{"b-completed", "cmt-done", ccbStatusCompleted},
	} {
		if _, err := db.Exec(
			`INSERT INTO cloud_comment_container_bindings(
				id, company_id, workspace_id, task_id, comment_id, execution_mode,
				depends_on_comment_id, status, mock_container_name, csc_id, created_at, updated_at
			) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
			c.id, "t1", "ws1", "task-sync", c.cid, "independent", "",
			c.status, "task-sync_"+c.cid, "csc_"+c.id, now, now,
		); err != nil {
			t.Fatal(err)
		}
	}
	body := `{"tenant_id":"t1","workspace_id":"ws1","task_id":"task-sync"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/cloud/compute/mark-terminal-released/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalMarkTerminalReleased(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var runStatus, doneStatus string
	if err := db.QueryRow(`SELECT status FROM cloud_comment_container_bindings WHERE id='b-run'`).Scan(&runStatus); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT status FROM cloud_comment_container_bindings WHERE id='b-completed'`).Scan(&doneStatus); err != nil {
		t.Fatal(err)
	}
	if runStatus != ccbStatusReleased {
		t.Fatalf("running binding status=%s want %s", runStatus, ccbStatusReleased)
	}
	if doneStatus != ccbStatusCompleted {
		t.Fatalf("completed binding status=%s want %s (must be preserved)", doneStatus, ccbStatusCompleted)
	}
}

func TestInternalMarkCommentTerminalReleased(t *testing.T) {
	setupCloudTestDB(t)
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "cfg-lr", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-lr",
		CommentID: "cmt-lr", Platform: "aliyun",
		LaunchRequestID: "req-123", LastRuntimeStatus: "Starting",
		// OPT-20260821-021: 开机中途终态时 instance 可能已回填，须一并清空
		InstanceID: "i-late-vm", PublicIP: "10.0.0.9", ServerURL: "http://10.0.0.9:8080",
		BusinessAPIEndpoint: "https://api.daydaymoney.com", ContainerVscodeURL: "https://vscode.example.com",
		ErrorReason: "starting",
	}); err != nil {
		t.Fatal(err)
	}
	body := `{"tenant_id":"t1","workspace_id":"ws1","task_id":"task-lr","comment_id":"cmt-lr"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/cloud/compute/mark-comment-terminal-released/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalMarkCommentTerminalReleased(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var lrs string
	var term int
	var instance, publicIP, serverURL string
	if err := db.QueryRow(
		`SELECT last_runtime_status, terminal_released, COALESCE(instance_id,''), COALESCE(public_ip,''), COALESCE(server_url,'') FROM cloud_server_configs WHERE id='cfg-lr'`,
	).Scan(&lrs, &term, &instance, &publicIP, &serverURL); err != nil {
		t.Fatal(err)
	}
	if lrs != "Released" {
		t.Fatalf("last_runtime_status=%q want Released", lrs)
	}
	if term != 1 {
		t.Fatalf("terminal_released=%d want 1", term)
	}
	if instance != "" || publicIP != "" || serverURL != "" {
		t.Fatalf("instance/ip/server_url should be cleared after terminal mark, got %q/%q/%q", instance, publicIP, serverURL)
	}

	// 缺 comment_id 应 400
	bad := httptest.NewRequest(http.MethodPost, "/api/internal/cloud/compute/mark-comment-terminal-released/", strings.NewReader(`{"tenant_id":"t1","task_id":"task-lr"}`))
	bad.Header.Set("Content-Type", "application/json")
	badRec := httptest.NewRecorder()
	handleInternalMarkCommentTerminalReleased(badRec, bad)
	if badRec.Code != http.StatusBadRequest {
		t.Fatalf("bad status=%d want 400", badRec.Code)
	}
}
