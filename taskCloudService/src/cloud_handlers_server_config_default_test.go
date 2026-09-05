package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"authz"
	"gatewayauth"
)

func TestHandleServerConfigDefaultPersistsHardwareFields(t *testing.T) {
	setupCloudTestDB(t)

	postBody := `{
		"authorization_id":"auth-hw-1",
		"platform_type":"aliyun",
		"region":"cn-hongkong",
		"zone_id":"cn-hongkong-d",
		"vpc_id":"vpc-hk",
		"vswitch_id":"vsw-hk",
		"security_group_id":"sg-hk",
		"payment_type":"PostPaid",
		"bandwidth_charging_mode":"PayByTraffic",
		"bandwidth":5,
		"cpu_cores":"2",
		"memory_gb":"8",
		"instance_type":"ecs.g6.large",
		"system_disk_category":"cloud_ssd",
		"data_disk_category":""
	}`
	postReq := httptest.NewRequest(http.MethodPost, "/api/cloud/server-config-default/tenant_id/t-hw/", strings.NewReader(postBody))
	postReq.Header.Set("Content-Type", "application/json")
	postReq.Header.Set(gatewayauth.HeaderAuthTenantID, "t-hw")
	postReq.Header.Set(gatewayauth.HeaderAuthUserID, "u-hw")
	postReq.Header.Set(authz.HeaderUserRoles, "super_admin")
	postRec := httptest.NewRecorder()
	handleServerConfigDefault(postRec, postReq, nil)
	if postRec.Code != 200 {
		t.Fatalf("POST status=%d body=%s", postRec.Code, postRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/cloud/server-config-default/tenant_id/t-hw/auth-hw-1/", nil)
	getReq.Header.Set(gatewayauth.HeaderAuthTenantID, "t-hw")
	getReq.Header.Set(gatewayauth.HeaderAuthUserID, "u-hw")
	getReq.Header.Set(authz.HeaderUserRoles, "super_admin")
	getRec := httptest.NewRecorder()
	handleServerConfigDefault(getRec, getReq, []string{"auth-hw-1"})
	if getRec.Code != 200 {
		t.Fatalf("GET status=%d body=%s", getRec.Code, getRec.Body.String())
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(getRec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	data, _ := payload["data"].(map[string]interface{})
	if data == nil {
		t.Fatalf("expected data object, got %#v", payload)
	}
	checks := map[string]string{
		"zone_id":              "cn-hongkong-d",
		"cpu_cores":            "2",
		"memory_gb":            "8",
		"instance_type":        "ecs.g6.large",
		"system_disk_category": "cloud_ssd",
		"data_disk_category":   "",
		"region":               "cn-hongkong",
	}
	for key, want := range checks {
		got := strings.TrimSpace(fmt.Sprint(data[key]))
		if got == "<nil>" {
			got = ""
		}
		if got != want {
			t.Fatalf("%s=%q want %q body=%v", key, got, want, data)
		}
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/cloud/server-config-default/tenant_id/t-hw/", nil)
	listReq.Header.Set(gatewayauth.HeaderAuthTenantID, "t-hw")
	listReq.Header.Set(gatewayauth.HeaderAuthUserID, "u-hw")
	listReq.Header.Set(authz.HeaderUserRoles, "super_admin")
	listRec := httptest.NewRecorder()
	handleServerConfigDefault(listRec, listReq, nil)
	if listRec.Code != 200 {
		t.Fatalf("LIST status=%d body=%s", listRec.Code, listRec.Body.String())
	}
	var listPayload map[string]interface{}
	if err := json.NewDecoder(listRec.Body).Decode(&listPayload); err != nil {
		t.Fatalf("list decode: %v", err)
	}
	items, _ := listPayload["data"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("list len=%d want 1 body=%s", len(items), listRec.Body.String())
	}
	item, _ := items[0].(map[string]interface{})
	if strings.TrimSpace(fmt.Sprint(item["instance_type"])) != "ecs.g6.large" {
		t.Fatalf("list missing instance_type: %#v", item)
	}
}

func TestHandleServerConfigDefaultPersistsBiddingFields(t *testing.T) {
	setupCloudTestDB(t)

	// POST 新配置：IoOptimized + 竞价策略
	postBody := `{
		"authorization_id":"auth-bid-1",
		"platform_type":"aliyun",
		"region":"cn-hongkong",
		"zone_id":"cn-hongkong-d",
		"vpc_id":"vpc-bid",
		"vswitch_id":"vsw-bid",
		"security_group_id":"sg-bid",
		"payment_type":"PostPaid",
		"cpu_cores":"2",
		"memory_gb":"8",
		"instance_type":"ecs.g6.large",
		"system_disk_category":"cloud_essd",
		"data_disk_category":"cloud_ssd",
		"io_optimized":"optimized",
		"spot_strategy":"SpotAsPriceGo"
	}`
	postReq := httptest.NewRequest(http.MethodPost, "/api/cloud/server-config-default/tenant_id/t-bid/", strings.NewReader(postBody))
	postReq.Header.Set("Content-Type", "application/json")
	postReq.Header.Set(gatewayauth.HeaderAuthTenantID, "t-bid")
	postReq.Header.Set(gatewayauth.HeaderAuthUserID, "u-bid")
	postReq.Header.Set(authz.HeaderUserRoles, "super_admin")
	postRec := httptest.NewRecorder()
	handleServerConfigDefault(postRec, postReq, nil)
	if postRec.Code != 200 {
		t.Fatalf("POST status=%d body=%s", postRec.Code, postRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/cloud/server-config-default/tenant_id/t-bid/auth-bid-1/", nil)
	getReq.Header.Set(gatewayauth.HeaderAuthTenantID, "t-bid")
	getReq.Header.Set(gatewayauth.HeaderAuthUserID, "u-bid")
	getReq.Header.Set(authz.HeaderUserRoles, "super_admin")
	getRec := httptest.NewRecorder()
	handleServerConfigDefault(getRec, getReq, []string{"auth-bid-1"})
	if getRec.Code != 200 {
		t.Fatalf("GET status=%d body=%s", getRec.Code, getRec.Body.String())
	}
	var payload map[string]interface{}
	if err := json.NewDecoder(getRec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	data, _ := payload["data"].(map[string]interface{})
	if data == nil {
		t.Fatalf("expected data object, got %#v", payload)
	}
	for key, want := range map[string]string{
		"io_optimized":  "optimized",
		"spot_strategy": "SpotAsPriceGo",
	} {
		got := strings.TrimSpace(fmt.Sprint(data[key]))
		if got == "<nil>" {
			got = ""
		}
		if got != want {
			t.Fatalf("%s=%q want %q body=%v", key, got, want, data)
		}
	}

	// Upsert 更新：io_optimized=none（boolean true/false 形态也应兼容）+ spot_strategy 变更
	updateBody := `{
		"authorization_id":"auth-bid-1",
		"platform_type":"aliyun",
		"region":"cn-hongkong",
		"zone_id":"cn-hongkong-d",
		"vpc_id":"vpc-bid",
		"vswitch_id":"vsw-bid",
		"security_group_id":"sg-bid",
		"payment_type":"PostPaid",
		"io_optimized":false,
		"spot_strategy":"NoSpot"
	}`
	updReq := httptest.NewRequest(http.MethodPost, "/api/cloud/server-config-default/tenant_id/t-bid/", strings.NewReader(updateBody))
	updReq.Header.Set("Content-Type", "application/json")
	updReq.Header.Set(gatewayauth.HeaderAuthTenantID, "t-bid")
	updReq.Header.Set(gatewayauth.HeaderAuthUserID, "u-bid")
	updReq.Header.Set(authz.HeaderUserRoles, "super_admin")
	updRec := httptest.NewRecorder()
	handleServerConfigDefault(updRec, updReq, nil)
	if updRec.Code != 200 {
		t.Fatalf("UPDATE status=%d body=%s", updRec.Code, updRec.Body.String())
	}

	getReq2 := httptest.NewRequest(http.MethodGet, "/api/cloud/server-config-default/tenant_id/t-bid/auth-bid-1/", nil)
	getReq2.Header.Set(gatewayauth.HeaderAuthTenantID, "t-bid")
	getReq2.Header.Set(gatewayauth.HeaderAuthUserID, "u-bid")
	getReq2.Header.Set(authz.HeaderUserRoles, "super_admin")
	getRec2 := httptest.NewRecorder()
	handleServerConfigDefault(getRec2, getReq2, []string{"auth-bid-1"})
	var payload2 map[string]interface{}
	if err := json.NewDecoder(getRec2.Body).Decode(&payload2); err != nil {
		t.Fatalf("decode: %v", err)
	}
	data2, _ := payload2["data"].(map[string]interface{})
	if got := strings.TrimSpace(fmt.Sprint(data2["io_optimized"])); got != "none" {
		t.Fatalf("after update io_optimized=%q want none body=%v", got, data2)
	}
	if got := strings.TrimSpace(fmt.Sprint(data2["spot_strategy"])); got != "NoSpot" {
		t.Fatalf("after update spot_strategy=%q want NoSpot body=%v", got, data2)
	}
}
