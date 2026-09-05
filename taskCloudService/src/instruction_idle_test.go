package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func idleProbeServer(t *testing.T, cloneDone bool) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/requirements/task-gate" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"clone_done": cloneDone})
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestInternalWorkspaceMachinePolicyDefaultsTo5(t *testing.T) {
	setupCloudTestDB(t)

	req := httptest.NewRequest(http.MethodGet,
		"/api/internal/cloud/workspace-machine-policy/?company_id=t1&workspace_id=ws1", nil)
	rec := httptest.NewRecorder()
	handleInternalWorkspaceMachinePolicy(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["idle_recycle_minutes"] != float64(5) {
		t.Fatalf("idle_recycle_minutes=%v want 5", body["idle_recycle_minutes"])
	}
	if _, ok := body["machine_release_sts"]; ok {
		t.Fatalf("machine_release_sts should be omitted without role: %v", body["machine_release_sts"])
	}
}

func TestInternalWorkspaceMachinePolicyReadsRow(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_workspace_machine_policies
		(company_id, workspace_id, idle_recycle_minutes, enabled_authorization_ids)
		VALUES ('t1','ws1',15,'[]')`)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet,
		"/api/internal/cloud/workspace-machine-policy/?company_id=t1&workspace_id=ws1", nil)
	rec := httptest.NewRecorder()
	handleInternalWorkspaceMachinePolicy(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"idle_recycle_minutes":15`) {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestMarkInstructionIdleFirstWriteWins(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id,company_id,workspace_id,task_id,comment_id,platform,instance_id,region,authorization_id,created_at,updated_at)
		VALUES ('cfg-idle','t1','ws1','task-1','cmt-1','mock','i-1','cn-test','auth1',NOW(),NOW())`)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := loadCloudServerConfigByID("cfg-idle")
	if err != nil {
		t.Fatal(err)
	}
	if err := markInstructionIdle(nil, cfg); err != nil {
		t.Fatal(err)
	}
	first, ok := loadInstructionIdleSince("cfg-idle")
	if !ok {
		t.Fatal("expected instruction_idle_since after mark")
	}
	time.Sleep(20 * time.Millisecond)
	if err := markInstructionIdle(nil, cfg); err != nil {
		t.Fatal(err)
	}
	second, ok := loadInstructionIdleSince("cfg-idle")
	if !ok {
		t.Fatal("column cleared unexpectedly")
	}
	if !second.Equal(first) {
		t.Fatalf("first-write-wins violated: %v vs %v", first, second)
	}
}

func TestApplyInstructionIdleMissingKeyNoOp(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id,company_id,workspace_id,task_id,comment_id,platform,instance_id,region,authorization_id,created_at,updated_at)
		VALUES ('cfg-miss','t1','ws1','task-1','cmt-1','mock','i-1','cn-test','auth1',NOW(),NOW())`)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := loadCloudServerConfigByID("cfg-miss")
	if err != nil {
		t.Fatal(err)
	}
	applyInstructionIdleFromHeartbeat(nil, cfg, map[string]any{"seq": 1})
	if _, ok := loadInstructionIdleSince("cfg-miss"); ok {
		t.Fatal("missing key must not set instruction_idle_since")
	}
}

func TestApplyInstructionIdleFalseClears(t *testing.T) {
	setupCloudTestDB(t)
	since := time.Now().UTC().Add(-10 * time.Minute).Format("2006-01-02 15:04:05")
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id,company_id,workspace_id,task_id,comment_id,platform,instance_id,region,authorization_id,instruction_idle_since,created_at,updated_at)
		VALUES ('cfg-clr','t1','ws1','task-1','cmt-1','mock','i-1','cn-test','auth1',?,NOW(),NOW())`, since)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := loadCloudServerConfigByID("cfg-clr")
	if err != nil {
		t.Fatal(err)
	}
	applyInstructionIdleFromHeartbeat(nil, cfg, map[string]any{"instruction_idle": false})
	if _, ok := loadInstructionIdleSince("cfg-clr"); ok {
		t.Fatal("false must clear instruction_idle_since")
	}
}

func TestRecycleInstructionIdleReleasesEvenWithServerURL(t *testing.T) {
	setupCloudTestDB(t)
	idleSince := time.Now().UTC().Add(-2 * time.Hour).Format("2006-01-02 15:04:05")
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id,company_id,workspace_id,task_id,comment_id,platform,instance_id,public_ip,region,authorization_id,server_url,instruction_idle_since,last_runtime_status,created_at,updated_at)
		VALUES ('cfg-instr','t1','ws1','task-instr','cmt-1','mock','mock-instr','1.2.3.4','cn-test','auth1','http://127.0.0.1:8080/',?,'Running',NOW(),NOW())`,
		idleSince)
	if err != nil {
		t.Fatal(err)
	}

	recycled, err := recycleIdleMachines(time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if recycled < 1 {
		t.Fatalf("recycled=%d want >=1 (instruction idle despite server_url)", recycled)
	}
	cfg, err := loadCloudServerConfigByID("cfg-instr")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(cfg.InstanceID) != "" {
		t.Fatalf("instance_id=%q want empty after instruction idle release", cfg.InstanceID)
	}
}

func TestRecycleUnmountIdleStillSkipsBusyServerURL(t *testing.T) {
	setupCloudTestDB(t)
	idleSince := time.Now().UTC().Add(-2 * time.Hour).Format("2006-01-02 15:04:05")
	_, err := db.Exec(`INSERT INTO cloud_workspace_machine_policies
		(company_id, workspace_id, idle_recycle_minutes, enabled_authorization_ids)
		VALUES ('t1','ws1',30,'[]')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_server_configs
		(id,company_id,workspace_id,task_id,platform,instance_id,public_ip,region,authorization_id,server_url,idle_since,last_runtime_status,created_at,updated_at)
		VALUES ('cfg-busy2','t1','ws1','task-busy2','mock','mock-busy2','1.2.3.4','cn-test','auth1','http://127.0.0.1:8080/',?,'Running',NOW(),NOW())`,
		idleSince)
	if err != nil {
		t.Fatal(err)
	}
	n, err := recycleIdleMachinesForWorkspace("t1", "ws1", 30, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("unmount idle recycled=%d want 0 when server_url set", n)
	}
}

func TestSTSReleaseDurationSeconds(t *testing.T) {
	if got := stsReleaseDurationSeconds(0); got != 0 {
		t.Fatalf("minutes=0 duration=%d want 0", got)
	}
	if got := stsReleaseDurationSeconds(30); got != 2700 {
		t.Fatalf("minutes=30 duration=%d want 2700", got)
	}
	if got := stsReleaseDurationSeconds(120); got != 3600 {
		t.Fatalf("minutes=120 duration=%d want 3600 (Aliyun default max)", got)
	}
	if got := stsReleaseDurationSeconds(1); got != 960 {
		t.Fatalf("minutes=1 duration=%d want 960", got)
	}
}

func TestBuildSTSReleaseSessionPolicySingleInstance(t *testing.T) {
	pol, err := buildSTSReleaseSessionPolicy("i-abc")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(pol, "ecs:DeleteInstance") {
		t.Fatalf("policy missing DeleteInstance: %s", pol)
	}
	if !strings.Contains(pol, "ecs:DescribeInstances") {
		t.Fatalf("policy missing DescribeInstances: %s", pol)
	}
	if !strings.Contains(pol, "acs:ecs:*:*:instance/i-abc") {
		t.Fatalf("policy missing instance resource: %s", pol)
	}
	if strings.Contains(pol, "GetSessionToken") {
		t.Fatal("session policy must not allow GetSessionToken")
	}
}

func TestMachineReleaseSTSOmittedWithoutRole(t *testing.T) {
	setupCloudTestDB(t)
	prev := stsReleaseMinter
	t.Cleanup(func() { stsReleaseMinter = prev })
	stsReleaseMinter = func(in stsReleaseMintInput) (map[string]any, error) {
		return map[string]any{"access_key_id": "should-not-appear"}, nil
	}
	got := maybeMachineReleaseSTS("t1", "ws1", "task-x", "cmt-x")
	if got != nil {
		t.Fatalf("expected nil STS without CSC/role, got %#v", got)
	}
}

func TestMachineReleaseSTSPresentWhenRoleAndMinter(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_platform_authorizations
		(id, platform_type, authorization_type, secret_id, secret_key, company_id, sts_release_role_arn)
		VALUES ('auth-sts','aliyun','access_key','ak','sk','t1','acs:ram::1:role/release')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_server_configs
		(id,company_id,workspace_id,task_id,comment_id,platform,instance_id,region,authorization_id,created_at,updated_at)
		VALUES ('cfg-sts','t1','ws1','task-sts','cmt-sts','aliyun','i-sts','cn-hangzhou','auth-sts',NOW(),NOW())`)
	if err != nil {
		t.Fatal(err)
	}
	prev := stsReleaseMinter
	t.Cleanup(func() { stsReleaseMinter = prev })
	stsReleaseMinter = func(in stsReleaseMintInput) (map[string]any, error) {
		if in.RoleARN != "acs:ram::1:role/release" {
			t.Fatalf("roleARN=%s", in.RoleARN)
		}
		if !strings.Contains(in.SessionPolicy, "i-sts") {
			t.Fatalf("policy=%s", in.SessionPolicy)
		}
		return map[string]any{"access_key_id": "STS.fake", "expiration": "2099-01-01T00:00:00Z"}, nil
	}
	got := maybeMachineReleaseSTS("t1", "ws1", "task-sts", "cmt-sts")
	if got == nil || got["access_key_id"] != "STS.fake" {
		t.Fatalf("got=%#v", got)
	}
}

func TestMachineReleaseSTSMintsViaAssumeRoleUsingCPASecrets(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_platform_authorizations
		(id, platform_type, authorization_type, secret_id, secret_key, company_id, sts_release_role_arn)
		VALUES ('auth-live','aliyun','access_key','ak-live','sk-live','t1','acs:ram::9:role/idle-release')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_server_configs
		(id,company_id,workspace_id,task_id,comment_id,platform,instance_id,region,authorization_id,created_at,updated_at)
		VALUES ('cfg-live','t1','ws1','task-live','cmt-live','aliyun','i-live','cn-hangzhou','auth-live',NOW(),NOW())`)
	if err != nil {
		t.Fatal(err)
	}
	prevMinter := stsReleaseMinter
	prevAPI := stsAssumeRoleAPI
	t.Cleanup(func() {
		stsReleaseMinter = prevMinter
		stsAssumeRoleAPI = prevAPI
	})
	stsReleaseMinter = mintAliyunReleaseSTS
	var seen stsReleaseMintInput
	stsAssumeRoleAPI = func(in stsReleaseMintInput) (map[string]any, error) {
		seen = in
		return map[string]any{
			"access_key_id":     "STS.live",
			"access_key_secret": "tmp-sk",
			"security_token":    "tok",
			"expiration":        "2099-01-01T00:00:00Z",
		}, nil
	}
	got := maybeMachineReleaseSTS("t1", "ws1", "task-live", "cmt-live")
	if got == nil || got["access_key_id"] != "STS.live" {
		t.Fatalf("got=%#v", got)
	}
	if seen.AccessKey != "ak-live" || seen.SecretKey != "sk-live" {
		t.Fatalf("CPA secrets not passed to AssumeRole: ak=%s", seen.AccessKey)
	}
	if seen.RoleARN != "acs:ram::9:role/idle-release" {
		t.Fatalf("RoleARN=%s", seen.RoleARN)
	}
	if seen.DurationSec != 1200 {
		t.Fatalf("DurationSec=%d want 1200 (default idle 5m + 15m buffer)", seen.DurationSec)
	}
	if !strings.Contains(seen.SessionPolicy, "i-live") {
		t.Fatalf("session policy=%s", seen.SessionPolicy)
	}
	if strings.Contains(seen.SessionPolicy, "GetSessionToken") {
		t.Fatal("AssumeRole session policy must not allow GetSessionToken")
	}
}

func TestMachineReleaseSTSSkippedWhenMinutesZero(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_workspace_machine_policies
		(company_id, workspace_id, idle_recycle_minutes, enabled_authorization_ids)
		VALUES ('t1','ws1',0,'[]')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_platform_authorizations
		(id, platform_type, authorization_type, secret_id, secret_key, company_id, sts_release_role_arn)
		VALUES ('auth-off','aliyun','access_key','ak','sk','t1','acs:ram::1:role/release')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_server_configs
		(id,company_id,workspace_id,task_id,comment_id,platform,instance_id,region,authorization_id,created_at,updated_at)
		VALUES ('cfg-sts-off','t1','ws1','task-off-sts','cmt-off','aliyun','i-off','cn-hangzhou','auth-off',NOW(),NOW())`)
	if err != nil {
		t.Fatal(err)
	}
	prev := stsReleaseMinter
	t.Cleanup(func() { stsReleaseMinter = prev })
	called := false
	stsReleaseMinter = func(in stsReleaseMintInput) (map[string]any, error) {
		called = true
		return map[string]any{"access_key_id": "STS.nope"}, nil
	}
	got := maybeMachineReleaseSTS("t1", "ws1", "task-off-sts", "cmt-off")
	if called {
		t.Fatal("minutes=0 must not AssumeRole")
	}
	if got != nil {
		t.Fatalf("expected nil STS when idle recycle disabled, got %#v", got)
	}
}

func TestMarkInstructionIdleSkippedWhenMinutesZero(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_workspace_machine_policies
		(company_id, workspace_id, idle_recycle_minutes, enabled_authorization_ids)
		VALUES ('t1','ws1',0,'[]')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cloud_server_configs
		(id,company_id,workspace_id,task_id,comment_id,platform,instance_id,region,authorization_id,created_at,updated_at)
		VALUES ('cfg-off','t1','ws1','task-off','cmt-1','mock','i-off','cn-test','auth1',NOW(),NOW())`)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := loadCloudServerConfigByID("cfg-off")
	if err != nil {
		t.Fatal(err)
	}
	if err := markInstructionIdle(nil, cfg); err != nil {
		t.Fatal(err)
	}
	if _, ok := loadInstructionIdleSince("cfg-off"); ok {
		t.Fatal("minutes=0 must not mark instruction idle")
	}
}

func TestRecycleNeverInstructedReadyNoJobsReleases(t *testing.T) {
	srv := idleProbeServer(t, true)
	setupCloudTestDB(t)
	verified := time.Now().UTC().Add(-2 * time.Hour).Format("2006-01-02 15:04:05")
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id,company_id,workspace_id,task_id,comment_id,platform,instance_id,public_ip,region,authorization_id,server_url,userdata_run_verified,last_runtime_status,created_at,updated_at)
		VALUES ('cfg-ready','t1','ws1','task-ready','cmt-ready','mock','mock-ready','1.2.3.4','cn-test','auth1',?,?,'Running',NOW(),NOW())`,
		srv.URL+"/ui/test-token/", verified)
	if err != nil {
		t.Fatal(err)
	}
	n, err := recycleIdleMachines(time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if n < 1 {
		t.Fatalf("recycled=%d want >=1 for ready machine with no instruction", n)
	}
	cfg, err := loadCloudServerConfigByID("cfg-ready")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(cfg.InstanceID) != "" {
		t.Fatalf("instance_id=%q want empty after never-instructed idle recycle", cfg.InstanceID)
	}
}

func TestRecycleNeverInstructedSkipsCloneStillInProgress(t *testing.T) {
	srv := idleProbeServer(t, false)
	setupCloudTestDB(t)
	verified := time.Now().UTC().Add(-2 * time.Hour).Format("2006-01-02 15:04:05")
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id,company_id,workspace_id,task_id,comment_id,platform,instance_id,public_ip,region,authorization_id,server_url,userdata_run_verified,last_runtime_status,created_at,updated_at)
		VALUES ('cfg-cloning','t1','ws1','task-cloning','cmt-cloning','mock','mock-cloning','1.2.3.4','cn-test','auth1',?,?,'Running',NOW(),NOW())`,
		srv.URL+"/ui/test-token/", verified)
	if err != nil {
		t.Fatal(err)
	}
	n, err := recycleIdleMachines(time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("recycled=%d want 0 when clone_done=false (mid-clone)", n)
	}
	cfg, err := loadCloudServerConfigByID("cfg-cloning")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(cfg.InstanceID) == "" {
		t.Fatal("mid-clone machine must keep instance")
	}
}

func TestRecycleNeverInstructedSkipsUnreachableProbe(t *testing.T) {
	setupCloudTestDB(t)
	verified := time.Now().UTC().Add(-2 * time.Hour).Format("2006-01-02 15:04:05")
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id,company_id,workspace_id,task_id,comment_id,platform,instance_id,public_ip,region,authorization_id,server_url,userdata_run_verified,last_runtime_status,created_at,updated_at)
		VALUES ('cfg-unreach','t1','ws1','task-unreach','cmt-unreach','mock','mock-unreach','1.2.3.4','cn-test','auth1','http://127.0.0.1:1/ui/test-token/',?,'Running',NOW(),NOW())`,
		verified)
	if err != nil {
		t.Fatal(err)
	}
	n, err := recycleIdleMachines(time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("recycled=%d want 0 when clone probe unreachable", n)
	}
	cfg, err := loadCloudServerConfigByID("cfg-unreach")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(cfg.InstanceID) == "" {
		t.Fatal("unreachable probe must keep instance")
	}
}

func TestRecycleNeverInstructedSkipsActiveJob(t *testing.T) {
	setupCloudTestDB(t)
	verified := time.Now().UTC().Add(-2 * time.Hour).Format("2006-01-02 15:04:05")
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id,company_id,workspace_id,task_id,comment_id,platform,instance_id,public_ip,region,authorization_id,server_url,userdata_run_verified,last_runtime_status,created_at,updated_at)
		VALUES ('cfg-busy','t1','ws1','task-busy','cmt-busy','mock','mock-busy','1.2.3.4','cn-test','auth1','http://127.0.0.1:8080/',?,'Running',NOW(),NOW())`,
		verified)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := insertJobExecutionEvent(jobExecutionEventRow{
		CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-busy", CommentID: "cmt-busy",
		JobID: "job-run", Seq: 1, Phase: "start", JobStatus: "running",
	}); err != nil {
		t.Fatal(err)
	}
	n, err := recycleIdleMachines(time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("recycled=%d want 0 while instruction job running", n)
	}
	cfg, err := loadCloudServerConfigByID("cfg-busy")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(cfg.InstanceID) == "" {
		t.Fatal("active job must keep instance")
	}
}

func TestRecycleNeverInstructedSkipsFreshUserdata(t *testing.T) {
	setupCloudTestDB(t)
	verified := time.Now().UTC().Add(-1 * time.Minute).Format("2006-01-02 15:04:05")
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id,company_id,workspace_id,task_id,comment_id,platform,instance_id,region,authorization_id,server_url,userdata_run_verified,last_runtime_status,created_at,updated_at)
		VALUES ('cfg-fresh','t1','ws1','task-fresh','cmt-fresh','mock','mock-fresh','cn-test','auth1','http://127.0.0.1:8080/',?,'Running',NOW(),NOW())`,
		verified)
	if err != nil {
		t.Fatal(err)
	}
	n, err := recycleIdleMachines(time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("recycled=%d want 0 when userdata_run_verified is within idle window", n)
	}
}

func TestHandleRuntimeEventMarksInstructionIdleOnBootstrapComplete(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id,company_id,workspace_id,task_id,comment_id,platform,instance_id,region,authorization_id,created_at,updated_at)
		VALUES ('cfg-boot','t1','ws1','task-boot','cmt-boot','mock','mock-boot','cn-test','auth1',NOW(),NOW())`)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := loadCloudServerConfigByID("cfg-boot")
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	handleRuntimeEvent(rec, context.Background(), cfg, map[string]any{"event": "BOOTSTRAP_COMPLETE"}, "t1", "ws1", "task-boot")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if _, ok := loadInstructionIdleSince("cfg-boot"); !ok {
		t.Fatal("BOOTSTRAP_COMPLETE must set instruction_idle_since")
	}
}
