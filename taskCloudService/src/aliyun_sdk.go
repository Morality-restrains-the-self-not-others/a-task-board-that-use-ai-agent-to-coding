package main

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	ecsclient "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/dara"
)

const describeInstanceTypesBatchSize = 10

type availableInstancesFilters struct {
	RegionID            string
	ZoneID              string
	Cores               string
	Memory              string
	IoOptimized         string
	SystemDiskCategory  string
	DataDiskCategory    string
	NetworkCategory     string
	SpotStrategy        string
	InstanceChargeType  string
	SpotDuration        string
	ResourceType        string
	DestinationResource string
	ImageArchitecture   string
	CloudImageID        string
	InstanceFamily      string // OPT-20260722-035: fuzzy search by family prefix (e.g. "g6")
	Page                int
	PageSize            int
}

func parseAvailableInstancesFilters(r *http.Request, regionID, zoneID string) availableInstancesFilters {
	f := availableInstancesFilters{
		RegionID:            regionID,
		ZoneID:              zoneID,
		Cores:               strings.TrimSpace(r.URL.Query().Get("Cores")),
		Memory:              strings.TrimSpace(r.URL.Query().Get("Memory")),
		IoOptimized:         strings.TrimSpace(r.URL.Query().Get("IoOptimized")),
		SystemDiskCategory:  strings.TrimSpace(r.URL.Query().Get("SystemDiskCategory")),
		DataDiskCategory:    strings.TrimSpace(r.URL.Query().Get("DataDiskCategory")),
		NetworkCategory:     strings.TrimSpace(r.URL.Query().Get("NetworkCategory")),
		SpotStrategy:        defaultQuery(r, "spot_strategy", "SpotAsPriceGo"),
		ResourceType:        defaultQuery(r, "ResourceType", "instance"),
		DestinationResource: defaultQuery(r, "DestinationResource", "InstanceType"),
	}
	f.InstanceChargeType = strings.TrimSpace(r.URL.Query().Get("InstanceChargeType"))
	if f.InstanceChargeType == "" {
		f.InstanceChargeType = defaultQuery(r, "instance_charge_type", "PostPaid")
	}
	f.SpotDuration = strings.TrimSpace(r.URL.Query().Get("SpotDuration"))
	if f.SpotDuration == "" {
		f.SpotDuration = strings.TrimSpace(r.URL.Query().Get("spot_duration"))
	}
	f.ImageArchitecture = strings.TrimSpace(r.URL.Query().Get("image_architecture"))
	f.CloudImageID = strings.TrimSpace(r.URL.Query().Get("image_id"))
	f.InstanceFamily = strings.TrimSpace(r.URL.Query().Get("family"))
	f.Page, f.PageSize = parsePaginationQuery(r.URL.Query().Get("page"), r.URL.Query().Get("page_size"))
	return f
}

func applyAvailableInstancesFilters(req *ecsclient.DescribeAvailableResourceRequest, f availableInstancesFilters) {
	if f.Cores != "" {
		if cores, err := strconv.ParseInt(f.Cores, 10, 32); err == nil {
			req.Cores = dara.Int32(int32(cores))
		}
	}
	if f.Memory != "" {
		if mem, err := strconv.ParseFloat(f.Memory, 32); err == nil {
			req.Memory = dara.Float32(float32(mem))
		}
	}
	if f.IoOptimized != "" {
		req.IoOptimized = dara.String(f.IoOptimized)
	}
	if f.SystemDiskCategory != "" {
		req.SystemDiskCategory = dara.String(f.SystemDiskCategory)
	}
	if f.DataDiskCategory != "" {
		req.DataDiskCategory = dara.String(f.DataDiskCategory)
	}
	if f.NetworkCategory != "" {
		req.NetworkCategory = dara.String(f.NetworkCategory)
	}
	if f.SpotStrategy != "" {
		req.SpotStrategy = dara.String(f.SpotStrategy)
	}
	if f.InstanceChargeType != "" {
		req.InstanceChargeType = dara.String(f.InstanceChargeType)
	}
	if f.SpotDuration != "" {
		if sd, err := strconv.ParseInt(f.SpotDuration, 10, 32); err == nil {
			req.SpotDuration = dara.Int32(int32(sd))
		}
	} else {
		req.SpotDuration = dara.Int32(0)
	}
}

func newECSClient(accessKey, secretKey, regionID string) (*ecsclient.Client, error) {
	cfg := &openapiutil.Config{
		AccessKeyId:     dara.String(accessKey),
		AccessKeySecret: dara.String(secretKey),
		RegionId:        dara.String(regionID),
	}
	applyAliyunECSNetwork(cfg, regionID)
	return ecsclient.NewClient(cfg)
}

func aliyunDescribeRegions(accessKey, secretKey string) ([]map[string]string, string, error) {
	client, err := newECSClient(accessKey, secretKey, "cn-hangzhou")
	if err != nil {
		return nil, "", err
	}
	resp, err := client.DescribeRegions(&ecsclient.DescribeRegionsRequest{})
	if err != nil {
		return nil, "", err
	}
	rid := ""
	out := []map[string]string{}
	if resp.Body != nil {
		rid = derefString(resp.Body.RequestId)
		if resp.Body.Regions != nil {
			for _, reg := range resp.Body.Regions.Region {
				if reg == nil {
					continue
				}
				out = append(out, map[string]string{
					"region_id":   derefString(reg.RegionId),
					"region_name": derefString(reg.LocalName),
				})
			}
		}
	}
	return out, rid, nil
}

func aliyunDescribeImages(accessKey, secretKey, regionID string) ([]map[string]interface{}, string, error) {
	client, err := newECSClient(accessKey, secretKey, regionID)
	if err != nil {
		return nil, "", err
	}
	allImages := []map[string]interface{}{}
	pageNumber := int32(1)
	pageSize := int32(100)
	var lastRequestID string
	totalCount := 0

	for {
		resp, err := client.DescribeImages(&ecsclient.DescribeImagesRequest{
			RegionId:   dara.String(regionID),
			Status:     dara.String("Available"),
			PageNumber: dara.Int32(pageNumber),
			PageSize:   dara.Int32(pageSize),
		})
		if err != nil {
			return nil, lastRequestID, err
		}
		if resp.Body == nil {
			break
		}
		lastRequestID = derefString(resp.Body.RequestId)
		if resp.Body.TotalCount != nil {
			totalCount = int(*resp.Body.TotalCount)
		}
		if resp.Body.Images == nil {
			break
		}
		for _, img := range resp.Body.Images.Image {
			if img == nil {
				continue
			}
			imageType := derefString(img.ImageOwnerAlias)
			if imageType == "" {
				imageType = "unknown"
			}
			var sizeVal interface{}
			if img.Size != nil {
				sizeVal = int(*img.Size)
			}
			allImages = append(allImages, map[string]interface{}{
				"id":           derefString(img.ImageId),
				"name":         defaultString(derefString(img.ImageName), "未知名称"),
				"os_type":      defaultString(derefString(img.OSType), "unknown"),
				"os_version":   defaultString(derefString(img.OSNameEn), "unknown"),
				"architecture": derefString(img.Architecture),
				"image_type":   imageType,
				"size":         sizeVal,
			})
		}
		if totalCount == 0 || len(allImages) >= totalCount {
			break
		}
		pageNumber++
	}
	return allImages, lastRequestID, nil
}

func defaultString(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func aliyunDescribeZones(accessKey, secretKey, regionID string) ([]map[string]string, string, error) {
	client, err := newECSClient(accessKey, secretKey, regionID)
	if err != nil {
		return nil, "", err
	}
	resp, err := client.DescribeZones(&ecsclient.DescribeZonesRequest{RegionId: dara.String(regionID)})
	if err != nil {
		return nil, "", err
	}
	rid := ""
	out := []map[string]string{}
	if resp.Body != nil {
		rid = derefString(resp.Body.RequestId)
		if resp.Body.Zones != nil {
			for _, z := range resp.Body.Zones.Zone {
				if z == nil {
					continue
				}
				out = append(out, map[string]string{
					"zone_id":   derefString(z.ZoneId),
					"zone_name": derefString(z.LocalName),
					"status":    "Available",
				})
			}
		}
	}
	return out, rid, nil
}

func aliyunDescribeBandwidth(accessKey, secretKey, regionID, instanceType string) (map[string]interface{}, string, error) {
	client, err := newECSClient(accessKey, secretKey, regionID)
	if err != nil {
		return nil, "", err
	}
	resp, err := client.DescribeBandwidthLimitation(&ecsclient.DescribeBandwidthLimitationRequest{
		RegionId:     dara.String(regionID),
		InstanceType: dara.String(instanceType),
	})
	if err != nil {
		return nil, "", err
	}
	rid := ""
	result := map[string]interface{}{}
	if resp.Body != nil {
		rid = derefString(resp.Body.RequestId)
		result["RequestId"] = rid
		minBW, maxBW := 0, 0
		found := false
		payByTrafficFound := false
		if resp.Body.Bandwidths != nil {
			for _, bw := range resp.Body.Bandwidths.Bandwidth {
				if bw == nil {
					continue
				}
				chargeType := derefString(bw.InternetChargeType)
				min := derefInt32(bw.Min)
				max := derefInt32(bw.Max)
				if chargeType == "PayByTraffic" || (!payByTrafficFound && max > 0) {
					minBW = min
					maxBW = max
					found = true
					if chargeType == "PayByTraffic" {
						payByTrafficFound = true
						break
					}
				}
			}
		}
		if found {
			result["min_bandwidth"] = minBW
			result["max_bandwidth"] = maxBW
			result["BandwidthInfo"] = map[string]interface{}{
				"min_bandwidth": minBW,
				"max_bandwidth": maxBW,
			}
		}
	}
	return result, rid, nil
}

func aliyunDescribeInstanceTypes(accessKey, secretKey, regionID string, instanceTypes []string) ([]map[string]interface{}, error) {
	client, err := newECSClient(accessKey, secretKey, regionID)
	if err != nil {
		return nil, err
	}
	specs, err := aliyunDescribeInstanceTypeSpecs(client, instanceTypes)
	if err != nil {
		return nil, err
	}
	out := []map[string]interface{}{}
	for _, it := range instanceTypes {
		spec, ok := specs[it]
		if !ok {
			continue
		}
		out = append(out, map[string]interface{}{
			"instance_type":          it,
			"cpu_cores":                spec.cpuCores,
			"memory_gb":                spec.memoryGB,
			"architecture":             spec.architecture,
			"status":                   "available",
			"instance_type_category":   "",
			"gpu_amount":               "",
			"gpu_spec":                 "",
			"instance_type_family":     familyFromInstanceType(it),
			"diskSupport":              map[string]interface{}{"storage_types": []string{"cloud_essd"}},
			"storage_type":             "cloud_essd",
		})
	}
	return out, nil
}

var describePriceSystemDiskFallbacks = []string{
	"cloud_essd",
	"cloud_efficiency",
	"cloud_ssd",
	"cloud",
	"cloud_auto",
}

func isUnsupportedSystemDiskCategoryErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "InvalidSystemDiskCategory") ||
		strings.Contains(msg, "SystemDisk.Category is not valid")
}

func diskCategoriesForDescribePrice(preferred string) []string {
	preferred = strings.TrimSpace(preferred)
	if preferred == "" {
		return append([]string{""}, describePriceSystemDiskFallbacks...)
	}
	out := []string{preferred}
	for _, cat := range describePriceSystemDiskFallbacks {
		if cat != preferred {
			out = append(out, cat)
		}
	}
	return out
}

func buildDescribePriceRequest(regionID, instanceType, diskCategory string, storageGB, bandwidth int, spotStrategy string) *ecsclient.DescribePriceRequest {
	req := &ecsclient.DescribePriceRequest{
		RegionId:                dara.String(regionID),
		ResourceType:            dara.String("instance"),
		InstanceType:            dara.String(instanceType),
		PriceUnit:               dara.String("Hour"),
		SpotStrategy:            dara.String(spotStrategy),
		InternetMaxBandwidthOut: dara.Int32(int32(bandwidth)),
		IoOptimized:             dara.String("optimized"),
	}
	if diskCategory != "" {
		req.SystemDisk = &ecsclient.DescribePriceRequestSystemDisk{
			Category: dara.String(diskCategory),
			Size:     dara.Int32(int32(storageGB)),
		}
	}
	return req
}

func parsePriceDetailInfos(infos *ecsclient.DescribePriceResponseBodyPriceInfoPriceDetailInfos) map[string]interface{} {
	if infos == nil || len(infos.DetailInfo) == 0 {
		return nil
	}
	details := make([]map[string]interface{}, 0, len(infos.DetailInfo))
	for _, item := range infos.DetailInfo {
		if item == nil {
			continue
		}
		entry := map[string]interface{}{}
		if item.Resource != nil {
			entry["Resource"] = *item.Resource
			entry["resource"] = *item.Resource
		}
		if item.TradePrice != nil {
			entry["TradePrice"] = *item.TradePrice
			entry["trade_price"] = *item.TradePrice
		}
		if item.OriginalPrice != nil {
			entry["OriginalPrice"] = *item.OriginalPrice
			entry["original_price"] = *item.OriginalPrice
		}
		if item.DiscountPrice != nil {
			entry["DiscountPrice"] = *item.DiscountPrice
			entry["discount_price"] = *item.DiscountPrice
		}
		if len(entry) > 0 {
			details = append(details, entry)
		}
	}
	if len(details) == 0 {
		return nil
	}
	return map[string]interface{}{
		"DetailInfo":  details,
		"detail_info": details,
	}
}

func parseDescribePriceResponse(resp *ecsclient.DescribePriceResponse, diskCategory string) (map[string]interface{}, string) {
	rid := ""
	result := map[string]interface{}{"currency": "CNY"}
	if diskCategory != "" {
		result["system_disk_category"] = diskCategory
	}
	if resp == nil || resp.Body == nil {
		return result, rid
	}
	rid = derefString(resp.Body.RequestId)
	result["RequestId"] = rid
	if resp.Body.PriceInfo == nil || resp.Body.PriceInfo.Price == nil {
		return result, rid
	}
	p := resp.Body.PriceInfo.Price
	currency := "CNY"
	if p.Currency != nil && derefString(p.Currency) != "" {
		currency = derefString(p.Currency)
	}
	result["currency"] = currency

	priceObj := map[string]interface{}{
		"Currency": currency,
	}
	if p.TradePrice != nil {
		result["price"] = *p.TradePrice
		priceObj["TradePrice"] = *p.TradePrice
	} else if p.OriginalPrice != nil {
		result["price"] = *p.OriginalPrice
	}
	if p.OriginalPrice != nil {
		priceObj["OriginalPrice"] = *p.OriginalPrice
	}
	if p.DiscountPrice != nil {
		priceObj["DiscountPrice"] = *p.DiscountPrice
	}
	if detailInfos := parsePriceDetailInfos(p.DetailInfos); detailInfos != nil {
		priceObj["DetailInfos"] = detailInfos
	}
	result["Price"] = priceObj
	return result, rid
}

func aliyunDescribePrice(accessKey, secretKey, regionID, instanceType, diskCategory string, storageGB, bandwidth int, spotStrategy string) (map[string]interface{}, string, error) {
	client, err := newECSClient(accessKey, secretKey, regionID)
	if err != nil {
		return nil, "", err
	}
	if spotStrategy == "" {
		spotStrategy = "NoSpot"
	}
	var lastErr error
	for _, cat := range diskCategoriesForDescribePrice(diskCategory) {
		req := buildDescribePriceRequest(regionID, instanceType, cat, storageGB, bandwidth, spotStrategy)
		resp, err := client.DescribePrice(req)
		if err == nil {
			result, rid := parseDescribePriceResponse(resp, cat)
			return result, rid, nil
		}
		if !isUnsupportedSystemDiskCategoryErr(err) {
			return nil, "", err
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, "", lastErr
	}
	return nil, "", errors.New("describe price failed: no disk category candidates")
}

func aliyunDescribeImageSupportInstanceTypes(accessKey, secretKey, regionID, imageID string) (map[string]interface{}, string, error) {
	client, err := newECSClient(accessKey, secretKey, regionID)
	if err != nil {
		return nil, "", err
	}
	resp, err := client.DescribeImageSupportInstanceTypes(&ecsclient.DescribeImageSupportInstanceTypesRequest{
		RegionId: dara.String(regionID),
		ImageId:  dara.String(imageID),
	})
	if err != nil {
		return map[string]interface{}{
			"ok": false, "error": err.Error(), "region_id": regionID, "image_id": imageID,
		}, "", nil
	}
	rid := ""
	ids := []string{}
	if resp.Body != nil {
		rid = derefString(resp.Body.RequestId)
		if resp.Body.InstanceTypes != nil {
			for _, it := range resp.Body.InstanceTypes.InstanceType {
				if it != nil && it.InstanceTypeId != nil {
					ids = append(ids, *it.InstanceTypeId)
				}
			}
		}
	}
	return map[string]interface{}{
		"ok":                  true,
		"region_id":           regionID,
		"image_id":            imageID,
		"request_id":          rid,
		"instance_type_ids":   ids,
		"instance_type_count": len(ids),
		"error":               nil,
		"debug":               map[string]interface{}{},
	}, rid, nil
}

func aliyunAvailableInstances(accessKey, secretKey string, filters availableInstancesFilters) (availableInstancesResult, error) {
	client, err := newECSClient(accessKey, secretKey, filters.RegionID)
	if err != nil {
		return availableInstancesResult{}, err
	}

	candidateSet, emptyEarly, err := buildInstanceTypeCandidateSet(client, accessKey, secretKey, filters)
	if err != nil {
		return availableInstancesResult{}, err
	}
	if emptyEarly {
		return emptyAvailableInstancesResult(filters), nil
	}

	req := &ecsclient.DescribeAvailableResourceRequest{
		RegionId:            dara.String(filters.RegionID),
		ResourceType:        dara.String(filters.ResourceType),
		DestinationResource: dara.String(filters.DestinationResource),
	}
	if filters.ZoneID != "" {
		req.ZoneId = dara.String(filters.ZoneID)
	}
	applyAvailableInstancesFilters(req, filters)
	applyInstanceTypeCandidatesToDARRequest(req, candidateSet)
	resp, err := client.DescribeAvailableResource(req)
	if err != nil {
		return availableInstancesResult{}, err
	}
	rid := ""
	if resp.Body != nil {
		rid = derefString(resp.Body.RequestId)
	}
	instanceTypeIDs := collectAvailableInstanceTypeIDs(resp)
	instanceTypeIDs = filterInstanceTypeIDs(instanceTypeIDs, candidateSet)
	if len(instanceTypeIDs) == 0 {
		return emptyAvailableInstancesResult(filters), nil
	}
	if len(instanceTypeIDs) > availableInstancesFlatMax {
		return availableInstancesResult{
			Payload:   buildPaginatedAvailableInstances(instanceTypeIDs, filters.Page, filters.PageSize),
			RequestID: rid,
		}, nil
	}
	specs, err := aliyunDescribeInstanceTypeSpecs(client, instanceTypeIDs)
	if err != nil {
		return availableInstancesResult{RequestID: rid}, err
	}
	return availableInstancesResult{
		Payload:   buildFlatAvailableInstances(instanceTypeIDs, specs),
		RequestID: rid,
	}, nil
}

func collectAvailableInstanceTypeIDs(resp *ecsclient.DescribeAvailableResourceResponse) []string {
	instanceTypeIDs := []string{}
	seen := map[string]struct{}{}
	if resp == nil || resp.Body == nil || resp.Body.AvailableZones == nil {
		return instanceTypeIDs
	}
	for _, az := range resp.Body.AvailableZones.AvailableZone {
		if az == nil || az.AvailableResources == nil {
			continue
		}
		for _, ar := range az.AvailableResources.AvailableResource {
			if ar == nil || ar.SupportedResources == nil {
				continue
			}
			for _, sr := range ar.SupportedResources.SupportedResource {
				if sr == nil || sr.Value == nil {
					continue
				}
				if sr.Status != nil {
					status := strings.TrimSpace(*sr.Status)
					if status != "" && !strings.EqualFold(status, "Available") {
						continue
					}
				}
				it := strings.TrimSpace(*sr.Value)
				if it == "" {
					continue
				}
				if _, ok := seen[it]; ok {
					continue
				}
				seen[it] = struct{}{}
				instanceTypeIDs = append(instanceTypeIDs, it)
			}
		}
	}
	return instanceTypeIDs
}

type instanceTypeSpec struct {
	cpuCores     int
	memoryGB     float32
	architecture string
}

func aliyunDescribeInstanceTypeSpecs(client *ecsclient.Client, instanceTypeIDs []string) (map[string]instanceTypeSpec, error) {
	specs := map[string]instanceTypeSpec{}
	if len(instanceTypeIDs) == 0 {
		return specs, nil
	}
	missing := make([]string, 0, len(instanceTypeIDs))
	for _, id := range instanceTypeIDs {
		if cached, ok := getCachedInstanceTypeSpec(id); ok {
			specs[id] = cached
			continue
		}
		missing = append(missing, id)
	}
	for start := 0; start < len(missing); start += describeInstanceTypesBatchSize {
		end := start + describeInstanceTypesBatchSize
		if end > len(missing) {
			end = len(missing)
		}
		batch := missing[start:end]
		resp, err := client.DescribeInstanceTypes(&ecsclient.DescribeInstanceTypesRequest{
			InstanceTypes: dara.StringSlice(batch),
		})
		if err != nil {
			return nil, err
		}
		if resp.Body == nil || resp.Body.InstanceTypes == nil {
			continue
		}
		for _, inst := range resp.Body.InstanceTypes.InstanceType {
			if inst == nil || inst.InstanceTypeId == nil {
				continue
			}
			spec := instanceTypeSpec{
				cpuCores:     derefInt32(inst.CpuCoreCount),
				memoryGB:     derefFloat32(inst.MemorySize),
				architecture: fromAliyunCpuArchitecture(derefString(inst.CpuArchitecture)),
			}
			if spec.architecture == "" {
				spec.architecture = inferInstanceArchitecture(*inst.InstanceTypeId)
			}
			specs[*inst.InstanceTypeId] = spec
			setCachedInstanceTypeSpec(*inst.InstanceTypeId, spec)
		}
	}
	return specs, nil
}

func familyFromInstanceType(it string) string {
	parts := strings.Split(it, ".")
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return it
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func derefInt32(p *int32) int {
	if p == nil {
		return 0
	}
	return int(*p)
}

func derefFloat32(p *float32) float32 {
	if p == nil {
		return 0
	}
	return *p
}

func cloudQueryError(w http.ResponseWriter, status int, msg string) {
	writeErrorMapJSON(w, nil, status, map[string]interface{}{"status": "error", "message": msg})
}

func setCloudRequestIDHeader(w http.ResponseWriter, rid string) {
	if rid != "" {
		w.Header().Set("X-Cloud-Request-Id", rid)
	}
}
