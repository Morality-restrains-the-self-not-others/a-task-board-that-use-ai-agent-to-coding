package main

import (
	"strconv"
	"strings"
	"sync"
	"time"
)

var aliyunAvailableInstancesFn = aliyunAvailableInstances

const availableInstancesCacheTTLDefault = 45 * time.Second

var availableInstancesCacheTTL = availableInstancesCacheTTLDefault
var availableInstancesNow = time.Now

type availableInstancesCacheEntry struct {
	payload interface{}
	rid     string
	expire  time.Time
}

var availableInstancesCache sync.Map

func availableInstancesCacheKey(secretID string, f availableInstancesFilters) string {
	return strings.Join([]string{
		secretID,
		f.RegionID,
		f.ZoneID,
		f.Cores,
		f.Memory,
		f.IoOptimized,
		f.SystemDiskCategory,
		f.DataDiskCategory,
		f.NetworkCategory,
		f.SpotStrategy,
		f.InstanceChargeType,
		f.SpotDuration,
		f.ResourceType,
		f.DestinationResource,
		f.ImageArchitecture,
		f.CloudImageID,
		f.InstanceFamily,
		strconv.Itoa(f.Page),
		strconv.Itoa(f.PageSize),
	}, "|")
}

func fetchAvailableInstances(secretID, secretKey string, f availableInstancesFilters) (payload interface{}, rid string, err error) {
	key := availableInstancesCacheKey(secretID, f)
	if v, ok := availableInstancesCache.Load(key); ok {
		ent := v.(availableInstancesCacheEntry)
		if availableInstancesNow().Before(ent.expire) {
			return ent.payload, ent.rid, nil
		}
	}
	result, err := aliyunAvailableInstancesFn(secretID, secretKey, f)
	if err != nil {
		return nil, "", err
	}
	availableInstancesCache.Store(key, availableInstancesCacheEntry{
		payload: result.Payload,
		rid:     result.RequestID,
		expire:  availableInstancesNow().Add(availableInstancesCacheTTL),
	})
	return result.Payload, result.RequestID, nil
}

func resetAvailableInstancesCache() {
	availableInstancesCache.Range(func(k, _ interface{}) bool {
		availableInstancesCache.Delete(k)
		return true
	})
}
