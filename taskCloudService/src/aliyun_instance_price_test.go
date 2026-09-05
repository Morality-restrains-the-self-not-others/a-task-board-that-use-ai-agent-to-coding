package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ecsclient "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/dara"
)

func TestIsUnsupportedSystemDiskCategoryErr(t *testing.T) {
	sdkErr := errors.New(`SDKError:
   StatusCode: 403
   Code: InvalidSystemDiskCategory.ValueNotSupported
   Message: code: 403, The specified parameter  SystemDisk.Category is not valid.`)
	if !isUnsupportedSystemDiskCategoryErr(sdkErr) {
		t.Fatalf("expected unsupported disk category error to match")
	}
	if isUnsupportedSystemDiskCategoryErr(errors.New("network timeout")) {
		t.Fatalf("expected unrelated error to not match")
	}
}

func TestDiskCategoriesForDescribePrice(t *testing.T) {
	cats := diskCategoriesForDescribePrice("cloud_efficiency")
	if len(cats) != 5 || cats[0] != "cloud_efficiency" || cats[1] != "cloud_essd" {
		t.Fatalf("unexpected category order: %#v", cats)
	}
	emptyCats := diskCategoriesForDescribePrice("")
	if len(emptyCats) != 6 || emptyCats[0] != "" {
		t.Fatalf("expected empty category first when preferred is blank: %#v", emptyCats)
	}
}

func TestBuildDescribePriceRequestSetsIoOptimized(t *testing.T) {
	req := buildDescribePriceRequest("cn-hongkong", "ecs.c7a.large", "cloud_efficiency", 40, 5, "SpotAsPriceGo")
	if req.IoOptimized == nil || *req.IoOptimized != "optimized" {
		t.Fatalf("expected IoOptimized=optimized, got %+v", req.IoOptimized)
	}
	if req.SystemDisk == nil || req.SystemDisk.Category == nil || *req.SystemDisk.Category != "cloud_efficiency" {
		t.Fatalf("expected system disk category cloud_efficiency, got %+v", req.SystemDisk)
	}
	if req.SpotStrategy == nil || *req.SpotStrategy != "SpotAsPriceGo" {
		t.Fatalf("expected SpotAsPriceGo, got %+v", req.SpotStrategy)
	}
}

func TestBuildDescribePriceRequestWithoutSystemDisk(t *testing.T) {
	req := buildDescribePriceRequest("cn-hongkong", "ecs.c7a.large", "", 40, 5, "NoSpot")
	if req.SystemDisk != nil {
		t.Fatalf("expected no system disk when category is blank, got %+v", req.SystemDisk)
	}
}

func TestParseDescribePriceResponse(t *testing.T) {
	resp := &ecsclient.DescribePriceResponse{
		Body: &ecsclient.DescribePriceResponseBody{
			RequestId: dara.String("req-123"),
			PriceInfo: &ecsclient.DescribePriceResponseBodyPriceInfo{
				Price: &ecsclient.DescribePriceResponseBodyPriceInfoPrice{
					OriginalPrice: dara.Float32(1.23),
					TradePrice:    dara.Float32(0.98),
					DiscountPrice: dara.Float32(0.25),
					Currency:      dara.String("CNY"),
					DetailInfos: &ecsclient.DescribePriceResponseBodyPriceInfoPriceDetailInfos{
						DetailInfo: []*ecsclient.DescribePriceResponseBodyPriceInfoPriceDetailInfosDetailInfo{
							{
								Resource:      dara.String("instanceType"),
								TradePrice:    dara.Float32(0.6),
								OriginalPrice: dara.Float32(0.75),
							},
						},
					},
				},
			},
		},
	}
	result, rid := parseDescribePriceResponse(resp, "cloud_essd")
	if rid != "req-123" {
		t.Fatalf("expected request id req-123, got %q", rid)
	}
	if result["system_disk_category"] != "cloud_essd" {
		t.Fatalf("expected system_disk_category in result: %#v", result)
	}
	if result["price"] != float32(0.98) {
		t.Fatalf("expected trade price 0.98, got %#v", result["price"])
	}
	priceObj, ok := result["Price"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected nested Price object: %#v", result)
	}
	if priceObj["TradePrice"] != float32(0.98) {
		t.Fatalf("expected TradePrice 0.98, got %#v", priceObj["TradePrice"])
	}
	detailInfos, ok := priceObj["DetailInfos"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected DetailInfos in Price: %#v", priceObj)
	}
	detailInfo, ok := detailInfos["DetailInfo"].([]map[string]interface{})
	if !ok || len(detailInfo) != 1 {
		t.Fatalf("expected one detail entry: %#v", detailInfos)
	}
}

func TestTenantCloudInstancePriceMock(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudAuth(t, "850256677331562496", "862031128628060160")
	t.Setenv("USE_IN_MEMORY_CLOUD", "true")

	req := httptest.NewRequest(http.MethodGet,
		"/api/tenant/850256677331562496/cloud-platform/862031128628060160/cloud/instance-price/?platform_type=aliyun&region_id=cn-hongkong&instance_type=ecs.c7a.large&system_disk_category=cloud_efficiency&storage_gb=40&bandwidth=5&spot_strategy=SpotAsPriceGo",
		nil)
	req.Header.Set("X-Auth-Tenant-Id", "850256677331562496")
	req.Header.Set("X-User-Id", "test-user")
	rec := httptest.NewRecorder()

	handleCloudPlatformRoutes(rec, req, "850256677331562496", "862031128628060160", "instance-price/")

	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"price"`) {
		t.Fatalf("expected price in mock response: %s", body)
	}
	if !strings.Contains(body, `"DetailInfo"`) {
		t.Fatalf("expected price detail breakdown in mock response: %s", body)
	}
}
