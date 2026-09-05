package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

func cloudServerConfigHistoryToJSON(h *CloudServerConfigHistory) map[string]interface{} {
	if h == nil {
		return nil
	}
	cpu, mem := resolveHistoryCPUMemory(h)
	out := map[string]interface{}{
		"id": h.ID, "company_id": h.CompanyID, "workspace_id": h.WorkspaceID, "task_id": h.TaskID,
		"platform": h.Platform, "platform_id": h.PlatformID,
		"instance_id": h.InstanceID, "instance_type_id": h.InstanceTypeID,
		"security_group_id": h.SecurityGroupID, "vswitch_id": h.VswitchID,
		"region": h.Region, "zone_id": h.ZoneID, "authorization_id": h.AuthorizationID,
		"public_ip": h.PublicIP, "server_url": h.ServerURL, "business_api_endpoint": h.BusinessAPIEndpoint,
		"error_reason": h.ErrorReason, "stop_reason": h.StopReason, "runtime_source": h.RuntimeSource,
		"launch_request_id": h.LaunchRequestID,
		"cpu_cores": cpu, "memory_gb": mem, "storage_gb": h.StorageGB,
		"started_at": h.StartedAt, "created_at": h.CreatedAt,
		"hardware_config": map[string]interface{}{
			"cpu_cores": cpu, "memory_gb": mem, "storage_gb": h.StorageGB,
		},
	}
	if h.StoppedAt != nil {
		out["stopped_at"] = *h.StoppedAt
	}
	return out
}

// resolveHistoryCPUMemory 优先用实例规格缓存校正历史记录中漂移的 CPU/内存（磁盘保持落库值）。
func resolveHistoryCPUMemory(h *CloudServerConfigHistory) (cpu int, mem int) {
	if h == nil {
		return 1, 1
	}
	cpu, mem = h.CpuCores, h.MemoryGB
	it := strings.TrimSpace(h.InstanceTypeID)
	if it == "" {
		return cpu, mem
	}
	spec, ok := getCachedInstanceTypeSpec(it)
	if !ok || spec.cpuCores <= 0 {
		return cpu, mem
	}
	cpu = spec.cpuCores
	if m := memoryGBFromInstanceTypeSpec(spec); m > 0 {
		mem = m
	}
	return cpu, mem
}

func historySelectSQL() string {
	return `SELECT id, company_id, workspace_id, task_id, platform, platform_id, instance_id,
		COALESCE(instance_type_id,''), COALESCE(security_group_id,''), COALESCE(vswitch_id,''),
		region, zone_id, authorization_id, public_ip, server_url, business_api_endpoint,
		COALESCE(error_reason,''), stop_reason, runtime_source, COALESCE(launch_request_id,''),
		cpu_cores, memory_gb, storage_gb, started_at, stopped_at, created_at
		FROM cloud_server_config_histories`
}

func listCloudServerConfigHistories(companyID, workspaceID, taskID string) ([]map[string]interface{}, error) {
	return listCloudServerConfigHistoriesFiltered(companyID, workspaceID, taskID, false)
}

func listCloudServerConfigHistoriesFiltered(companyID, workspaceID, taskID string, openOnly bool) ([]map[string]interface{}, error) {
	q := historySelectSQL() + ` WHERE company_id=? AND task_id=?`
	args := []interface{}{companyID, taskID}
	if workspaceID != "" {
		q += ` AND workspace_id=?`
		args = append(args, workspaceID)
	}
	if openOnly {
		q += ` AND stopped_at IS NULL`
	}
	q += ` ORDER BY created_at DESC LIMIT 100`
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		h, err := scanHistoryRow(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, cloudServerConfigHistoryToJSON(h))
	}
	return out, nil
}

func loadLatestCloudServerConfigHistory(companyID, workspaceID, taskID string) (*CloudServerConfigHistory, error) {
	q := historySelectSQL() + ` WHERE company_id=? AND task_id=?`
	args := []interface{}{companyID, taskID}
	if workspaceID != "" {
		q += ` AND workspace_id=?`
		args = append(args, workspaceID)
	}
	q += ` ORDER BY created_at DESC LIMIT 1`
	row := db.QueryRow(q, args...)
	return scanHistoryRow(row.Scan)
}

func loadOpenCloudServerConfigHistory(companyID, workspaceID, taskID string) (*CloudServerConfigHistory, error) {
	q := historySelectSQL() + ` WHERE company_id=? AND task_id=? AND stopped_at IS NULL`
	args := []interface{}{companyID, taskID}
	if workspaceID != "" {
		q += ` AND workspace_id=?`
		args = append(args, workspaceID)
	}
	q += ` ORDER BY started_at DESC, created_at DESC LIMIT 1`
	row := db.QueryRow(q, args...)
	h, err := scanHistoryRow(row.Scan)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return h, err
}

func loadCloudServerConfigHistoryByID(id string) (*CloudServerConfigHistory, error) {
	row := db.QueryRow(historySelectSQL()+` WHERE id=?`, id)
	return scanHistoryRow(row.Scan)
}

func scanHistoryRow(scan func(dest ...interface{}) error) (*CloudServerConfigHistory, error) {
	var h CloudServerConfigHistory
	var started, stopped, created sql.NullString
	err := scan(
		&h.ID, &h.CompanyID, &h.WorkspaceID, &h.TaskID, &h.Platform, &h.PlatformID, &h.InstanceID, &h.InstanceTypeID,
		&h.SecurityGroupID, &h.VswitchID, &h.Region, &h.ZoneID, &h.AuthorizationID, &h.PublicIP, &h.ServerURL,
		&h.BusinessAPIEndpoint, &h.ErrorReason, &h.StopReason, &h.RuntimeSource, &h.LaunchRequestID,
		&h.CpuCores, &h.MemoryGB, &h.StorageGB, &started, &stopped, &created,
	)
	if err != nil {
		return nil, err
	}
	if started.Valid {
		h.StartedAt, _ = time.Parse(time.RFC3339, started.String)
	}
	if stopped.Valid && stopped.String != "" {
		t, _ := time.Parse(time.RFC3339, stopped.String)
		h.StoppedAt = &t
	}
	if created.Valid {
		h.CreatedAt, _ = time.Parse(time.RFC3339, created.String)
	}
	return &h, nil
}

func upsertCloudServerConfigHistory(h CloudServerConfigHistory) error {
	now := time.Now().UTC()
	if h.CreatedAt.IsZero() {
		h.CreatedAt = now
	}
	if h.StartedAt.IsZero() {
		h.StartedAt = h.CreatedAt
	}
	var stopped interface{}
	if h.StoppedAt != nil {
		stopped = *h.StoppedAt
	}
	_, err := db.Exec(
		`INSERT INTO cloud_server_config_histories(
			id, company_id, workspace_id, task_id, platform, platform_id, instance_id, instance_type_id,
			security_group_id, vswitch_id, region, zone_id, authorization_id, public_ip, server_url,
			business_api_endpoint, error_reason, stop_reason, runtime_source, launch_request_id,
			cpu_cores, memory_gb, storage_gb, started_at, stopped_at, created_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON DUPLICATE KEY UPDATE
			company_id=VALUES(company_id), workspace_id=VALUES(workspace_id), task_id=VALUES(task_id),
			platform=VALUES(platform), platform_id=VALUES(platform_id), instance_id=VALUES(instance_id),
			instance_type_id=VALUES(instance_type_id), security_group_id=VALUES(security_group_id),
			vswitch_id=VALUES(vswitch_id), region=VALUES(region), zone_id=VALUES(zone_id),
			authorization_id=VALUES(authorization_id), public_ip=VALUES(public_ip),
			server_url=VALUES(server_url), business_api_endpoint=VALUES(business_api_endpoint),
			error_reason=VALUES(error_reason), stop_reason=VALUES(stop_reason),
			runtime_source=VALUES(runtime_source), launch_request_id=VALUES(launch_request_id),
			cpu_cores=VALUES(cpu_cores), memory_gb=VALUES(memory_gb), storage_gb=VALUES(storage_gb),
			started_at=VALUES(started_at), stopped_at=VALUES(stopped_at), created_at=VALUES(created_at)`,
		h.ID, h.CompanyID, h.WorkspaceID, h.TaskID, h.Platform, h.PlatformID, h.InstanceID, h.InstanceTypeID,
		h.SecurityGroupID, h.VswitchID, h.Region, h.ZoneID, h.AuthorizationID, h.PublicIP, h.ServerURL,
		h.BusinessAPIEndpoint, h.ErrorReason, h.StopReason, h.RuntimeSource, h.LaunchRequestID,
		h.CpuCores, h.MemoryGB, h.StorageGB, h.StartedAt, stopped, h.CreatedAt,
	)
	return err
}

func closeOpenCloudServerConfigHistories(companyID, workspaceID, taskID string, patch map[string]interface{}, instanceID string) (int, error) {
	openRows, err := listCloudServerConfigHistoriesFiltered(companyID, workspaceID, taskID, true)
	if err != nil {
		return 0, err
	}
	instanceID = strings.TrimSpace(instanceID)
	count := 0
	for _, row := range openRows {
		if instanceID != "" {
			rowInst := strings.TrimSpace(fmt.Sprintf("%v", row["instance_id"]))
			if rowInst != instanceID {
				continue
			}
		}
		id := fmt.Sprintf("%v", row["id"])
		body := map[string]interface{}{}
		for k, v := range patch {
			body[k] = v
		}
		if _, ok := body["stopped_at"]; !ok {
			body["stopped_at"] = time.Now().UTC().Format("2006-01-02 15:04:05")
		}
		h, err := loadCloudServerConfigHistoryByID(id)
		if err != nil {
			continue
		}
		applyHistoryPatchFromMap(h, body)
		if err := upsertCloudServerConfigHistory(*h); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func applyHistoryPatchFromMap(h *CloudServerConfigHistory, body map[string]interface{}) {
	tmp := map[string]interface{}{}
	for k, v := range body {
		tmp[k] = v
	}
	applyHistoryPatch(h, tmp)
}

func importCloudServerConfigHistories(rows []CloudServerConfigHistory) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	count := 0
	for _, h := range rows {
		if h.ID == "" || h.CompanyID == "" || h.TaskID == "" {
			return count, fmt.Errorf("history row %d: id, company_id, task_id required", count)
		}
		started := h.StartedAt
		if started.IsZero() {
			started = time.Now().UTC()
		}
		created := h.CreatedAt
		if created.IsZero() {
			created = started
		}
		var stopped interface{}
		if h.StoppedAt != nil {
			stopped = *h.StoppedAt
		}
		_, err := tx.Exec(
			`INSERT INTO cloud_server_config_histories(
				id, company_id, workspace_id, task_id, platform, platform_id, instance_id, instance_type_id,
				security_group_id, vswitch_id, region, zone_id, authorization_id, public_ip, server_url,
				business_api_endpoint, error_reason, stop_reason, runtime_source, launch_request_id,
				cpu_cores, memory_gb, storage_gb, started_at, stopped_at, created_at
			) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
			ON DUPLICATE KEY UPDATE
				company_id=VALUES(company_id), workspace_id=VALUES(workspace_id), task_id=VALUES(task_id),
				platform=VALUES(platform), platform_id=VALUES(platform_id), instance_id=VALUES(instance_id),
				instance_type_id=VALUES(instance_type_id), security_group_id=VALUES(security_group_id),
				vswitch_id=VALUES(vswitch_id), region=VALUES(region), zone_id=VALUES(zone_id),
				authorization_id=VALUES(authorization_id), public_ip=VALUES(public_ip),
				server_url=VALUES(server_url), business_api_endpoint=VALUES(business_api_endpoint),
				error_reason=VALUES(error_reason), stop_reason=VALUES(stop_reason),
				runtime_source=VALUES(runtime_source), launch_request_id=VALUES(launch_request_id),
				cpu_cores=VALUES(cpu_cores), memory_gb=VALUES(memory_gb), storage_gb=VALUES(storage_gb),
				started_at=VALUES(started_at), stopped_at=VALUES(stopped_at), created_at=VALUES(created_at)`,
			h.ID, h.CompanyID, h.WorkspaceID, h.TaskID, h.Platform, h.PlatformID, h.InstanceID, h.InstanceTypeID,
			h.SecurityGroupID, h.VswitchID, h.Region, h.ZoneID, h.AuthorizationID, h.PublicIP, h.ServerURL,
			h.BusinessAPIEndpoint, h.ErrorReason, h.StopReason, h.RuntimeSource, h.LaunchRequestID,
			h.CpuCores, h.MemoryGB, h.StorageGB, started, stopped, created,
		)
		if err != nil {
			return count, err
		}
		count++
	}
	return count, tx.Commit()
}
