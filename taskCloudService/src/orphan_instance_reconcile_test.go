package main

import (
	"strings"
	"testing"
	"time"
)

func TestReconcileOrphanInstancesByNameDeletesNonCurrent(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_platform_authorizations
		(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active)
		VALUES('auth-orphan','aliyun','access_key','sid','skey','', 't1',1)`)
	if err != nil {
		t.Fatal(err)
	}
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "cfg-orphan", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task_orphan1",
		CommentID: "cmt_orphan", Platform: "aliyun", InstanceID: "i-current", Region: "cn-hongkong",
		AuthorizationID: "auth-orphan", LastRuntimeStatus: "Running",
	}); err != nil {
		t.Fatal(err)
	}
	now := "2026-07-18 00:00:00"
	_, err = db.Exec(`INSERT INTO cloud_server_config_histories
		(id,company_id,workspace_id,task_id,platform,platform_id,instance_id,region,zone_id,authorization_id,started_at,created_at)
		VALUES('h-orphan','t1','ws1','task_orphan1','aliyun',1,'i-orphan','cn-hongkong','cn-hongkong-d','auth-orphan',?,?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}

	oldDesc := describeInstancesByName
	oldDel := deleteCloudInstanceForOrphan
	t.Cleanup(func() {
		describeInstancesByName = oldDesc
		deleteCloudInstanceForOrphan = oldDel
		orphanReconcileMu.Lock()
		orphanReconcileCache = map[string]orphanReconcileCacheEntry{}
		orphanReconcileMu.Unlock()
	})

	var deleted []string
	var seenNames []string
	describeInstancesByName = func(accessKey, secretKey, regionID, instanceName string) ([]map[string]interface{}, string, error) {
		seenNames = append(seenNames, instanceName)
		switch instanceName {
		case ecsInstanceName("task_orphan1", "cmt_orphan"):
			return []map[string]interface{}{
				{"InstanceId": "i-current", "Status": "Running"},
				{"InstanceId": "i-orphan", "Status": "Running"},
			}, "req", nil
		default:
			t.Fatalf("unexpected InstanceName %q", instanceName)
			return nil, "", nil
		}
	}
	deleteCloudInstanceForOrphan = func(accessKey, secretKey, regionID, instanceID string) error {
		deleted = append(deleted, instanceID)
		return nil
	}

	reconcileOrphanInstancesByName("t1", "ws1")

	if len(deleted) != 1 || deleted[0] != "i-orphan" {
		t.Fatalf("deleted=%v want [i-orphan]", deleted)
	}
	if len(seenNames) < 1 || seenNames[0] != ecsInstanceName("task_orphan1", "cmt_orphan") {
		t.Fatalf("expected comment InstanceName first, seenNames=%v", seenNames)
	}
	cfg, err := loadCloudServerConfigForComment("t1", "ws1", "task_orphan1", "cmt_orphan")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(cfg.InstanceID) != "i-current" {
		t.Fatalf("current binding should remain, got %q", cfg.InstanceID)
	}
	h, err := loadCloudServerConfigHistoryByID("h-orphan")
	if err != nil {
		t.Fatal(err)
	}
	if h.StoppedAt == nil {
		t.Fatal("expected orphan history closed")
	}
}

func TestECSInstanceNamesCommentOnlyNoTaskFallback(t *testing.T) {
	taskID := "task_13838391081043321882"
	cmt := "cmt_42"
	want := taskID + "_" + cmt
	if ecsInstanceName(taskID, cmt) != want {
		t.Fatalf("name=%q want %q", ecsInstanceName(taskID, cmt), want)
	}
	names := ecsInstanceNames(taskID, cmt)
	if len(names) != 1 || names[0] != want {
		t.Fatalf("lookup=%v want [%s]", names, want)
	}
	if ecsInstanceName(taskID, "") != "" {
		t.Fatalf("empty comment must not fall back to task id, got %q", ecsInstanceName(taskID, ""))
	}
	if got := ecsInstanceNames(taskID, ""); got != nil {
		t.Fatalf("empty comment lookup must be empty, got %v", got)
	}
}

func TestECSInstanceNameCommentScopedMatchesContainerName(t *testing.T) {
	taskID := "task_876810593758113792"
	java := "cmt_876810842430009344"
	lisp := "cmt_876810929306628096"
	javaName := ecsInstanceName(taskID, java)
	lispName := ecsInstanceName(taskID, lisp)
	if javaName != taskID+"_"+java {
		t.Fatalf("java InstanceName=%q", javaName)
	}
	if lispName != taskID+"_"+lisp {
		t.Fatalf("lisp InstanceName=%q", lispName)
	}
	if javaName == lispName {
		t.Fatal("two comments must not share InstanceName")
	}
	if javaName != buildCommentMockContainerName(taskID, java) {
		t.Fatalf("InstanceName must equal container name, got %q vs %q", javaName, buildCommentMockContainerName(taskID, java))
	}
	names := ecsInstanceNames(taskID, java)
	if len(names) != 1 || names[0] != javaName {
		t.Fatalf("lookup must be comment-only, got %v", names)
	}
	if ecsInstanceName(taskID, "") != "" {
		t.Fatalf("empty comment must not fall back to task id, got %q", ecsInstanceName(taskID, ""))
	}
}

func TestReconcileOrphanCommentScopedNameDeletesNonCurrent(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_platform_authorizations
		(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active)
		VALUES('auth-orphan','aliyun','access_key','sid','skey','', 't1',1)`)
	if err != nil {
		t.Fatal(err)
	}
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "cfg-cmt", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task_cmt1",
		CommentID: "cmt_a", Platform: "aliyun", InstanceID: "i-cmt-keep",
		Region: "cn-hongkong", AuthorizationID: "auth-orphan", LastRuntimeStatus: "Running",
	}); err != nil {
		t.Fatal(err)
	}

	oldDesc := describeInstancesByName
	oldDel := deleteCloudInstanceForOrphan
	t.Cleanup(func() {
		describeInstancesByName = oldDesc
		deleteCloudInstanceForOrphan = oldDel
		orphanReconcileMu.Lock()
		orphanReconcileCache = map[string]orphanReconcileCacheEntry{}
		orphanReconcileMu.Unlock()
	})

	var deleted []string
	var seenNames []string
	wantName := ecsInstanceName("task_cmt1", "cmt_a")
	describeInstancesByName = func(accessKey, secretKey, regionID, instanceName string) ([]map[string]interface{}, string, error) {
		seenNames = append(seenNames, instanceName)
		if instanceName == wantName {
			return []map[string]interface{}{
				{"InstanceId": "i-cmt-keep", "Status": "Running"},
				{"InstanceId": "i-cmt-orphan", "Status": "Running"},
			}, "req", nil
		}
		return nil, "req", nil
	}
	deleteCloudInstanceForOrphan = func(accessKey, secretKey, regionID, instanceID string) error {
		deleted = append(deleted, instanceID)
		return nil
	}

	reconcileOrphanInstancesByName("t1", "ws1")

	if len(deleted) != 1 || deleted[0] != "i-cmt-orphan" {
		t.Fatalf("deleted=%v want [i-cmt-orphan] seenNames=%v", deleted, seenNames)
	}
	if len(seenNames) == 0 || seenNames[0] != wantName {
		t.Fatalf("must describe comment-scoped InstanceName first, seenNames=%v want %q", seenNames, wantName)
	}
}

func TestReconcileOrphanSkipsInstanceOwnedByOtherCSC(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_platform_authorizations
		(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active)
		VALUES('auth-orphan','aliyun','access_key','sid','skey','', 't1',1)`)
	if err != nil {
		t.Fatal(err)
	}
	// Source unbound after idle reuse (empty instance_id) but cloud still named for source comment.
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "cfg-source", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task_source",
		CommentID: "cmt_src", Platform: "aliyun", InstanceID: "", Region: "cn-hongkong",
		AuthorizationID: "auth-orphan", LastRuntimeStatus: "",
	}); err != nil {
		t.Fatal(err)
	}
	// Fork / target still holds the instance.
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "cfg-fork", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task_fork",
		CommentID: "cmt_fork", Platform: "aliyun", InstanceID: "i-shared", Region: "cn-hongkong",
		AuthorizationID: "auth-orphan", LastRuntimeStatus: "Running",
	}); err != nil {
		t.Fatal(err)
	}

	oldDesc := describeInstancesByName
	oldDel := deleteCloudInstanceForOrphan
	t.Cleanup(func() {
		describeInstancesByName = oldDesc
		deleteCloudInstanceForOrphan = oldDel
		orphanReconcileMu.Lock()
		orphanReconcileCache = map[string]orphanReconcileCacheEntry{}
		orphanReconcileMu.Unlock()
	})

	var deleted []string
	describeInstancesByName = func(accessKey, secretKey, regionID, instanceName string) ([]map[string]interface{}, string, error) {
		if instanceName == ecsInstanceName("task_source", "cmt_src") {
			return []map[string]interface{}{
				{"InstanceId": "i-shared", "Status": "Running"},
			}, "req", nil
		}
		return nil, "req", nil
	}
	deleteCloudInstanceForOrphan = func(accessKey, secretKey, regionID, instanceID string) error {
		deleted = append(deleted, instanceID)
		return nil
	}

	reconcileOrphanInstancesByName("t1", "ws1")

	if len(deleted) != 0 {
		t.Fatalf("must not delete cross-CSC owned instance, deleted=%v", deleted)
	}
	forkCfg, err := loadCloudServerConfigForComment("t1", "ws1", "task_fork", "cmt_fork")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(forkCfg.InstanceID) != "i-shared" {
		t.Fatalf("fork binding cleared unexpectedly: %q", forkCfg.InstanceID)
	}
}

func TestParseECSCreationTimeLayouts(t *testing.T) {
	now := time.Now().UTC()
	cases := []string{
		now.Format(time.RFC3339),
		now.Format("2006-01-02T15:04:05Z"),
		now.Format("2006-01-02T15:04Z"),
		now.Format("2006-01-02T15:04:05.000Z"),
		now.Format("2006-01-02T15:04:05Z0700"),
	}
	for _, raw := range cases {
		got, ok := parseECSCreationTime(raw)
		if !ok {
			t.Fatalf("parseECSCreationTime(%q) failed", raw)
		}
		if got.IsZero() {
			t.Fatalf("parseECSCreationTime(%q) zero time", raw)
		}
	}
	if _, ok := parseECSCreationTime("not-a-time"); ok {
		t.Fatal("garbage CreationTime must not parse")
	}
}

func TestIsInstanceFreshlyCreatedUnknownAgeIsNotFreshWhenKeepBound(t *testing.T) {
	fresh, _ := isInstanceFreshlyCreated(map[string]interface{}{"Status": "Running"})
	if fresh {
		t.Fatal("missing CreationTime must not be globally treated as fresh (bound leftovers still recycle)")
	}
}

func TestReconcileOrphanSkipsKeepEmptyUnknownAge(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_platform_authorizations
		(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active)
		VALUES('auth-orphan','aliyun','access_key','sid','skey','', 't1',1)`)
	if err != nil {
		t.Fatal(err)
	}
	// RunInstances 已返回但 csc.instance_id 尚未落库：keep 为空。
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "cfg-boot", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task_boot",
		CommentID: "cmt_boot", Platform: "aliyun", InstanceID: "", Region: "cn-hongkong",
		AuthorizationID: "auth-orphan", LastRuntimeStatus: "",
	}); err != nil {
		t.Fatal(err)
	}

	oldDesc := describeInstancesByName
	oldDel := deleteCloudInstanceForOrphan
	t.Cleanup(func() {
		describeInstancesByName = oldDesc
		deleteCloudInstanceForOrphan = oldDel
		orphanReconcileMu.Lock()
		orphanReconcileCache = map[string]orphanReconcileCacheEntry{}
		orphanReconcileMu.Unlock()
	})

	var deleted []string
	describeInstancesByName = func(accessKey, secretKey, regionID, instanceName string) ([]map[string]interface{}, string, error) {
		if instanceName == ecsInstanceName("task_boot", "cmt_boot") {
			return []map[string]interface{}{
				{"InstanceId": "i-m5e90gb0vd6y283aumev", "Status": "Running"},
			}, "req", nil
		}
		return nil, "req", nil
	}
	deleteCloudInstanceForOrphan = func(accessKey, secretKey, regionID, instanceID string) error {
		deleted = append(deleted, instanceID)
		return nil
	}

	reconcileOrphanInstancesByName("t1", "ws1")

	if len(deleted) != 0 {
		t.Fatalf("must not DeleteInstance while keep empty and CreationTime unknown, deleted=%v", deleted)
	}
}

func TestReconcileOrphanSkipsKeepEmptyMinuteCreationTime(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_platform_authorizations
		(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active)
		VALUES('auth-orphan','aliyun','access_key','sid','skey','', 't1',1)`)
	if err != nil {
		t.Fatal(err)
	}
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "cfg-boot2", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task_boot2",
		CommentID: "cmt_boot2", Platform: "aliyun", InstanceID: "", Region: "cn-hongkong",
		AuthorizationID: "auth-orphan", LastRuntimeStatus: "",
	}); err != nil {
		t.Fatal(err)
	}

	oldDesc := describeInstancesByName
	oldDel := deleteCloudInstanceForOrphan
	t.Cleanup(func() {
		describeInstancesByName = oldDesc
		deleteCloudInstanceForOrphan = oldDel
		orphanReconcileMu.Lock()
		orphanReconcileCache = map[string]orphanReconcileCacheEntry{}
		orphanReconcileMu.Unlock()
	})

	created := time.Now().UTC().Format("2006-01-02T15:04Z")
	var deleted []string
	describeInstancesByName = func(accessKey, secretKey, regionID, instanceName string) ([]map[string]interface{}, string, error) {
		return []map[string]interface{}{
			{"InstanceId": "i-newboot", "Status": "Running", "CreationTime": created},
		}, "req", nil
	}
	deleteCloudInstanceForOrphan = func(accessKey, secretKey, regionID, instanceID string) error {
		deleted = append(deleted, instanceID)
		return nil
	}

	reconcileOrphanInstancesByName("t1", "ws1")

	if len(deleted) != 0 {
		t.Fatalf("Aliyun CreationTime without seconds (age < 5m) must skip, deleted=%v", deleted)
	}
}
