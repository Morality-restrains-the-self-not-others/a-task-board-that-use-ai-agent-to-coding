package main

import (
	"sort"
	"strconv"
	"strings"

	ecsclient "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/dara"
)

func resolveCloudImageIDForInstanceQuery(tenantID, containerImageID, platformType, regionID string) string {
	containerImageID = strings.TrimSpace(containerImageID)
	tenantID = strings.TrimSpace(tenantID)
	platformType = strings.TrimSpace(platformType)
	regionID = strings.TrimSpace(regionID)
	if containerImageID == "" || tenantID == "" {
		return ""
	}
	img, err := getInstalledImage(tenantID, containerImageID)
	if err != nil || img == nil || strings.TrimSpace(img.ExternalImageID) == "" {
		return ""
	}
	envs, err := fetchAIPublicImageRuntimeEnvironments(img.ExternalImageID)
	if err != nil || len(envs) == 0 {
		return ""
	}
	for _, env := range envs {
		if runtimeEnvString(env, "platform_type") == platformType && runtimeEnvString(env, "region") == regionID {
			return runtimeEnvString(env, "image_id")
		}
	}
	return ""
}

func stringSetFromSlice(ids []string) map[string]struct{} {
	if len(ids) == 0 {
		return map[string]struct{}{}
	}
	out := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		out[id] = struct{}{}
	}
	return out
}

func intersectStringSets(a, b map[string]struct{}) map[string]struct{} {
	if a == nil && b == nil {
		return nil
	}
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	out := make(map[string]struct{})
	for id := range a {
		if _, ok := b[id]; ok {
			out[id] = struct{}{}
		}
	}
	return out
}

func filterInstanceTypeIDs(ids []string, allowed map[string]struct{}) []string {
	if allowed == nil {
		return ids
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, ok := allowed[id]; ok {
			out = append(out, id)
		}
	}
	return out
}

func fetchInstanceTypeIDsMatchingCPUArchitecture(
	client *ecsclient.Client,
	filters availableInstancesFilters,
	aliCPUArch string,
) ([]string, error) {
	if filters.Cores == "" || filters.Memory == "" {
		return nil, nil
	}
	cores, err := strconv.ParseInt(filters.Cores, 10, 32)
	if err != nil {
		return nil, nil
	}
	memory, err := strconv.ParseFloat(filters.Memory, 32)
	if err != nil {
		return nil, nil
	}

	ids := make([]string, 0, 32)
	seen := map[string]struct{}{}
	var nextToken *string
	for {
		req := &ecsclient.DescribeInstanceTypesRequest{
			MinimumCpuCoreCount: dara.Int32(int32(cores)),
			MaximumCpuCoreCount: dara.Int32(int32(cores)),
			MinimumMemorySize:   dara.Float32(float32(memory)),
			MaximumMemorySize:   dara.Float32(float32(memory)),
			MaxResults:          dara.Int64(100),
		}
		if aliCPUArch != "" {
			req.CpuArchitecture = dara.String(aliCPUArch)
		}
		if nextToken != nil && *nextToken != "" {
			req.NextToken = nextToken
		}
		resp, err := client.DescribeInstanceTypes(req)
		if err != nil {
			return nil, err
		}
		if resp.Body == nil || resp.Body.InstanceTypes == nil {
			break
		}
		for _, inst := range resp.Body.InstanceTypes.InstanceType {
			if inst == nil || inst.InstanceTypeId == nil {
				continue
			}
			id := strings.TrimSpace(*inst.InstanceTypeId)
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
		if resp.Body.NextToken == nil || strings.TrimSpace(*resp.Body.NextToken) == "" {
			break
		}
		nextToken = resp.Body.NextToken
	}
	sort.Strings(ids)
	return ids, nil
}

func buildInstanceTypeCandidateSet(
	client *ecsclient.Client,
	accessKey, secretKey string,
	filters availableInstancesFilters,
) (map[string]struct{}, bool, error) {
	var imageSupportSet map[string]struct{}
	var specMatchedSet map[string]struct{}

	if filters.CloudImageID != "" {
		payload, _, err := aliyunDescribeImageSupportInstanceTypes(accessKey, secretKey, filters.RegionID, filters.CloudImageID)
		if err != nil {
			return nil, false, err
		}
		rawIDs, _ := payload["instance_type_ids"].([]string)
		if rawIDs == nil {
			if anyIDs, ok := payload["instance_type_ids"].([]interface{}); ok {
				for _, item := range anyIDs {
					if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
						rawIDs = append(rawIDs, s)
					}
				}
			}
		}
		if ok, _ := payload["ok"].(bool); ok || len(rawIDs) > 0 {
			imageSupportSet = stringSetFromSlice(rawIDs)
		}
	}

	if filters.Cores != "" && filters.Memory != "" {
		aliArch := ""
		if filters.ImageArchitecture != "" {
			aliArch = toAliyunCpuArchitecture(filters.ImageArchitecture)
		}
		ids, err := fetchInstanceTypeIDsMatchingCPUArchitecture(client, filters, aliArch)
		if err != nil {
			return nil, false, err
		}
		if ids != nil {
			specMatchedSet = stringSetFromSlice(ids)
		}
	}

	narrowed := intersectStringSets(imageSupportSet, specMatchedSet)
	if narrowed != nil && len(narrowed) == 0 {
		return nil, true, nil
	}
	// OPT-20260722-035: filter by instance family prefix (e.g. "g6" matches "ecs.g6.*")
	if f := strings.TrimSpace(filters.InstanceFamily); f != "" {
		if narrowed == nil {
			// No narrowing yet: use all spec-matched as base
			narrowed = make(map[string]struct{})
			for id := range specMatchedSet {
				narrowed[id] = struct{}{}
			}
		}
		familyFiltered := make(map[string]struct{})
		for id := range narrowed {
			if strings.Contains(id, "."+f+".") || strings.HasPrefix(id, f+".") {
				familyFiltered[id] = struct{}{}
			}
		}
		narrowed = familyFiltered
	}
	if narrowed != nil && len(narrowed) == 0 {
		return nil, true, nil
	}
	return narrowed, false, nil
}

const darInstanceTypesMaxQuery = 50

func instanceTypeCandidateQueryValue(candidateSet map[string]struct{}) string {
	if candidateSet == nil || len(candidateSet) == 0 || len(candidateSet) > darInstanceTypesMaxQuery {
		return ""
	}
	ids := make([]string, 0, len(candidateSet))
	for id := range candidateSet {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return strings.Join(ids, ",")
}

func applyInstanceTypeCandidatesToDARRequest(req *ecsclient.DescribeAvailableResourceRequest, candidateSet map[string]struct{}) {
	queryValue := instanceTypeCandidateQueryValue(candidateSet)
	if queryValue == "" {
		return
	}
	req.InstanceType = dara.String(queryValue)
	// Aliyun rejects InstanceType together with Cores/Memory on DescribeAvailableResource.
	// CPU/memory filtering is applied earlier via DescribeInstanceTypes candidate narrowing.
	req.Cores = nil
	req.Memory = nil
}

func emptyAvailableInstancesResult(filters availableInstancesFilters) availableInstancesResult {
	page, pageSize := filters.Page, filters.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	return availableInstancesResult{
		Payload: buildPaginatedAvailableInstances(nil, page, pageSize),
	}
}
