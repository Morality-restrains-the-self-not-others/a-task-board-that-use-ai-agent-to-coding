package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	ecsclient "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/dara"
)

type orphanReconcileCacheEntry struct {
	checkedAt time.Time
}

var (
	orphanReconcileMu    sync.Mutex
	orphanReconcileCache = map[string]orphanReconcileCacheEntry{}
)

const orphanReconcileTTL = 30 * time.Second

func orphanReconcileCacheKey(companyID, workspaceID, taskID, commentID string) string {
	return trim(companyID) + "|" + trim(workspaceID) + "|" + trim(taskID) + "|" + trim(commentID)
}

type describeInstancesByNameFn func(accessKey, secretKey, regionID, instanceName string) ([]map[string]interface{}, string, error)

// describeInstancesByName is overridable in tests.
var describeInstancesByName describeInstancesByNameFn = aliyunDescribeInstancesByName

type deleteInstanceFn func(accessKey, secretKey, regionID, instanceID string) error

// deleteCloudInstanceForOrphan is overridable in tests (defaults to aliyunStopVM).
var deleteCloudInstanceForOrphan deleteInstanceFn = func(accessKey, secretKey, regionID, instanceID string) error {
	_, _, err := aliyunStopVMFn(accessKey, secretKey, regionID, instanceID)
	return err
}

// ecsInstanceName 阿里云 InstanceName：与评论容器名相同（task_{taskId}_{commentId}）。
// 缺少 comment_id 时返回空串，不回退任务级 / legacy `task-{taskId}`。
func ecsInstanceName(taskID, commentID string) string {
	return buildCommentMockContainerName(taskID, commentID)
}

// ecsInstanceNames 查找用 InstanceName：仅评论级规范名。
func ecsInstanceNames(taskID, commentID string) []string {
	if name := ecsInstanceName(taskID, commentID); name != "" {
		return []string{name}
	}
	return nil
}

func aliyunDescribeInstancesByName(accessKey, secretKey, regionID, instanceName string) ([]map[string]interface{}, string, error) {
	accessKey = strings.TrimSpace(accessKey)
	secretKey = strings.TrimSpace(secretKey)
	regionID = strings.TrimSpace(regionID)
	instanceName = strings.TrimSpace(instanceName)
	if accessKey == "" || secretKey == "" || regionID == "" || instanceName == "" {
		return nil, "", fmt.Errorf("missing aliyun describe-by-name parameters")
	}
	client, err := ecsclient.NewClient(&openapiutil.Config{
		AccessKeyId:     dara.String(accessKey),
		AccessKeySecret: dara.String(secretKey),
		RegionId:        dara.String(regionID),
	})
	if err != nil {
		return nil, "", err
	}
	req := &ecsclient.DescribeInstancesRequest{
		RegionId:     dara.String(regionID),
		InstanceName: dara.String(instanceName),
		PageNumber:   dara.Int32(1),
		PageSize:     dara.Int32(50),
	}
	resp, err := client.DescribeInstances(req)
	if err != nil {
		return nil, "", err
	}
	requestID := ""
	if resp != nil && resp.Body != nil && resp.Body.RequestId != nil {
		requestID = strings.TrimSpace(*resp.Body.RequestId)
	}
	out := []map[string]interface{}{}
	if resp == nil || resp.Body == nil || resp.Body.Instances == nil {
		return out, requestID, nil
	}
	for _, inst := range resp.Body.Instances.Instance {
		if inst == nil {
			continue
		}
		out = append(out, mapDescribeInstancesInstance(inst))
	}
	return out, requestID, nil
}

// isInstanceInitializingStatus 判断实例是否处于不可安全回收的初始化状态。
func isInstanceInitializingStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "initializing", "starting", "pending", "stopping", "creating":
		return true
	}
	return false
}

// parseECSCreationTime 解析阿里云 DescribeInstances.CreationTime。
// 常见 ISO8601 变体：带秒 RFC3339、无秒 `2006-01-02T15:04Z`、带毫秒。
func parseECSCreationTime(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "<nil>" {
		return time.Time{}, false
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04Z",
		"2006-01-02T15:04:05Z0700",
		"2006-01-02T15:04Z0700",
		"2006-01-02T15:04:05Z07:00",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}

// isInstanceFreshlyCreated 判断实例是否创建于宽限期（5 分钟）内。
// RunInstances 返回后 csc.instance_id 落库存在窗口，期间不做孤儿回收（OPT-20260809-023）。
// 缺 CreationTime / 无法解析时返回 false（由 keep 为空时的 unknown_age 分支单独保护，避免误伤已绑定残留）。
func isInstanceFreshlyCreated(attr map[string]interface{}) (bool, time.Duration) {
	const grace = 5 * time.Minute
	raw := strings.TrimSpace(fmt.Sprintf("%v", attr["CreationTime"]))
	t, ok := parseECSCreationTime(raw)
	if !ok {
		return false, 0
	}
	age := time.Since(t)
	return age < grace, age
}

// reconcileOrphanInstancesByName deletes ECS instances named for this comment CSC
// that are not bound by any workspace CSC (covers double-start leftovers
// invisible to local UI). Task-level template rows (empty comment_id) are skipped.
func reconcileOrphanInstancesByName(companyID, workspaceID string) {
	companyID = trim(companyID)
	workspaceID = trim(workspaceID)
	if companyID == "" || workspaceID == "" || db == nil {
		return
	}

	rows, err := db.Query(
		`SELECT task_id, COALESCE(comment_id,''), COALESCE(instance_id,''), COALESCE(authorization_id,''), COALESCE(region,''), COALESCE(platform,'')
		 FROM cloud_server_configs
		 WHERE company_id=? AND workspace_id=?`,
		companyID, workspaceID,
	)
	if err != nil {
		return
	}
	defer rows.Close()

	type rowT struct {
		taskID, commentID, instanceID, authID, region, platform string
	}
	var list []rowT
	for rows.Next() {
		var r rowT
		if err := rows.Scan(&r.taskID, &r.commentID, &r.instanceID, &r.authID, &r.region, &r.platform); err != nil {
			return
		}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		return
	}

	// Cross-CSC ownership (C): never delete an instance still bound by any task in this workspace.
	owned := map[string]string{}
	for _, r := range list {
		id := strings.TrimSpace(r.instanceID)
		if id == "" || isMockMachineInstanceID(id) {
			continue
		}
		if _, ok := owned[id]; !ok {
			owned[id] = strings.TrimSpace(r.taskID)
		}
	}

	now := time.Now()
	for _, r := range list {
		taskID := strings.TrimSpace(r.taskID)
		if taskID == "" || strings.TrimSpace(r.commentID) == "" || isMockMachineInstanceID(r.instanceID) {
			continue
		}
		names := ecsInstanceNames(taskID, r.commentID)
		if len(names) == 0 {
			continue
		}
		key := orphanReconcileCacheKey(companyID, workspaceID, taskID, r.commentID)
		orphanReconcileMu.Lock()
		ent, cached := orphanReconcileCache[key]
		orphanReconcileMu.Unlock()
		if cached && now.Sub(ent.checkedAt) < orphanReconcileTTL {
			continue
		}

		auth, err := loadCloudAuth(companyID, r.authID)
		if err != nil || auth == nil {
			continue
		}
		region := strings.TrimSpace(r.region)
		if region == "" {
			region = "cn-hongkong"
		}
		found := make([]map[string]interface{}, 0, 4)
		seenCloudID := map[string]struct{}{}
		var describeErr error
		for _, name := range names {
			part, _, err := describeInstancesByName(auth.SecretID, auth.SecretKey, region, name)
			if err != nil {
				describeErr = err
				logInfo(fmt.Sprintf("event=orphan_reconcile_describe_failed task_id=%s name=%s err=%s",
					taskID, name, err.Error()), taskID)
				continue
			}
			for _, attr := range part {
				cloudID := strings.TrimSpace(fmt.Sprintf("%v", attr["InstanceId"]))
				if cloudID == "" {
					continue
				}
				if _, ok := seenCloudID[cloudID]; ok {
					continue
				}
				seenCloudID[cloudID] = struct{}{}
				found = append(found, attr)
			}
		}
		orphanReconcileMu.Lock()
		orphanReconcileCache[key] = orphanReconcileCacheEntry{checkedAt: now}
		orphanReconcileMu.Unlock()
		if describeErr != nil && len(found) == 0 {
			continue
		}
		keep := strings.TrimSpace(r.instanceID)
		for _, attr := range found {
			cloudID := strings.TrimSpace(fmt.Sprintf("%v", attr["InstanceId"]))
			if cloudID == "" || isMockMachineInstanceID(cloudID) {
				continue
			}
			if keep != "" && cloudID == keep {
				continue
			}
			// 初始化中/启动中的实例不可安全停止删除（IncorrectInstanceStatus.Initializing 403）；
			// 且 csc.instance_id 落库存在窗口，新建实例给 5 分钟宽限期再回收（OPT-20260809-023）。
			if status, st := attr["Status"]; st && isInstanceInitializingStatus(strings.TrimSpace(fmt.Sprintf("%v", status))) {
				logInfo(fmt.Sprintf(
					"event=orphan_reconcile_skip_initializing task_id=%s orphan_instance_id=%s keep_instance_id=%s status=%v",
					taskID, cloudID, keep, status,
				), taskID)
				continue
			}
			if fresh, age := isInstanceFreshlyCreated(attr); fresh {
				logInfo(fmt.Sprintf(
					"event=orphan_reconcile_skip_fresh task_id=%s orphan_instance_id=%s keep_instance_id=%s age=%s",
					taskID, cloudID, keep, age,
				), taskID)
				continue
			}
			// keep 为空 = CSC 尚未落 instance_id（或刚被 stop 清空）。此时不得凭
			// 「Running + 无 CreationTime」DeleteInstance——Describe 与落库竞态会
			// 把刚 RunInstances 的节点当孤儿杀掉（i-m5e90gb0vd6y283aumev）。
			if keep == "" {
				createdRaw := strings.TrimSpace(fmt.Sprintf("%v", attr["CreationTime"]))
				if _, ok := parseECSCreationTime(createdRaw); !ok {
					logInfo(fmt.Sprintf(
						"event=orphan_reconcile_skip_unknown_age task_id=%s orphan_instance_id=%s keep_instance_id=%s",
						taskID, cloudID, keep,
					), taskID)
					continue
				}
				st := strings.TrimSpace(fmt.Sprintf("%v", attr["Status"]))
				if st == "" || st == "<nil>" {
					logInfo(fmt.Sprintf(
						"event=orphan_reconcile_skip_unknown_status task_id=%s orphan_instance_id=%s keep_instance_id=%s",
						taskID, cloudID, keep,
					), taskID)
					continue
				}
			}
			if ownerTask, held := owned[cloudID]; held {
				logInfo(fmt.Sprintf(
					"event=orphan_reconcile_skip_owned task_id=%s orphan_instance_id=%s keep_instance_id=%s owner_task=%s owner_hint=cross_csc",
					taskID, cloudID, keep, ownerTask,
				), taskID)
				continue
			}
			logInfo(fmt.Sprintf(
				"event=orphan_reconcile_delete task_id=%s orphan_instance_id=%s keep_instance_id=%s",
				taskID, cloudID, keep,
			), taskID)
			if delErr := deleteCloudInstanceForOrphan(auth.SecretID, auth.SecretKey, region, cloudID); delErr != nil {
				logInfo(fmt.Sprintf("event=orphan_reconcile_delete_failed task_id=%s instance_id=%s err=%s",
					taskID, cloudID, delErr.Error()), taskID)
				continue
			}
			_, _ = clearContainerReachabilityNative(companyID, workspaceID, taskID, "orphan_reconcile", cloudID)
		}
	}
}
