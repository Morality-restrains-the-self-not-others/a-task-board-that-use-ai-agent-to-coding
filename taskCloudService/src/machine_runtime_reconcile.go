package main

import (
	"strings"
	"sync"
	"time"
)

type runtimeReconcileCacheEntry struct {
	checkedAt time.Time
}

var (
	runtimeReconcileMu    sync.Mutex
	runtimeReconcileCache = map[string]runtimeReconcileCacheEntry{}
)

const runtimeReconcileTTL = 30 * time.Second

func runtimeReconcileCacheKey(companyID, workspaceID, instanceID string) string {
	return trim(companyID) + "|" + trim(workspaceID) + "|" + trim(instanceID)
}

// reconcileWorkspaceMachineRuntimes refreshes last_runtime_status for non-mock instances
// so work-panel summary/filter do not treat Released/Starting as 「已启动」.
func reconcileWorkspaceMachineRuntimes(companyID, workspaceID string) {
	companyID = trim(companyID)
	workspaceID = trim(workspaceID)
	if companyID == "" || workspaceID == "" || db == nil {
		return
	}

	rows, err := db.Query(
		`SELECT task_id, COALESCE(instance_id,''), COALESCE(authorization_id,''), COALESCE(region,''),
		        COALESCE(last_runtime_status,''), COALESCE(platform,''), COALESCE(public_ip,'')
		 FROM cloud_server_configs
		 WHERE company_id=? AND workspace_id=?
		   AND TRIM(COALESCE(comment_id,'')) != ''
		   AND TRIM(COALESCE(instance_id,'')) != ''`,
		companyID, workspaceID,
	)
	if err != nil {
		return
	}
	defer rows.Close()

	type rowT struct {
		taskID, instanceID, authID, region, lastStatus, platform, publicIP string
	}
	var list []rowT
	for rows.Next() {
		var r rowT
		if err := rows.Scan(&r.taskID, &r.instanceID, &r.authID, &r.region, &r.lastStatus, &r.platform, &r.publicIP); err != nil {
			return
		}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		return
	}

	now := time.Now()
	for _, r := range list {
		instanceID := strings.TrimSpace(r.instanceID)
		if instanceID == "" || isMockMachineInstanceID(instanceID) {
			if isMockMachineInstanceID(instanceID) && normalizeMachineRuntimeStatus(r.lastStatus) == "" {
				_ = setCloudServerLastRuntimeStatus(companyID, workspaceID, r.taskID, "", instanceID, machineRuntimeRunning)
			}
			continue
		}
		status := normalizeMachineRuntimeStatus(r.lastStatus)
		// Fresh Running status within TTL: skip cloud call.
		// 但 public_ip 仍空时必须再 Describe，以便回填（否则 server-content 永久 400）。
		key := runtimeReconcileCacheKey(companyID, workspaceID, instanceID)
		runtimeReconcileMu.Lock()
		ent, cached := runtimeReconcileCache[key]
		runtimeReconcileMu.Unlock()
		hasPublicIP := strings.TrimSpace(r.publicIP) != ""
		if status == machineRuntimeRunning && hasPublicIP && cached && now.Sub(ent.checkedAt) < runtimeReconcileTTL {
			continue
		}
		if status != "" && status != machineRuntimeRunning && !machineRuntimeCountsAsStarting(instanceID, status) &&
			cached && now.Sub(ent.checkedAt) < runtimeReconcileTTL {
			// Recently reconciled non-running transitional/down states.
			continue
		}

		auth, err := loadCloudAuth(companyID, r.authID)
		if err != nil || auth == nil {
			continue
		}
		region := strings.TrimSpace(r.region)
		if region == "" {
			continue
		}
		attr, found, _, err := describeInstanceForRuntime(auth.SecretID, auth.SecretKey, region, instanceID)
		if err != nil {
			if code := aliyunErrorCode(err); code == "InvalidInstanceId.NotFound" || strings.Contains(code, "InvalidInstanceId") {
				applyObservedMachineRuntimeStatus(companyID, workspaceID, r.taskID, instanceID, machineRuntimeReleased, false)
			}
			runtimeReconcileMu.Lock()
			runtimeReconcileCache[key] = runtimeReconcileCacheEntry{checkedAt: now}
			runtimeReconcileMu.Unlock()
			continue
		}
		runtimeStatus := ""
		if found {
			if s, ok := attr["Status"].(string); ok {
				runtimeStatus = s
			}
			if !hasPublicIP {
				if ip := publicIPFromDescribeAttr(attr); ip != "" {
					persistCloudServerPublicIPByInstanceID(instanceID, ip, "")
				}
			}
		}
		applyObservedMachineRuntimeStatus(companyID, workspaceID, r.taskID, instanceID, runtimeStatus, found)
		runtimeReconcileMu.Lock()
		runtimeReconcileCache[key] = runtimeReconcileCacheEntry{checkedAt: now}
		runtimeReconcileMu.Unlock()
	}

	reconcileOrphanInstancesByName(companyID, workspaceID)
}
