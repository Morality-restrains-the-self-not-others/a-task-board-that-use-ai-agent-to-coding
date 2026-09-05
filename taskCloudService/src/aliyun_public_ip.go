package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	ecsclient "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/dara"
)

// persistStartVmInstanceBinding 在 RunInstances 成功后立即把 instance_id（+ launch_request_id）
// 落到对应评论 CSC 行，并写 last_runtime_status=Starting。必须更新至少一行。
// 匹配优先级：csc_id 精确评论行 > 同 task+comment 空 instance 行。禁止写任务级。
func persistStartVmInstanceBinding(eventData map[string]interface{}, instanceID, launchRequestID string) error {
	taskID := trim(strField(eventData, "task_id"))
	instanceID = trim(instanceID)
	if taskID == "" || instanceID == "" {
		return fmt.Errorf("task_id and instance_id required")
	}
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	commentID := trim(strField(eventData, "comment_id"))
	cscID := trim(strField(eventData, "csc_id"))
	if cscID == "" && commentID == "" {
		logWarn(fmt.Sprintf("event=start_vm_instance_unscoped_skipped task_id=%s instance_id=%s", taskID, instanceID), taskID)
		return fmt.Errorf("start-vm 缺少 comment_id/csc_id，拒绝写入任务级 instance")
	}

	setSQL := `UPDATE cloud_server_configs SET instance_id=?, launch_request_id=?, terminal_released=0, last_runtime_status=IF(last_runtime_status='Running', last_runtime_status, ?), updated_at=CURRENT_TIMESTAMP`
	tryUpdate := func(q string, args ...interface{}) (int64, error) {
		res, err := db.Exec(q, args...)
		if err != nil {
			return 0, err
		}
		n, _ := res.RowsAffected()
		return n, nil
	}

	if cscID != "" {
		n, err := tryUpdate(setSQL+` WHERE id=? AND TRIM(COALESCE(comment_id,''))!=''`,
			instanceID, launchRequestID, machineRuntimeStarting, cscID)
		if err != nil {
			logWarn("persist start instance_id failed: "+err.Error(), taskID)
			return err
		}
		if n >= 1 || commentCSCAlreadyBound(cscID, "", "", instanceID) {
			logStartVmInstancePersisted(eventData, taskID, commentID, cscID, instanceID, launchRequestID)
			return nil
		}
		logWarn(fmt.Sprintf("event=start_vm_instance_csc_miss task_id=%s csc_id=%s instance_id=%s", taskID, cscID, instanceID), taskID)
	}
	if commentID != "" {
		n, err := tryUpdate(setSQL+` WHERE task_id=? AND comment_id=? AND (COALESCE(instance_id,'')='' OR instance_id=?)`,
			instanceID, launchRequestID, machineRuntimeStarting, taskID, commentID, instanceID)
		if err != nil {
			logWarn("persist start instance_id failed: "+err.Error(), taskID)
			return err
		}
		if n >= 1 || commentCSCAlreadyBound("", taskID, commentID, instanceID) {
			logStartVmInstancePersisted(eventData, taskID, commentID, cscID, instanceID, launchRequestID)
			return nil
		}
		if err := insertCommentCSCForStartPersist(eventData, instanceID, launchRequestID); err == nil {
			logStartVmInstancePersisted(eventData, taskID, commentID, cscID, instanceID, launchRequestID)
			return nil
		} else if !errors.Is(err, errPersistCommentCSCIncomplete) {
			logWarn("persist start instance_id insert failed: "+err.Error(), taskID)
			return err
		}
	}
	logWarn(fmt.Sprintf("event=start_vm_instance_persist_zero_rows task_id=%s comment_id=%s csc_id=%s instance_id=%s",
		taskID, commentID, cscID, instanceID), taskID)
	return fmt.Errorf("persist instance_id 未更新任何评论 CSC 行")
}

func logStartVmInstancePersisted(eventData map[string]interface{}, taskID, commentID, cscID, instanceID, launchRequestID string) {
	tid := persistStartTraceAfterInstanceBind(eventData)
	if tid == "" {
		tid = taskID
	}
	logInfo(fmt.Sprintf("event=start_vm_instance_persisted task_id=%s comment_id=%s csc_id=%s instance_id=%s launch_request_id=%s",
		taskID, commentID, cscID, instanceID, launchRequestID), tid)
}

func persistStartTraceAfterInstanceBind(eventData map[string]interface{}) string {
	if eventData == nil {
		return ""
	}
	taskID := trim(strField(eventData, "task_id"))
	commentID := trim(strField(eventData, "comment_id"))
	if taskID == "" || commentID == "" {
		return ""
	}
	return ensureCommentBindingStartTraceID(taskID, commentID, strField(eventData, "trace_id"))
}

var errPersistCommentCSCIncomplete = fmt.Errorf("persist comment CSC identity incomplete")

// insertCommentCSCForStartPersist 在 UPDATE 0 行时插入评论 CSC，避免阿里云实例成为孤儿。
func insertCommentCSCForStartPersist(eventData map[string]interface{}, instanceID, launchRequestID string) error {
	taskID := trim(strField(eventData, "task_id"))
	commentID := trim(strField(eventData, "comment_id"))
	companyID := trim(strField(eventData, "company_id"))
	workspaceID := trim(strField(eventData, "workspace_id"))
	if taskID == "" || commentID == "" || companyID == "" || workspaceID == "" {
		return errPersistCommentCSCIncomplete
	}
	platform := trim(strField(eventData, "cloud_platform_type"))
	if platform == "" {
		platform = trim(strField(eventData, "platform"))
	}
	cfg := CloudServerConfig{
		ID:                genID("csc"),
		CompanyID:         companyID,
		WorkspaceID:       workspaceID,
		TaskID:            taskID,
		CommentID:         commentID,
		Platform:          platform,
		Region:            trim(strField(eventData, "region_id")),
		ZoneID:            trim(strField(eventData, "zone_id")),
		AuthorizationID:   trim(strField(eventData, "authorization_id")),
		SecurityGroupID:   trim(strField(eventData, "security_group_id")),
		VswitchID:         trim(strField(eventData, "vswitch_id")),
		InstanceID:        trim(instanceID),
		LaunchRequestID:   trim(launchRequestID),
		LastRuntimeStatus: machineRuntimeStarting,
	}
	if commentCSCCloudMetaIllegal(&cfg) {
		return errPersistCommentCSCIncomplete
	}
	if err := upsertCloudServerConfig(cfg); err != nil {
		return err
	}
	logInfo(fmt.Sprintf("event=start_vm_instance_persisted_insert task_id=%s comment_id=%s csc_id=%s instance_id=%s",
		taskID, commentID, cfg.ID, instanceID), taskID)
	return nil
}

func commentCSCAlreadyBound(cscID, taskID, commentID, instanceID string) bool {
	if db == nil || trim(instanceID) == "" {
		return false
	}
	var existing string
	var err error
	if trim(cscID) != "" {
		err = db.QueryRow(`SELECT instance_id FROM cloud_server_configs WHERE id=? AND TRIM(COALESCE(comment_id,''))!=''`, trim(cscID)).Scan(&existing)
	} else {
		err = db.QueryRow(`SELECT instance_id FROM cloud_server_configs WHERE task_id=? AND comment_id=?`, trim(taskID), trim(commentID)).Scan(&existing)
	}
	return err == nil && trim(existing) == trim(instanceID)
}

func bindStartVmInstanceOrError(eventData map[string]interface{}, instanceID, launchRequestID string) error {
	instanceID = trim(instanceID)
	taskID := trim(strField(eventData, "task_id"))
	if instanceID == "" {
		logWarn(fmt.Sprintf("event=start_vm_instance_empty task_id=%s comment_id=%s", taskID, trim(strField(eventData, "comment_id"))), taskID)
		return fmt.Errorf("RunInstances 未返回 instance_id，禁止标记启动成功")
	}
	if err := persistStartVmInstanceBinding(eventData, instanceID, launchRequestID); err != nil {
		logWarn(fmt.Sprintf("event=start_vm_instance_bind_failed task_id=%s instance_id=%s err=%s", taskID, instanceID, err.Error()), taskID)
		return err
	}
	return nil
}

// persistCloudServerPublicIPByInstanceID 把公网 IP 写入已绑定该 instance_id 的 CSC 行。
// RunInstances 后 instance_id 先落库、公网 IP 稍后才分配；server-content 只读 CSC.public_ip。
//
// 故意不写 server_url：公网 IP 就绪 ≠ 容器 :8080 已监听。若此处写入
// http://{ip}:8080，commentCSCHasRuntime / ccbTryPromoteStartingToRunning 会把
// binding 误升为 running（UI「容器已就绪，服务可用」），而网关 dial 仍 connection refused。
// server_url 仅由 register-reachability（容器主动登记）写入。serverURL 参数保留兼容调用方，忽略。
func persistCloudServerPublicIPByInstanceID(instanceID, publicIP, serverURL string) {
	instanceID = trim(instanceID)
	publicIP = trim(publicIP)
	_ = trim(serverURL) // 兼容旧调用；不得落库推测性 URL
	if instanceID == "" || publicIP == "" || db == nil {
		return
	}
	if _, err := db.Exec(
		`UPDATE cloud_server_configs SET public_ip=?, last_runtime_status=?, terminal_released=0, updated_at=CURRENT_TIMESTAMP
		 WHERE instance_id=? AND TRIM(COALESCE(comment_id,''))!=''`,
		publicIP, machineRuntimeRunning, instanceID,
	); err != nil {
		logWarn("persist public_ip failed: "+err.Error(), instanceID)
		return
	}
	logInfo(fmt.Sprintf("event=start_vm_public_ip_persisted instance_id=%s public_ip=%s last_runtime_status=Running", instanceID, publicIP), instanceID)
}

// pollInstancePublicIP retries DescribeInstances up to maxRetries times
// to get the public IP (which may not be available immediately after RunInstances).
// Matches Django's behavior: max 5 retries, 3s apart.
func pollInstancePublicIP(accessKey, secretKey, regionID, instanceID string, maxRetries int, interval time.Duration) (publicIP, serverURL string) {
	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			time.Sleep(interval)
		}
		ip, url := queryInstancePublicIPFn(accessKey, secretKey, regionID, instanceID)
		if ip != "" {
			return ip, url
		}
	}
	return "", ""
}

// queryInstancePublicIPFn 可在单测中替换，避免真实阿里云 DescribeInstances。
var queryInstancePublicIPFn = queryInstancePublicIP

func queryInstancePublicIP(accessKey, secretKey, regionID, instanceID string) (publicIP, serverURL string) {
	client, err := newECSClient(accessKey, secretKey, regionID)
	if err != nil {
		return "", ""
	}
	resp, err := client.DescribeInstances(&ecsclient.DescribeInstancesRequest{
		RegionId:    dara.String(regionID),
		InstanceIds: dara.String(fmt.Sprintf(`["%s"]`, instanceID)),
	})
	if err != nil || resp.Body == nil {
		return "", ""
	}
	instances := resp.Body.Instances
	if instances == nil || len(instances.Instance) == 0 {
		return "", ""
	}
	inst := instances.Instance[0]
	pubIP := ""
	if inst.PublicIpAddress != nil && len(inst.PublicIpAddress.IpAddress) > 0 {
		pubIP = derefString(inst.PublicIpAddress.IpAddress[0])
	}
	// VPC + EIP 场景：公网地址在 EipAddress，不在 PublicIpAddress
	if pubIP == "" && inst.EipAddress != nil && inst.EipAddress.IpAddress != nil {
		pubIP = strings.TrimSpace(*inst.EipAddress.IpAddress)
	}
	url := ""
	if pubIP != "" {
		url = fmt.Sprintf("http://%s:8080", pubIP)
	}
	return pubIP, url
}
