package main

func mockCloudRegions() []map[string]string {
	return []map[string]string{
		{"region_id": "cn-hangzhou", "region_name": "华东1（杭州）"},
		{"region_id": "cn-hongkong", "region_name": "中国香港"},
	}
}

func mockCloudImages() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"id":           "m-ubuntu-001",
			"name":         "Ubuntu 22.04 LTS",
			"os_type":      "linux",
			"os_version":   "ubuntu_22_04",
			"architecture": "x86_64",
			"image_type":   "system",
			"size":         40,
		},
		{
			"id":           "m-centos-001",
			"name":         "CentOS 7",
			"os_type":      "linux",
			"os_version":   "centos_7",
			"architecture": "arm64",
			"image_type":   "system",
			"size":         40,
		},
	}
}

func mockCloudZones(regionID string) []map[string]string {
	return []map[string]string{
		{"zone_id": regionID + "-a", "zone_name": "可用区A"},
		{"zone_id": regionID + "-b", "zone_name": "可用区B"},
	}
}

func mockBandwidthLimitation() map[string]interface{} {
	return map[string]interface{}{
		"RequestId":     "mock-bandwidth-request-id",
		"min_bandwidth": 1,
		"max_bandwidth": 100,
		"BandwidthInfo": map[string]interface{}{
			"min_bandwidth": 1,
			"max_bandwidth": 100,
		},
	}
}

func mockAvailableInstances() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"instance_type":        "ecs.g6.large",
			"instance_type_id":     "ecs.g6.large",
			"cpu_cores":            2,
			"memory_gb":            8,
			"architecture":         "x86_64",
			"instance_type_family": "ecs.g6",
		},
		{
			"instance_type":        "ecs.t6-c1m2.large",
			"instance_type_id":     "ecs.t6-c1m2.large",
			"cpu_cores":            2,
			"memory_gb":            4,
			"architecture":         "x86_64",
			"instance_type_family": "ecs.t6",
		},
	}
}

func mockInstancePrice() map[string]interface{} {
	return map[string]interface{}{
		"RequestId":            "mock-price-request-id",
		"price":                0.42,
		"currency":             "CNY",
		"system_disk_category": "cloud_essd",
		"Price": map[string]interface{}{
			"Currency":      "CNY",
			"TradePrice":      0.42,
			"OriginalPrice":   0.5,
			"DiscountPrice":   0.08,
			"DetailInfos": map[string]interface{}{
				"DetailInfo": []map[string]interface{}{
					{"Resource": "instanceType", "resource": "instanceType", "TradePrice": 0.25, "trade_price": 0.25, "OriginalPrice": 0.3, "original_price": 0.3},
					{"Resource": "systemDisk", "resource": "systemDisk", "TradePrice": 0.12, "trade_price": 0.12, "OriginalPrice": 0.15, "original_price": 0.15},
					{"Resource": "bandwidth", "resource": "bandwidth", "TradePrice": 0.05, "trade_price": 0.05, "OriginalPrice": 0.05, "original_price": 0.05},
				},
			},
		},
	}
}

func mockInstanceDetails(instanceTypes []string) []map[string]interface{} {
	out := []map[string]interface{}{}
	for _, it := range instanceTypes {
		out = append(out, map[string]interface{}{
			"instance_type":          it,
			"cpu_cores":                2,
			"memory_gb":                8,
			"architecture":             "x86_64",
			"status":                   "available",
			"instance_type_category":   "GeneralPurpose",
			"gpu_amount":               "",
			"gpu_spec":                 "",
			"instance_type_family":     "ecs.g6",
			"diskSupport":              map[string]interface{}{"storage_types": []string{"cloud_essd"}},
			"storage_type":             "cloud_essd",
		})
	}
	return out
}

func mockImageSupportInstanceTypes() map[string]interface{} {
	return map[string]interface{}{
		"ok":                  true,
		"instance_type_ids":   []string{"ecs.g6.large", "ecs.t6-c1m2.large"},
		"instance_type_count": 2,
		"request_id":          "mock-image-support-rid",
		"error":               nil,
		"debug":               map[string]interface{}{},
	}
}
