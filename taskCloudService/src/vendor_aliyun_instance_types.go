package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	ecsclient "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/dara"
)

const (
	aliyunImageSupportTimeoutSec     = 50
	aliyunSkipStockIfMoreThan        = 40
	aliyunMaxInstanceTypesReturned   = 250
	aliyunInstanceStockCheckWorkers  = 10
	aliyunStockCheckDeadlineSec      = 56
)

type vendorInstanceTypeCandidate struct {
	InstanceTypeID     string
	CPUCoreCount       interface{}
	MemorySize         interface{}
	InstanceTypeFamily string
	Generation         string
}

func aliyunVendorInstanceTypes(accessKey, secretKey, regionID, imageID, zoneID string) ([]map[string]interface{}, string, map[string]interface{}, error) {
	imageID = strings.TrimSpace(imageID)
	if imageID == "" {
		return nil, "", nil, errors.New("请先指定云平台镜像 ID（image_id），以便按镜像与可用资源过滤实例规格")
	}
	useRegion := strings.TrimSpace(regionID)
	if useRegion == "" {
		useRegion = "cn-hangzhou"
	}
	zoneID = strings.TrimSpace(zoneID)

	meta := map[string]interface{}{
		"inventory_stock_checked":       false,
		"inventory_stock_skipped_reason": nil,
		"inventory_candidates_total":    0,
		"inventory_truncate_returned":   false,
	}

	client, err := newECSClient(accessKey, secretKey, useRegion)
	if err != nil {
		return nil, "", meta, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), aliyunImageSupportTimeoutSec*time.Second)
	defer cancel()

	type imgSupportResult struct {
		resp *ecsclient.DescribeImageSupportInstanceTypesResponse
		err  error
	}
	ch := make(chan imgSupportResult, 1)
	go func() {
		resp, callErr := client.DescribeImageSupportInstanceTypes(&ecsclient.DescribeImageSupportInstanceTypesRequest{
			RegionId: dara.String(useRegion),
			ImageId:  dara.String(imageID),
		})
		ch <- imgSupportResult{resp: resp, err: callErr}
	}()

	var imgResp *ecsclient.DescribeImageSupportInstanceTypesResponse
	select {
	case <-ctx.Done():
		return nil, "", meta, fmt.Errorf(
			"查询镜像支持的实例规格超时（DescribeImageSupportInstanceTypes），请检查网络、地域与镜像 ID 是否正确，或稍后重试",
		)
	case res := <-ch:
		if res.err != nil {
			return nil, "", meta, res.err
		}
		imgResp = res.resp
	}

	requestID := ""
	candidates := []vendorInstanceTypeCandidate{}
	if imgResp != nil && imgResp.Body != nil {
		requestID = derefString(imgResp.Body.RequestId)
		if imgResp.Body.InstanceTypes != nil {
			for _, it := range imgResp.Body.InstanceTypes.InstanceType {
				if it == nil || it.InstanceTypeId == nil {
					continue
				}
				iid := strings.TrimSpace(*it.InstanceTypeId)
				if iid == "" {
					continue
				}
				var mem interface{}
				if it.MemorySize != nil {
					mem = int(math.Round(float64(*it.MemorySize)))
				}
				var cpu interface{}
				if it.CpuCoreCount != nil {
					cpu = int(*it.CpuCoreCount)
				}
				candidates = append(candidates, vendorInstanceTypeCandidate{
					InstanceTypeID:     iid,
					CPUCoreCount:       cpu,
					MemorySize:         mem,
					InstanceTypeFamily: derefString(it.InstanceTypeFamily),
					Generation:         "",
				})
			}
		}
	}

	before := len(candidates)
	meta["inventory_candidates_total"] = before

	var result []vendorInstanceTypeCandidate
	if len(candidates) == 0 {
		result = []vendorInstanceTypeCandidate{}
	} else if before > aliyunSkipStockIfMoreThan {
		meta["inventory_stock_skipped_reason"] = fmt.Sprintf(
			"镜像支持的规格数量为 %d，超过单次库存校验上限 %d，已跳过 DescribeAvailableResource，仅返回镜像兼容规格。",
			before, aliyunSkipStockIfMoreThan,
		)
		result = candidates
	} else {
		meta["inventory_stock_checked"] = true
		inStock := aliyunFilterInstanceTypesByStock(accessKey, secretKey, useRegion, zoneID, candidates)
		result = make([]vendorInstanceTypeCandidate, 0, len(inStock))
		for _, c := range candidates {
			if inStock[c.InstanceTypeID] {
				result = append(result, c)
			}
		}
	}

	if len(result) > aliyunMaxInstanceTypesReturned {
		meta["inventory_truncate_returned"] = true
		meta["inventory_returned_before_truncation"] = len(result)
		result = result[:aliyunMaxInstanceTypesReturned]
	}

	out := make([]map[string]interface{}, 0, len(result))
	for _, c := range result {
		out = append(out, map[string]interface{}{
			"instance_type_id":     c.InstanceTypeID,
			"cpu_core_count":       c.CPUCoreCount,
			"memory_size":          c.MemorySize,
			"instance_type_family": c.InstanceTypeFamily,
			"generation":           c.Generation,
		})
	}
	return out, requestID, meta, nil
}

func aliyunFilterInstanceTypesByStock(accessKey, secretKey, regionID, zoneID string, candidates []vendorInstanceTypeCandidate) map[string]bool {
	inStock := map[string]bool{}
	if len(candidates) == 0 {
		return inStock
	}

	deadline := time.Now().Add(aliyunStockCheckDeadlineSec * time.Second)
	sem := make(chan struct{}, aliyunInstanceStockCheckWorkers)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, c := range candidates {
		if time.Now().After(deadline) {
			metaReason := "部分规格库存校验未在时限内完成，已按已完成部分过滤；未完成项视为无货。"
			_ = metaReason
			break
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(instanceTypeID string) {
			defer wg.Done()
			defer func() { <-sem }()
			ok := aliyunInstanceTypeInStock(accessKey, secretKey, regionID, instanceTypeID, zoneID)
			if ok {
				mu.Lock()
				inStock[instanceTypeID] = true
				mu.Unlock()
			}
		}(c.InstanceTypeID)
	}
	wg.Wait()
	return inStock
}

func aliyunInstanceTypeInStock(accessKey, secretKey, regionID, instanceTypeID, zoneID string) bool {
	client, err := newECSClient(accessKey, secretKey, regionID)
	if err != nil {
		return true
	}
	req := &ecsclient.DescribeAvailableResourceRequest{
		RegionId:            dara.String(regionID),
		DestinationResource: dara.String("InstanceType"),
		IoOptimized:         dara.String("optimized"),
		InstanceType:        dara.String(instanceTypeID),
	}
	if zoneID != "" {
		req.ZoneId = dara.String(zoneID)
	}
	resp, err := client.DescribeAvailableResource(req)
	if err != nil {
		return true
	}
	return aliyunStockFromDescribeAvailableBody(resp, instanceTypeID, zoneID)
}

func aliyunStockFromDescribeAvailableBody(resp *ecsclient.DescribeAvailableResourceResponse, instanceTypeID, zoneID string) bool {
	if resp == nil || resp.Body == nil || resp.Body.AvailableZones == nil {
		return false
	}
	for _, z := range resp.Body.AvailableZones.AvailableZone {
		if z == nil {
			continue
		}
		if zoneID != "" && derefString(z.ZoneId) != zoneID {
			continue
		}
		if derefString(z.Status) == "SoldOut" {
			continue
		}
		if z.AvailableResources == nil {
			continue
		}
		for _, ar := range z.AvailableResources.AvailableResource {
			if ar == nil || derefString(ar.Type) != "InstanceType" {
				continue
			}
			if ar.SupportedResources == nil {
				continue
			}
			for _, sr := range ar.SupportedResources.SupportedResource {
				if sr == nil || derefString(sr.Value) != instanceTypeID {
					continue
				}
				if aliyunSupportedResourceInStock(sr) {
					return true
				}
			}
		}
	}
	return false
}

func aliyunSupportedResourceInStock(sr *ecsclient.DescribeAvailableResourceResponseBodyAvailableZonesAvailableZoneAvailableResourcesAvailableResourceSupportedResourcesSupportedResource) bool {
	if sr == nil {
		return false
	}
	st := derefString(sr.Status)
	if st == "SoldOut" {
		return false
	}
	if st == "Available" {
		return true
	}
	sc := derefString(sr.StatusCategory)
	return sc == "WithStock" || sc == "ClosedWithStock"
}

func mockVendorInstanceTypes() ([]map[string]interface{}, map[string]interface{}) {
	types := []map[string]interface{}{
		{
			"instance_type_id":     "ecs.g6.large",
			"cpu_core_count":       2,
			"memory_size":          8,
			"instance_type_family": "ecs.g6",
			"generation":           "",
		},
		{
			"instance_type_id":     "ecs.t6-c1m2.large",
			"cpu_core_count":       2,
			"memory_size":          4,
			"instance_type_family": "ecs.t6",
			"generation":           "",
		},
	}
	meta := map[string]interface{}{
		"inventory_stock_checked":        false,
		"inventory_stock_skipped_reason": nil,
		"inventory_candidates_total":     len(types),
		"inventory_truncate_returned":    false,
	}
	return types, meta
}
