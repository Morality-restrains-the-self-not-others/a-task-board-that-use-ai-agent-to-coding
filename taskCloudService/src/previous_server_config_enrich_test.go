package main

import (
	"testing"
	"time"
)

func TestEnrichPreviousServerConfigSkipsMockAuth(t *testing.T) {
	h := &CloudServerConfigHistory{
		AuthorizationID: "mock-auth",
		InstanceTypeID:  "ecs.c7a.large",
		Region:          "cn-hongkong",
		ZoneID:          "cn-hongkong-b",
		CpuCores:        2,
		MemoryGB:        4,
		StorageGB:       40,
		CreatedAt:       time.Now().UTC(),
	}
	cfg := previousServerConfigFromHistory(h)
	out := enrichPreviousServerConfig("100", cfg, h)
	if _, ok := out["availability"]; ok {
		t.Fatal("mock auth should skip availability")
	}
	if _, ok := out["price"]; ok {
		t.Fatal("mock auth should skip price")
	}
}

func TestEnrichPreviousServerConfigShape(t *testing.T) {
	h := &CloudServerConfigHistory{
		AuthorizationID: "mock-auth",
		Platform:        "mock",
		InstanceTypeID:  "",
		CpuCores:        1,
		CreatedAt:       time.Now().UTC(),
	}
	cfg := previousServerConfigFromHistory(h)
	if cfg["platform"] != "mock" {
		t.Fatalf("platform=%v", cfg["platform"])
	}
	hw, ok := cfg["hardware_config"].(map[string]interface{})
	if !ok {
		t.Fatal("missing hardware_config")
	}
	if hw["cpu_cores"] != 1 {
		t.Fatalf("cpu=%v", hw["cpu_cores"])
	}
}
