package main

import (
	"math"
	"strconv"
)

const availableInstancesFlatMax = 30

type availableInstancesResult struct {
	Payload   interface{}
	RequestID string
}

func parsePaginationQuery(pageRaw, pageSizeRaw string) (page, pageSize int) {
	page = 1
	pageSize = 10
	if pageRaw != "" {
		if v, err := strconv.Atoi(pageRaw); err == nil && v > 0 {
			page = v
		}
	}
	if pageSizeRaw != "" {
		if v, err := strconv.Atoi(pageSizeRaw); err == nil && v > 0 {
			pageSize = v
		}
	}
	return page, pageSize
}

func buildPaginatedAvailableInstances(instanceTypeIDs []string, page, pageSize int) map[string]interface{} {
	total := len(instanceTypeIDs)
	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(pageSize)))
	}
	return map[string]interface{}{
		"instance_types": instanceTypeIDs,
		"pagination": map[string]interface{}{
			"page":             page,
			"page_size":        pageSize,
			"total_pages":      totalPages,
			"total_instances":  total,
		},
	}
}

func buildFlatAvailableInstances(instanceTypeIDs []string, specs map[string]instanceTypeSpec) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(instanceTypeIDs))
	for _, it := range instanceTypeIDs {
		spec := specs[it]
		out = append(out, map[string]interface{}{
			"instance_type":        it,
			"instance_type_id":     it,
			"cpu_cores":            spec.cpuCores,
			"memory_gb":            spec.memoryGB,
			"architecture":         spec.architecture,
			"instance_type_family": familyFromInstanceType(it),
		})
	}
	return out
}
