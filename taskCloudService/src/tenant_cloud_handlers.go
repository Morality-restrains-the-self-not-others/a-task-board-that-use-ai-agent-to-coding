package main

import (
	"net/http"
	"strconv"
	"strings"
)

func handleCloudPlatformRoutes(w http.ResponseWriter, r *http.Request, tenantID, authID, subPath string) {
	if !ensureTenantMember(w, r, tenantID) {
		return
	}
	auth, err := loadCloudAuth(tenantID, authID)
	if err != nil {
		cloudQueryError(w, 500, err.Error())
		return
	}
	if auth == nil {
		writeCloudAuthNotFound(w)
		return
	}
	sub := strings.Trim(strings.TrimPrefix(subPath, "/"), "/")
	switch sub {
	case "regions", "regions/":
		handleTenantCloudRegions(w, r, auth)
	case "zones", "zones/":
		handleTenantCloudZones(w, r, auth)
	case "bandwidth-limitation", "bandwidth-limitation/":
		handleTenantCloudBandwidth(w, r, auth)
	case "available-instances", "available-instances/":
		handleTenantCloudAvailableInstances(w, r, tenantID, auth)
	case "instance-price", "instance-price/":
		handleTenantCloudInstancePrice(w, r, auth)
	case "instance-details", "instance-details/":
		handleTenantCloudInstanceDetails(w, r, auth)
	case "debug/describe-image-support-instance-types", "debug/describe-image-support-instance-types/":
		handleTenantCloudDebugImageSupport(w, r, auth)
	default:
		writeErrorMapJSON(w, r, 404, map[string]interface{}{"status": "error", "message": "route not found"})
	}
}

func handleTenantCloudRegions(w http.ResponseWriter, r *http.Request, auth *cloudAuthRecord) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, 405, "method not allowed")
		return
	}
	if !validatePlatformTypeQuery(w, r, auth) {
		return
	}
	var regions []map[string]string
	var rid string
	var err error
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

func handleTenantCloudZones(w http.ResponseWriter, r *http.Request, auth *cloudAuthRecord) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, 405, "method not allowed")
		return
	}
	regionID := r.URL.Query().Get("region_id")
	if regionID == "" {
		cloudQueryError(w, 400, "缺少 region_id 参数")
		return
	}
	var zones []map[string]string
	var rid string
	var err error
	if useInMemoryCloud() {
		zones = mockCloudZones(regionID)
	} else {
		zones, rid, err = aliyunDescribeZones(auth.SecretID, auth.SecretKey, regionID)
		if err != nil {
			cloudQueryError(w, 500, err.Error())
			return
		}
	}
	setCloudRequestIDHeader(w, rid)
	writeJSON(w, 200, formatZonesIDName(zones))
}

func handleTenantCloudBandwidth(w http.ResponseWriter, r *http.Request, auth *cloudAuthRecord) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, 405, "method not allowed")
		return
	}
	if !validatePlatformTypeQuery(w, r, auth) {
		return
	}
	regionID := r.URL.Query().Get("region_id")
	instanceType := r.URL.Query().Get("instance_type")
	if regionID == "" || instanceType == "" {
		cloudQueryError(w, 400, "缺少 region_id 或 instance_type 参数")
		return
	}
	platformType := r.URL.Query().Get("platform_type")
	if platformType == "" {
		platformType = auth.PlatformType
	}
	if platformType != "aliyun" {
		cloudQueryError(w, 400, "当前云平台 "+platformType+" 不支持带宽限制查询")
		return
	}
	var data map[string]interface{}
	var rid string
	var err error
	if useInMemoryCloud() {
		data = mockBandwidthLimitation()
		rid, _ = data["RequestId"].(string)
	} else {
		data, rid, err = aliyunDescribeBandwidth(auth.SecretID, auth.SecretKey, regionID, instanceType)
		if err != nil {
			cloudQueryError(w, 500, err.Error())
			return
		}
	}
	if rid != "" {
		w.Header().Set("X-Cloud-Request-Id-get_bandwidth_limitation", rid)
	}
	writeJSON(w, 200, data)
}

func handleTenantCloudAvailableInstances(w http.ResponseWriter, r *http.Request, tenantID string, auth *cloudAuthRecord) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, 405, "method not allowed")
		return
	}
	if !validatePlatformTypeQuery(w, r, auth) {
		return
	}
	regionID := r.URL.Query().Get("region_id")
	if r.URL.Query().Get("platform_type") == "" && auth.PlatformType == "" {
		cloudQueryError(w, 400, "缺少 platform_type 或 region_id 参数")
		return
	}
	if regionID == "" {
		cloudQueryError(w, 400, "缺少 platform_type 或 region_id 参数")
		return
	}
	zoneID := r.URL.Query().Get("zone_id")
	filters := parseAvailableInstancesFilters(r, regionID, zoneID)
	platformType := strings.TrimSpace(r.URL.Query().Get("platform_type"))
	if platformType == "" {
		platformType = auth.PlatformType
	}
	if filters.CloudImageID == "" {
		containerImageID := strings.TrimSpace(r.URL.Query().Get("container_image_id"))
		if containerImageID != "" {
			filters.CloudImageID = resolveCloudImageIDForInstanceQuery(tenantID, containerImageID, platformType, regionID)
		}
	}
	var payload interface{}
	var rid string
	var err error
	if useInMemoryCloud() {
		payload = mockAvailableInstances()
	} else {
		payload, rid, err = fetchAvailableInstances(auth.SecretID, auth.SecretKey, filters)
		if err != nil {
			cloudQueryError(w, 500, err.Error())
			return
		}
	}
	setCloudRequestIDHeader(w, rid)
	writeJSON(w, 200, payload)
}

func handleTenantCloudInstancePrice(w http.ResponseWriter, r *http.Request, auth *cloudAuthRecord) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, 405, "method not allowed")
		return
	}
	if !validatePlatformTypeQuery(w, r, auth) {
		return
	}
	regionID := r.URL.Query().Get("region_id")
	instanceType := r.URL.Query().Get("instance_type")
	diskCategory := r.URL.Query().Get("system_disk_category")
	if regionID == "" || instanceType == "" {
		cloudQueryError(w, 400, "缺少必要参数")
		return
	}
	storageGB, _ := strconv.Atoi(defaultQuery(r, "storage_gb", "40"))
	bandwidth, _ := strconv.Atoi(defaultQuery(r, "bandwidth", "0"))
	spotStrategy := defaultQuery(r, "spot_strategy", "NoSpot")
	var data map[string]interface{}
	var rid string
	var err error
	if useInMemoryCloud() {
		data = mockInstancePrice()
	} else {
		data, rid, err = aliyunDescribePrice(auth.SecretID, auth.SecretKey, regionID, instanceType, diskCategory, storageGB, bandwidth, spotStrategy)
		if err != nil {
			cloudQueryError(w, 500, err.Error())
			return
		}
	}
	setCloudRequestIDHeader(w, rid)
	writeJSON(w, 200, data)
}

func handleTenantCloudInstanceDetails(w http.ResponseWriter, r *http.Request, auth *cloudAuthRecord) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, 405, "method not allowed")
		return
	}
	regionID := r.URL.Query().Get("region_id")
	if regionID == "" {
		cloudQueryError(w, 400, "缺少 region_id 参数")
		return
	}
	if !validatePlatformTypeQuery(w, r, auth) {
		return
	}
	raw := r.URL.Query().Get("instance_types")
	ids := []string{}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			ids = append(ids, part)
		}
	}
	if len(ids) == 0 {
		writeJSON(w, 200, []interface{}{})
		return
	}
	if len(ids) > 10 {
		ids = ids[:10]
	}
	var details []map[string]interface{}
	var err error
	if useInMemoryCloud() {
		details = mockInstanceDetails(ids)
	} else {
		details, err = aliyunDescribeInstanceTypes(auth.SecretID, auth.SecretKey, regionID, ids)
		if err != nil {
			cloudQueryError(w, 500, err.Error())
			return
		}
	}
	writeJSON(w, 200, details)
}

func handleTenantCloudDebugImageSupport(w http.ResponseWriter, r *http.Request, auth *cloudAuthRecord) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, 405, "method not allowed")
		return
	}
	if !validatePlatformTypeQuery(w, r, auth) {
		return
	}
	regionID := r.URL.Query().Get("region_id")
	imageID := r.URL.Query().Get("image_id")
	platformType := r.URL.Query().Get("platform_type")
	if platformType == "" {
		platformType = auth.PlatformType
	}
	if regionID == "" {
		cloudQueryError(w, 400, "缺少 region_id")
		return
	}
	if platformType != "aliyun" {
		cloudQueryError(w, 400, "当前仅支持 platform_type=aliyun")
		return
	}
	if imageID == "" {
		cloudQueryError(w, 400, "缺少 image_id，或提供 container_image_id 且无法解析为云镜像 ID")
		return
	}
	var payload map[string]interface{}
	var rid string
	var err error
	if useInMemoryCloud() {
		payload = mockImageSupportInstanceTypes()
		payload["region_id"] = regionID
		payload["image_id"] = imageID
		rid, _ = payload["request_id"].(string)
	} else {
		payload, rid, err = aliyunDescribeImageSupportInstanceTypes(auth.SecretID, auth.SecretKey, regionID, imageID)
		if err != nil {
			cloudQueryError(w, 500, err.Error())
			return
		}
	}
	setCloudRequestIDHeader(w, rid)
	writeJSON(w, 200, payload)
}

func defaultQuery(r *http.Request, key, fallback string) string {
	v := strings.TrimSpace(r.URL.Query().Get(key))
	if v == "" {
		return fallback
	}
	return v
}
