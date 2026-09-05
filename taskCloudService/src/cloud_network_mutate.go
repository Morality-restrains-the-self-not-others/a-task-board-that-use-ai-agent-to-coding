package main

import (
	"net/http"
	"strings"
)

func handleCloudOccupiedCidrBlocks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, 405, "method not allowed")
		return
	}
	platformType := strings.TrimSpace(r.URL.Query().Get("platform_type"))
	if platformType == "" {
		platformType = "aliyun"
	}
	regionID := strings.TrimSpace(r.URL.Query().Get("region"))
	if regionID == "" {
		cloudQueryError(w, 400, "缺少 region 参数")
		return
	}
	authID := strings.TrimSpace(r.URL.Query().Get("authorization_id"))
	if authID == "" {
		cloudQueryError(w, 400, "缺少 authorization_id 参数")
		return
	}
	auth, err := loadCloudAuthByID(authID)
	if err != nil {
		cloudQueryError(w, 500, err.Error())
		return
	}
	if auth == nil {
		cloudQueryError(w, 404, "未找到云平台授权")
		return
	}

	var cidrBlocks []string
	if useInMemoryCloud() {
		cidrBlocks = []string{}
	} else {
		vpcs, _, err := aliyunDescribeVpcsRich(auth.SecretID, auth.SecretKey, regionID)
		if err != nil {
			cloudQueryError(w, 500, err.Error())
			return
		}
		for _, vpc := range vpcs {
			if vpc.CidrBlock != "" {
				cidrBlocks = append(cidrBlocks, vpc.CidrBlock)
			}
		}
	}
	writeJSON(w, 200, map[string]interface{}{
		"status":               "success",
		"occupied_cidr_blocks": cidrBlocks,
		"platform":             platformType,
	})
}

// --- VPC CRUD ---

func handleCloudCreateVpc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, 405, "method not allowed")
		return
	}
	body, _ := readJSONBody(r)
	name := strField(body, "name")
	regionID := strField(body, "region")
	cidrBlock := strField(body, "cidr_block")
	authID := strField(body, "authorization_id")
	if name == "" || regionID == "" || cidrBlock == "" || authID == "" {
		cloudQueryError(w, 400, "缺少必填参数: name, region, cidr_block, authorization_id")
		return
	}
	auth, err := loadCloudAuthByID(authID)
	if err != nil {
		cloudQueryError(w, 500, err.Error())
		return
	}
	if auth == nil {
		cloudQueryError(w, 404, "未找到云平台授权")
		return
	}

	var vpcID, requestID string
	if useInMemoryCloud() {
		vpcID = "vpc-mock-" + genID("vpc")
	} else {
		vpcID, _, err = aliyunCreateVpc(auth.SecretID, auth.SecretKey, regionID, name, cidrBlock)
		if err != nil {
			cloudQueryError(w, 500, err.Error())
			return
		}
	}
	_ = requestID
	writeJSON(w, 201, map[string]interface{}{
		"status":  "success",
		"vpc_id":  vpcID,
		"message": "VPC创建成功",
	})
}

func handleCloudUpdateVpc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeErrorJSON(w, r, 405, "method not allowed")
		return
	}
	body, _ := readJSONBody(r)
	vpcID := strField(body, "vpc_id")
	name := strField(body, "name")
	authID := strField(body, "authorization_id")
	if vpcID == "" || authID == "" {
		cloudQueryError(w, 400, "缺少必填参数: vpc_id, authorization_id")
		return
	}
	auth, err := loadCloudAuthByID(authID)
	if err != nil {
		cloudQueryError(w, 500, err.Error())
		return
	}
	if auth == nil {
		cloudQueryError(w, 404, "未找到云平台授权")
		return
	}

	// Aliyun ModifyVpcAttribute: only supports name/description change, not CIDR
	if name != "" && !useInMemoryCloud() {
		err = aliyunModifyVpcAttribute(auth.SecretID, auth.SecretKey, strField(body, "region"), vpcID, name)
		if err != nil {
			cloudQueryError(w, 500, err.Error())
			return
		}
	}
	writeJSON(w, 200, map[string]interface{}{
		"status":  "success",
		"message": "VPC更新成功",
	})
}

// --- VSwitch CRUD ---

func handleCloudCreateVswitch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, 405, "method not allowed")
		return
	}
	body, _ := readJSONBody(r)
	name := strField(body, "name")
	zoneID := strField(body, "zone_id")
	cidrBlock := strField(body, "cidr_block")
	vpcID := strField(body, "vpc_id")
	authID := strField(body, "authorization_id")
	regionID := strField(body, "region")
	if name == "" || zoneID == "" || vpcID == "" || authID == "" || regionID == "" {
		cloudQueryError(w, 400, "缺少必填参数: name, zone_id, vpc_id, authorization_id, region")
		return
	}
	auth, err := loadCloudAuthByID(authID)
	if err != nil {
		cloudQueryError(w, 500, err.Error())
		return
	}
	if auth == nil {
		cloudQueryError(w, 404, "未找到云平台授权")
		return
	}

	var vswitchID string
	if useInMemoryCloud() {
		vswitchID = "vsw-mock-" + genID("vsw")
	} else {
		vswitchID, err = aliyunCreateVSwitch(auth.SecretID, auth.SecretKey, regionID, zoneID, vpcID, name, cidrBlock)
		if err != nil {
			cloudQueryError(w, 500, err.Error())
			return
		}
	}
	writeJSON(w, 201, map[string]interface{}{
		"status":     "success",
		"vswitch_id": vswitchID,
		"message":    "交换机创建成功",
	})
}

func handleCloudUpdateVswitch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeErrorJSON(w, r, 405, "method not allowed")
		return
	}
	body, _ := readJSONBody(r)
	vswitchID := strField(body, "vswitch_id")
	name := strField(body, "name")
	authID := strField(body, "authorization_id")
	regionID := strField(body, "region")
	if vswitchID == "" || authID == "" {
		cloudQueryError(w, 400, "缺少必填参数: vswitch_id, authorization_id")
		return
	}
	auth, err := loadCloudAuthByID(authID)
	if err != nil {
		cloudQueryError(w, 500, err.Error())
		return
	}
	if auth == nil {
		cloudQueryError(w, 404, "未找到云平台授权")
		return
	}

	// Aliyun ModifyVSwitchAttribute: supports name change
	if name != "" && !useInMemoryCloud() {
		_ = regionID
		err = aliyunModifyVSwitchAttribute(auth.SecretID, auth.SecretKey, regionID, vswitchID, name)
		if err != nil {
			cloudQueryError(w, 500, err.Error())
			return
		}
	}
	writeJSON(w, 200, map[string]interface{}{
		"status":  "success",
		"message": "交换机更新成功",
	})
}

// --- Security Group CRUD ---

func handleCloudCreateSecurityGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, 405, "method not allowed")
		return
	}
	body, _ := readJSONBody(r)
	name := strField(body, "name")
	vpcID := strField(body, "vpc_id")
	authID := strField(body, "authorization_id")
	regionID := strField(body, "region")
	if name == "" || vpcID == "" || authID == "" || regionID == "" {
		cloudQueryError(w, 400, "缺少必填参数: name, vpc_id, authorization_id, region")
		return
	}
	auth, err := loadCloudAuthByID(authID)
	if err != nil {
		cloudQueryError(w, 500, err.Error())
		return
	}
	if auth == nil {
		cloudQueryError(w, 404, "未找到云平台授权")
		return
	}

	var sgID string
	if useInMemoryCloud() {
		sgID = "sg-mock-" + genID("sg")
	} else {
		sgID, err = aliyunCreateSecurityGroup(auth.SecretID, auth.SecretKey, regionID, vpcID, name)
		if err != nil {
			cloudQueryError(w, 500, err.Error())
			return
		}
	}
	writeJSON(w, 201, map[string]interface{}{
		"status":            "success",
		"security_group_id": sgID,
		"message":           "安全组创建成功",
	})
}

func handleCloudUpdateSecurityGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeErrorJSON(w, r, 405, "method not allowed")
		return
	}
	body, _ := readJSONBody(r)
	sgID := strField(body, "security_group_id")
	name := strField(body, "name")
	authID := strField(body, "authorization_id")
	regionID := strField(body, "region")
	if sgID == "" || authID == "" {
		cloudQueryError(w, 400, "缺少必填参数: security_group_id, authorization_id")
		return
	}
	auth, err := loadCloudAuthByID(authID)
	if err != nil {
		cloudQueryError(w, 500, err.Error())
		return
	}
	if auth == nil {
		cloudQueryError(w, 404, "未找到云平台授权")
		return
	}

	// Aliyun ModifySecurityGroupAttribute: supports name/description change
	if name != "" && !useInMemoryCloud() {
		_ = regionID
		err = aliyunModifySecurityGroupAttribute(auth.SecretID, auth.SecretKey, regionID, sgID, name)
		if err != nil {
			cloudQueryError(w, 500, err.Error())
			return
		}
	}
	writeJSON(w, 200, map[string]interface{}{
		"status":  "success",
		"message": "安全组更新成功",
	})
}
