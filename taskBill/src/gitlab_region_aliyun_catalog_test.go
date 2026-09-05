package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNormalizeGitlabRegionInfraStatus(t *testing.T) {
	if NormalizeGitlabRegionInfraStatus("") != gitlabRegionInfraReady {
		t.Fatal("empty must be ready")
	}
	if NormalizeGitlabRegionInfraStatus("pending_node") != gitlabRegionInfraPendingNode {
		t.Fatal("pending_node")
	}
}

func TestEnsureTenantGitlabGroupForRegion_PendingNodeNoHTTP(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	err := ensureTenantGitlabGroupForRegion(t.Context(), 1, 1024, &GitlabRegion{
		Slug:              "aliyun-cn-hangzhou",
		GitlabAPIBase:     srv.URL,
		AdminPrivateToken: "pat",
		InfraStatus:       gitlabRegionInfraPendingNode,
	})
	if err != errGitlabRegionInfraPending {
		t.Fatalf("want pending err, got %v", err)
	}
	if hits != 0 {
		t.Fatalf("hits=%d want 0", hits)
	}
}

func TestGitlabRegionsListIncludesAliyunPendingNode(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	req := httptest.NewRequest(http.MethodGet, "/api/billing/gitlab-regions/", nil)
	rec := httptest.NewRecorder()
	handleGitlabRegionsList(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	if !regionListHasSlug(t, rec.Body.Bytes(), "aliyun-cn-hangzhou") {
		t.Fatal("aliyun-cn-hangzhou must be sellable")
	}
	var payload struct {
		Regions []struct {
			Slug        string `json:"slug"`
			InfraStatus string `json:"infra_status"`
			Cloud       string `json:"cloud_provider"`
			Token       string `json:"admin_private_token"`
		} `json:"regions"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range payload.Regions {
		if r.Slug == "aliyun-cn-hangzhou" {
			found = true
			if r.InfraStatus != gitlabRegionInfraPendingNode {
				t.Fatalf("infra_status=%q", r.InfraStatus)
			}
			if r.Cloud != "aliyun" {
				t.Fatalf("cloud=%q", r.Cloud)
			}
			if r.Token != "" {
				t.Fatal("token must be stripped")
			}
		}
	}
	if !found {
		t.Fatal("missing hangzhou in parsed list")
	}
}

func TestAdminProvisionPendingNodeConflict(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	reg, err := getGitlabRegionBySlug("aliyun-cn-hangzhou")
	if err != nil {
		t.Fatal(err)
	}
	if !gitlabRegionIsPendingNode(reg) {
		t.Fatal("seed must be pending_node")
	}
	err = ensureTenantGitlabGroupForRegion(t.Context(), 1, 1024, reg)
	if !errors.Is(err, errGitlabRegionInfraPending) {
		t.Fatalf("want pending, got %v", err)
	}
}

func TestParseGitlabRegionInfraStatusRejectsUnknown(t *testing.T) {
	if _, err := ParseGitlabRegionInfraStatus("deploying"); err == nil {
		t.Fatal("want error")
	}
}

func TestCreateOrderGitlabDiskAliyunPendingNode(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9300000202
	seedVIP1Membership(t, tenantID)
	order, items, err := createOrder(context.Background(), tenantID, []orderItemInput{
		{ResourceType: ResourceTypeGitlabDisk, Quantity: GitlabDiskMinPurchaseGB, Region: "aliyun-cn-hangzhou", DiskMonths: 1},
	})
	if err != nil {
		t.Fatalf("createOrder: %v", err)
	}
	if order == nil || len(items) != 1 || items[0].Region != "aliyun-cn-hangzhou" {
		t.Fatalf("items=%+v", items)
	}
}

func TestMarkOrderPaidAliyunPendingNodeQueuesFulfillment(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9300000201
	seedVIP1Membership(t, tenantID)
	orig := gitlabRegionEventPublisher
	t.Cleanup(func() { gitlabRegionEventPublisher = orig })
	var gotType, gotKey string
	gitlabRegionEventPublisher = func(_ context.Context, eventType string, payload map[string]interface{}, key string) error {
		if eventType == "GitlabManualNodeFulfillmentQueued" {
			gotType = eventType
			gotKey = key
			if payload["region_slug"] != "aliyun-cn-hangzhou" {
				t.Errorf("region_slug=%v", payload["region_slug"])
			}
			if payload["cloud_provider"] != "aliyun" {
				t.Errorf("cloud_provider=%v", payload["cloud_provider"])
			}
		}
		return nil
	}
	order, _, err := createOrder(context.Background(), tenantID, []orderItemInput{
		{ResourceType: ResourceTypeGitlabDisk, Quantity: GitlabDiskMinPurchaseGB, Region: "aliyun-cn-hangzhou", DiskMonths: 1},
	})
	if err != nil {
		t.Fatalf("createOrder: %v", err)
	}
	if err := markOrderPaid(context.Background(), order.ID, "wechat", "mock-aliyun-ref", tenantID); err != nil {
		t.Fatalf("markOrderPaid: %v", err)
	}
	var status string
	if err := db.QueryRow(
		`SELECT provisioning_status FROM billing_tenant_gitlab_resource WHERE tenant_id = ? AND region = ?`,
		tenantID, "aliyun-cn-hangzhou",
	).Scan(&status); err != nil {
		t.Fatalf("scan resource: %v", err)
	}
	if status != "pending_admin" {
		t.Fatalf("provisioning_status=%q", status)
	}
	if gotType != "GitlabManualNodeFulfillmentQueued" {
		t.Fatalf("event=%q", gotType)
	}
	if gotKey != formatID(order.ID) {
		t.Fatalf("key=%q want %s", gotKey, formatID(order.ID))
	}
}

func TestAdminPutInfraStatusReady(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	orig := gitlabRegionEventPublisher
	t.Cleanup(func() { gitlabRegionEventPublisher = orig })
	var gotType, gotKey string
	gitlabRegionEventPublisher = func(_ context.Context, eventType string, _ map[string]interface{}, key string) error {
		if eventType == "GitlabRegionInfraMarkedReady" {
			gotType = eventType
			gotKey = key
		}
		return nil
	}
	body, _ := json.Marshal(map[string]any{"infra_status": "ready"})
	req := httptest.NewRequest(http.MethodPut, "/api/system_admin/gitlab-regions/aliyun-cn-hangzhou/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleSystemAdminGitlabRegions(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	reg, err := getAnyGitlabRegionBySlug("aliyun-cn-hangzhou")
	if err != nil {
		t.Fatal(err)
	}
	if NormalizeGitlabRegionInfraStatus(reg.InfraStatus) != gitlabRegionInfraReady {
		t.Fatalf("infra=%q", reg.InfraStatus)
	}
	if gotType != "GitlabRegionInfraMarkedReady" {
		t.Fatalf("event=%q", gotType)
	}
	if gotKey != "aliyun-cn-hangzhou" {
		t.Fatalf("key=%q", gotKey)
	}
}

func TestAdminPutInfraStatusInvalid(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	body, _ := json.Marshal(map[string]any{"infra_status": "deploying"})
	req := httptest.NewRequest(http.MethodPut, "/api/system_admin/gitlab-regions/aliyun-cn-hangzhou/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleSystemAdminGitlabRegions(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestBillingEventTopicGitlabManualNodeFulfillmentQueued(t *testing.T) {
	if billingEventTopics["GitlabManualNodeFulfillmentQueued"] != "gitlab-manual-node-fulfillment-queued" {
		t.Fatalf("topic=%q", billingEventTopics["GitlabManualNodeFulfillmentQueued"])
	}
	if billingEventTopics["GitlabRegionInfraMarkedReady"] != "gitlab-region-infra-marked-ready" {
		t.Fatalf("ready topic=%q", billingEventTopics["GitlabRegionInfraMarkedReady"])
	}
}
