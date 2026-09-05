package main

import (
	"sync"
	"time"
)

const instanceTypeSpecCacheTTL = 6 * time.Hour

type cachedInstanceTypeSpec struct {
	spec      instanceTypeSpec
	expiresAt time.Time
}

var instanceTypeSpecCache sync.Map

func getCachedInstanceTypeSpec(instanceTypeID string) (instanceTypeSpec, bool) {
	raw, ok := instanceTypeSpecCache.Load(instanceTypeID)
	if !ok {
		return instanceTypeSpec{}, false
	}
	entry, ok := raw.(cachedInstanceTypeSpec)
	if !ok || time.Now().After(entry.expiresAt) {
		instanceTypeSpecCache.Delete(instanceTypeID)
		return instanceTypeSpec{}, false
	}
	return entry.spec, true
}

func setCachedInstanceTypeSpec(instanceTypeID string, spec instanceTypeSpec) {
	instanceTypeSpecCache.Store(instanceTypeID, cachedInstanceTypeSpec{
		spec:      spec,
		expiresAt: time.Now().Add(instanceTypeSpecCacheTTL),
	})
}

func resetInstanceTypeSpecCacheForTest() {
	instanceTypeSpecCache = sync.Map{}
}
