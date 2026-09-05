package main

import (
	"database/sql"
	"net/http"
	"strings"
	"time"
)

func handleCloudNetworkList(w http.ResponseWriter, r *http.Request, resourceType string) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, 405, "method not allowed")
		return
	}
	platformType := strings.TrimSpace(r.URL.Query().Get("platform_type"))
	if platformType == "" {
		platformType = "aliyun"
	}
	regionID := strings.TrimSpace(r.URL.Query().Get("region_id"))
	if regionID == "" {
		cloudQueryError(w, 400, "缺少 region_id 参数")
		return
	}
	vpcID := strings.TrimSpace(r.URL.Query().Get("vpc_id"))
	zoneID := strings.TrimSpace(r.URL.Query().Get("zone_id"))
	tenantID := resolveGlobalCloudTenantID(r)
	if tenantID == "" {
		cloudQueryError(w, 400, "无法解析租户，请从租户页面访问或传入 tenant_id")
		return
	}
	if !ensureTenantMember(w, r, tenantID) {
		return
	}
	auth, err := loadFirstCloudAuthForCompany(tenantID, platformType)
	if err != nil {
		cloudQueryError(w, 500, err.Error())
		return
	}
	if auth == nil {
		cloudQueryError(w, 400, "未找到平台 "+platformType+" 的云平台授权，请先在云平台授权页面添加授权")
		return
	}

	var items interface{}
	var rid string
	if useInMemoryCloud() {
		items = []networkResource{}
	} else {
		switch resourceType {
		case "vpc":
			vpcs, reqID, e := aliyunDescribeVpcsRich(auth.SecretID, auth.SecretKey, regionID)
			items, rid, err = vpcs, reqID, e
		case "vswitch":
			vswitches, reqID, e := aliyunDescribeVSwitchesRich(auth.SecretID, auth.SecretKey, regionID, vpcID, zoneID)
			items, rid, err = vswitches, reqID, e
		case "security_group":
			sgs, reqID, e := aliyunDescribeSecurityGroupsRich(auth.SecretID, auth.SecretKey, regionID, vpcID)
			items, rid, err = sgs, reqID, e
		default:
			writeErrorJSON(w, r, 400, "不支持的网络资源类型: "+resourceType)
			return
		}
	}
	if err != nil {
		cloudQueryError(w, 500, err.Error())
		return
	}
	setCloudRequestIDHeader(w, rid)
	// Return bare array — frontend vpcs/vswitches/security-groups dropdowns expect JSON array directly
	writeJSON(w, 200, items)
}

// --- Regions & Images ---

func handleCloudRegions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, 405, "method not allowed")
		return
	}
	platformType := strings.TrimSpace(r.URL.Query().Get("platform_type"))
	if platformType == "" {
		cloudQueryError(w, 400, "缺少 platform_type 参数")
		return
	}
	tenantID := resolveGlobalCloudTenantID(r)
	if tenantID == "" {
		cloudQueryError(w, 400, "无法解析租户，请从租户页面访问或传入 tenant_id")
		return
	}
	if !ensureTenantMember(w, r, tenantID) {
		return
	}
	auth, err := loadFirstCloudAuthForCompany(tenantID, platformType)
	if err != nil {
		cloudQueryError(w, 500, err.Error())
		return
	}
	if auth == nil {
		cloudQueryError(w, 400, "未找到平台 "+platformType+" 的云平台授权，请先在云平台授权页面添加授权")
		return
	}
	var regions []map[string]string
	var rid string
	if useInMemoryCloud() {
		regions = mockCloudRegions()
	} else {
		regions, rid, err = aliyunDescribeRegions(auth.SecretID, auth.SecretKey)
		if err != nil {
			cloudQueryError(w, 500, err.Error())
			return
		}
	}
	setCloudRequestIDHeader(w, rid)
	writeJSON(w, 200, formatRegionsIDName(regions))
}

func handleCloudImages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, 405, "method not allowed")
		return
	}
	platformType := strings.TrimSpace(r.URL.Query().Get("platform_type"))
	if platformType == "" {
		platformType = "aliyun"
	}
	regionID := strings.TrimSpace(r.URL.Query().Get("region_id"))
	if regionID == "" {
		cloudQueryError(w, 400, "缺少 region_id 参数")
		return
	}
	tenantID := resolveGlobalCloudTenantID(r)
	if tenantID == "" {
		cloudQueryError(w, 400, "无法解析租户，请从租户页面访问或传入 tenant_id")
		return
	}
	if !ensureTenantMember(w, r, tenantID) {
		return
	}
	auth, err := loadFirstCloudAuthForCompany(tenantID, platformType)
	if err != nil {
		cloudQueryError(w, 500, err.Error())
		return
	}
	if auth == nil {
		cloudQueryError(w, 400, "未找到平台 "+platformType+" 的云平台授权，请先在云平台授权页面添加授权")
		return
	}

	var images []map[string]interface{}
	var rid string
	if useInMemoryCloud() {
		images = []map[string]interface{}{}
	} else {
		images, rid, err = aliyunDescribeImages(auth.SecretID, auth.SecretKey, regionID)
	}
	if err != nil {
		cloudQueryError(w, 500, err.Error())
		return
	}
	setCloudRequestIDHeader(w, rid)
	writeJSON(w, 200, map[string]interface{}{
		"images": images, "platform": platformType,
	})
}

func handleCloudServerImages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, 405, "method not allowed")
		return
	}
	platformType := strings.TrimSpace(r.URL.Query().Get("platform_type"))
	if platformType == "" {
		platformType = "aliyun"
	}
	regionID := strings.TrimSpace(r.URL.Query().Get("region_id"))
	if regionID == "" {
		cloudQueryError(w, 400, "缺少 region_id 参数")
		return
	}
	tenantID := resolveGlobalCloudTenantID(r)
	if tenantID == "" {
		cloudQueryError(w, 400, "无法解析租户，请从租户页面访问或传入 tenant_id")
		return
	}
	if !ensureTenantMember(w, r, tenantID) {
		return
	}
	auth, err := loadFirstCloudAuthForCompany(tenantID, platformType)
	if err != nil {
		cloudQueryError(w, 500, err.Error())
		return
	}
	if auth == nil {
		cloudQueryError(w, 400, "未找到平台 "+platformType+" 的云平台授权，请先在云平台授权页面添加授权")
		return
	}

	var images []map[string]interface{}
	var rid string
	if useInMemoryCloud() {
		images = []map[string]interface{}{}
	} else {
		images, rid, err = aliyunDescribeImages(auth.SecretID, auth.SecretKey, regionID)
	}
	if err != nil {
		cloudQueryError(w, 500, err.Error())
		return
	}
	setCloudRequestIDHeader(w, rid)
	writeJSON(w, 200, map[string]interface{}{
		"server_images": images, "platform": platformType,
	})
}

// --- Server Config Default ---

const serverConfigDefaultSelectCols = `id, company_id, authorization_id, platform_type, region, zone_id, vpc_id, vswitch_id,
		security_group_id, payment_type, bandwidth_charging_mode, bandwidth,
		cpu_cores, memory_gb, instance_type, system_disk_category, data_disk_category,
		io_optimized, spot_strategy,
		created_at, updated_at`

func mapServerConfigDefaultRow(
	id, cid, aid, pt, region, zoneID, vpc, vswitch, sg, payType, bwMode string,
	bw int,
	cpuCores, memoryGB, instanceType, systemDisk, dataDisk string,
	ioOptimized, spotStrategy string,
	ca, ua time.Time,
) map[string]interface{} {
	return map[string]interface{}{
		"id":                      id,
		"company_id":              cid,
		"authorization_id":        aid,
		"platform":                pt, // frontend expects "platform", not "platform_type"
		"platform_type":           pt,
		"region":                  region,
		"zone_id":                 zoneID,
		"vpc_id":                  vpc,
		"vswitch_id":              vswitch,
		"security_group_id":       sg,
		"payment_type":            payType,
		"bandwidth_charging_mode": bwMode,
		"bandwidth":               bw,
		"cpu_cores":               cpuCores,
		"memory_gb":               memoryGB,
		"instance_type":           instanceType,
		"system_disk_category":    systemDisk,
		"data_disk_category":      dataDisk,
		"io_optimized":            ioOptimized,
		"spot_strategy":           spotStrategy,
		"created_at":              ca,
		"updated_at":              ua,
	}
}

// normalizeIoOptimized 将 io_optimized 输入（'optimized'/'none' 或 boolean）归一化为
// available-instances 查询契约使用的 'optimized'/'none' 字符串。
func normalizeIoOptimized(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "optimized":
		return "optimized"
	case "0", "false", "no", "none":
		return "none"
	default:
		return "optimized"
	}
}

func handleServerConfigDefault(w http.ResponseWriter, r *http.Request, subParts []string) {
	tenantID := getAuthTenant(r)
	if !ensureTenantMember(w, r, tenantID) {
		return
	}
	// Extract authorization_id from sub-path: /server-config-default/{authorization_id}/
	authID := ""
	if len(subParts) > 0 {
		authID = strings.TrimSpace(strings.TrimSuffix(subParts[0], "/"))
	}

	switch r.Method {
	case http.MethodGet:
		if authID == "" {
			// List all default configs for the tenant (used by ProjectRunTemplatePanel)
			rows, err := db.Query(
				`SELECT `+serverConfigDefaultSelectCols+`
				 FROM cloud_server_config_defaults
				 WHERE company_id=?`, tenantID)
			if err != nil {
				logError("handleServerConfigDefault GET list: "+err.Error(), r.Header.Get("X-Trace-Id"))
				writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
					"status": "error", "message": "加载默认配置列表失败",
					"trace_id": traceIDFromRequest(r),
				})
				return
			}
			defer rows.Close()
			items := make([]map[string]interface{}, 0)
			for rows.Next() {
				var id, cid, aid, pt, region, zoneID, vpc, vswitch, sg, payType, bwMode string
				var bw int
				var cpuCores, memoryGB, instanceType, systemDisk, dataDisk string
				var ioOptimized, spotStrategy string
				var ca, ua time.Time
				if scanErr := rows.Scan(
					&id, &cid, &aid, &pt, &region, &zoneID, &vpc, &vswitch, &sg, &payType, &bwMode, &bw,
					&cpuCores, &memoryGB, &instanceType, &systemDisk, &dataDisk,
					&ioOptimized, &spotStrategy, &ca, &ua,
				); scanErr != nil {
					logError("handleServerConfigDefault GET list scan: "+scanErr.Error(), r.Header.Get("X-Trace-Id"))
					continue
				}
				items = append(items, mapServerConfigDefaultRow(
					id, cid, aid, pt, region, zoneID, vpc, vswitch, sg, payType, bwMode, bw,
					cpuCores, memoryGB, instanceType, systemDisk, dataDisk,
					ioOptimized, spotStrategy, ca, ua,
				))
			}
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"status": "success",
				"data":   items,
			})
			return
		}
		row := db.QueryRow(
			`SELECT `+serverConfigDefaultSelectCols+`
			 FROM cloud_server_config_defaults
			 WHERE company_id=? AND authorization_id=?`, tenantID, authID)
		var id, cid, aid, pt, region, zoneID, vpc, vswitch, sg, payType, bwMode string
		var bw int
		var cpuCores, memoryGB, instanceType, systemDisk, dataDisk string
		var ioOptimized, spotStrategy string
		var ca, ua time.Time
		if err := row.Scan(
			&id, &cid, &aid, &pt, &region, &zoneID, &vpc, &vswitch, &sg, &payType, &bwMode, &bw,
			&cpuCores, &memoryGB, &instanceType, &systemDisk, &dataDisk,
			&ioOptimized, &spotStrategy, &ca, &ua,
		); err != nil {
			if err == sql.ErrNoRows {
				writeJSON(w, http.StatusOK, map[string]interface{}{
					"status": "success", "data": nil, "message": "未找到默认配置",
				})
				return
			}
			logError("handleServerConfigDefault GET scan: "+err.Error(), r.Header.Get("X-Trace-Id"))
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"status": "error", "message": "加载默认配置失败",
				"trace_id": traceIDFromRequest(r),
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status": "success",
			"data": mapServerConfigDefaultRow(
				id, cid, aid, pt, region, zoneID, vpc, vswitch, sg, payType, bwMode, bw,
				cpuCores, memoryGB, instanceType, systemDisk, dataDisk,
				ioOptimized, spotStrategy, ca, ua,
			),
		})

	case http.MethodPost:
		body, err := readJSONBody(r)
		if err != nil {
			writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{
				"status": "error", "message": "无法解析请求体",
			})
			return
		}
		reqAuthID := strField(body, "authorization_id")
		if reqAuthID == "" {
			writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{
				"status": "error", "message": "缺少 authorization_id",
			})
			return
		}
		systemDisk := strField(body, "system_disk_category")
		if systemDisk == "" {
			systemDisk = "cloud_essd"
		}
		// data_disk_category 允许空字符串（表示不要数据盘）；仅缺省时回落默认盘型
		dataDisk := "cloud_essd"
		if raw, ok := body["data_disk_category"]; ok && raw != nil {
			dataDisk = strField(body, "data_disk_category")
		}
		// io_optimized / spot_strategy：实例筛选区已支持，默认配置表单一并持久化，
		// 使「快速应用默认模版」后 filter_options 与实例区对齐（OPT-20260812-020）
		ioOptimized := normalizeIoOptimized(strField(body, "io_optimized"))
		spotStrategy := strField(body, "spot_strategy")
		// Upsert: INSERT ... ON DUPLICATE KEY UPDATE
		id := genID("cscd")
		_, execErr := db.Exec(
			`INSERT INTO cloud_server_config_defaults
				(id, company_id, authorization_id, platform_type, region, zone_id, vpc_id, vswitch_id,
				 security_group_id, payment_type, bandwidth_charging_mode, bandwidth,
				 cpu_cores, memory_gb, instance_type, system_disk_category, data_disk_category,
				 io_optimized, spot_strategy)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			 ON DUPLICATE KEY UPDATE
				platform_type=VALUES(platform_type), region=VALUES(region), zone_id=VALUES(zone_id),
				vpc_id=VALUES(vpc_id), vswitch_id=VALUES(vswitch_id),
				security_group_id=VALUES(security_group_id), payment_type=VALUES(payment_type),
				bandwidth_charging_mode=VALUES(bandwidth_charging_mode), bandwidth=VALUES(bandwidth),
				cpu_cores=VALUES(cpu_cores), memory_gb=VALUES(memory_gb),
				instance_type=VALUES(instance_type),
				system_disk_category=VALUES(system_disk_category),
				data_disk_category=VALUES(data_disk_category),
				io_optimized=VALUES(io_optimized),
				spot_strategy=VALUES(spot_strategy),
				updated_at=CURRENT_TIMESTAMP`,
			id, tenantID, reqAuthID,
			strField(body, "platform_type"),
			strField(body, "region"),
			strField(body, "zone_id"),
			strField(body, "vpc_id"),
			strField(body, "vswitch_id"),
			strField(body, "security_group_id"),
			strField(body, "payment_type"),
			strField(body, "bandwidth_charging_mode"),
			intField(body, "bandwidth", 1),
			strField(body, "cpu_cores"),
			strField(body, "memory_gb"),
			strField(body, "instance_type"),
			systemDisk,
			dataDisk,
			ioOptimized,
			spotStrategy,
		)
		if execErr != nil {
			logError("handleServerConfigDefault POST insert: "+execErr.Error(), r.Header.Get("X-Trace-Id"))
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"status": "error", "message": "保存默认配置失败",
				"trace_id": traceIDFromRequest(r),
			})
			return
		}
		logInfo("cloud server config default saved: company="+tenantID+" auth="+reqAuthID+
			" instance_type="+strField(body, "instance_type")+
			" cpu="+strField(body, "cpu_cores")+" mem="+strField(body, "memory_gb"),
			r.Header.Get("X-Trace-Id"))
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status": "success", "message": "默认配置保存成功",
		})

	default:
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// --- UserData ---

func handleUserDataTemplates(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]interface{}{
		"templates": []interface{}{}, "message": "userdata templates — TODO",
	})
}

// --- AI Model Auths ---

func handleAIModelAuths(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]interface{}{
		"authorizations": []interface{}{}, "message": "ai model auths — TODO",
	})
}

// --- Start Server ---

func handleStartServer(w http.ResponseWriter, r *http.Request) {
	body, _ := readJSONBody(r)
	id := genID("csc")
	platform := strField(body, "platform")
	if platform == "" {
		platform = "aliyun"
	}
	db.Exec(`INSERT INTO cloud_server_configs(id,company_id,workspace_id,task_id,platform,region,zone_id,authorization_id) VALUES(?,?,?,?,?,?,?,?)`,
		id, getAuthTenant(r), strField(body, "workspace_id"), strField(body, "task_id"),
		platform, strField(body, "region"), strField(body, "zone_id"), strField(body, "authorization_id"))
	logInfo("server start requested: "+id, r.Header.Get("X-Trace-Id"))
	writeJSON(w, 202, map[string]interface{}{
		"config": map[string]string{"id": id}, "message": "instance creation queued — aliyun-sdk-go pending",
	})
}
