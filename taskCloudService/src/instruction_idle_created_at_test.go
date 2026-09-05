package main

import (
	"strings"
	"testing"
	"time"
)

func TestParseIdleClockCSTWallWhenUTCWouldBeFuture(t *testing.T) {
	now := time.Date(2026, 8, 24, 17, 44, 0, 0, time.UTC)
	// 22:48 CST = 14:48 UTC; parsed as UTC would be in the future.
	got, ok := parseIdleClockPreferPast("2026-08-24 22:48:01", now)
	if !ok {
		t.Fatal("expected parse")
	}
	want := time.Date(2026, 8, 24, 14, 48, 1, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got=%s want=%s", got.UTC(), want)
	}
}

func TestParseIdleClockUTCWhenInThePast(t *testing.T) {
	now := time.Date(2026, 8, 24, 17, 44, 0, 0, time.UTC)
	got, ok := parseIdleClockPreferPast("2026-08-24 16:00:00", now)
	if !ok {
		t.Fatal("expected parse")
	}
	want := time.Date(2026, 8, 24, 16, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got=%s want=%s", got.UTC(), want)
	}
}

func TestRecycleNeverInstructedUsesCreatedAtWhenUserdataNull(t *testing.T) {
	srv := idleProbeServer(t, true)
	setupCloudTestDB(t)
	created := time.Now().UTC().Add(-2 * time.Hour).Format("2006-01-02 15:04:05")
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id,company_id,workspace_id,task_id,comment_id,platform,instance_id,public_ip,region,authorization_id,server_url,last_runtime_status,created_at,updated_at)
		VALUES ('cfg-noud','t1','ws1','task-noud','cmt-noud','mock','mock-noud','1.2.3.4','cn-test','auth1',?,'Running',?,NOW())`,
		srv.URL+"/ui/test-token/", created)
	if err != nil {
		t.Fatal(err)
	}
	n, err := recycleIdleMachines(time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if n < 1 {
		t.Fatalf("recycled=%d want >=1 when userdata_run_verified is NULL but created_at is old", n)
	}
	cfg, err := loadCloudServerConfigByID("cfg-noud")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(cfg.InstanceID) != "" {
		t.Fatalf("instance_id=%q want empty", cfg.InstanceID)
	}
}

func TestRecycleNeverInstructedCreatedAtCSTWallReleases(t *testing.T) {
	srv := idleProbeServer(t, true)
	setupCloudTestDB(t)
	now := time.Now().UTC()
	// CST wall clock ≈ now+6h: UTC parse is future; Shanghai parse is now-2h.
	cstWall := now.Add(6 * time.Hour).Format("2006-01-02 15:04:05")
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id,company_id,workspace_id,task_id,comment_id,platform,instance_id,public_ip,region,authorization_id,server_url,last_runtime_status,created_at,updated_at)
		VALUES ('cfg-cst','t1','ws1','task-cst','cmt-cst','mock','mock-cst','1.2.3.4','cn-test','auth1',?,'Running',?,NOW())`,
		srv.URL+"/ui/test-token/", cstWall)
	if err != nil {
		t.Fatal(err)
	}
	n, err := recycleIdleMachines(now)
	if err != nil {
		t.Fatal(err)
	}
	if n < 1 {
		t.Fatalf("recycled=%d want >=1 for CST created_at wall clock", n)
	}
}

func TestRecycleNeverInstructedNullIdleSinceWithPolicyStillRecycles(t *testing.T) {
	srv := idleProbeServer(t, true)
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_workspace_machine_policies
		(company_id, workspace_id, idle_recycle_minutes, enabled_authorization_ids)
		VALUES ('t1','ws1',5,'[]')`)
	if err != nil {
		t.Fatal(err)
	}
	created := time.Now().UTC().Add(-2 * time.Hour).Format("2006-01-02 15:04:05")
	_, err = db.Exec(`INSERT INTO cloud_server_configs
		(id,company_id,workspace_id,task_id,comment_id,platform,instance_id,public_ip,region,authorization_id,server_url,last_runtime_status,created_at,updated_at)
		VALUES ('cfg-nullidle','t1','ws1','task-nullidle','cmt-nullidle','mock','mock-nullidle','1.2.3.4','cn-test','auth1',?,'Running',?,NOW())`,
		srv.URL+"/ui/test-token/", created)
	if err != nil {
		t.Fatal(err)
	}
	n, err := recycleIdleMachines(time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if n < 1 {
		t.Fatalf("recycled=%d want >=1 (NULL idle_since must not abort instruction idle)", n)
	}
}

func TestRecycleNeverInstructedFreshCreatedAtSkipped(t *testing.T) {
	setupCloudTestDB(t)
	created := time.Now().UTC().Add(-1 * time.Minute).Format("2006-01-02 15:04:05")
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id,company_id,workspace_id,task_id,comment_id,platform,instance_id,region,authorization_id,server_url,last_runtime_status,created_at,updated_at)
		VALUES ('cfg-new','t1','ws1','task-new','cmt-new','mock','mock-new','cn-test','auth1','http://127.0.0.1:8080/','Running',?,NOW())`,
		created)
	if err != nil {
		t.Fatal(err)
	}
	n, err := recycleIdleMachines(time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("recycled=%d want 0 for fresh created_at", n)
	}
}
