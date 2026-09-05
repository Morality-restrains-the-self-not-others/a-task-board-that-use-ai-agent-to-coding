package main

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	ecsclient "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/dara"
)

func TestToAliyunCpuArchitecture(t *testing.T) {
	cases := map[string]string{
		"x86_64": "X86",
		"amd64":  "X86",
		"arm64":  "ARM",
		"aarch64": "ARM",
		"":       "",
		"riscv":  "",
	}
	for input, want := range cases {
		if got := toAliyunCpuArchitecture(input); got != want {
			t.Fatalf("toAliyunCpuArchitecture(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestFromAliyunCpuArchitecture(t *testing.T) {
	if got := fromAliyunCpuArchitecture("X86"); got != "x86_64" {
		t.Fatalf("expected x86_64, got %q", got)
	}
	if got := fromAliyunCpuArchitecture("ARM"); got != "arm64" {
		t.Fatalf("expected arm64, got %q", got)
	}
}

func TestIntersectStringSets(t *testing.T) {
	a := stringSetFromSlice([]string{"ecs.a", "ecs.b"})
	b := stringSetFromSlice([]string{"ecs.b", "ecs.c"})
	got := intersectStringSets(a, b)
	if len(got) != 1 {
		t.Fatalf("expected 1 intersection, got %d", len(got))
	}
	if _, ok := got["ecs.b"]; !ok {
		t.Fatalf("expected ecs.b in intersection")
	}
}

func TestFilterInstanceTypeIDs(t *testing.T) {
	allowed := stringSetFromSlice([]string{"ecs.a"})
	got := filterInstanceTypeIDs([]string{"ecs.a", "ecs.b"}, allowed)
	if len(got) != 1 || got[0] != "ecs.a" {
		t.Fatalf("unexpected filtered ids: %#v", got)
	}
}

func TestInstanceTypeCandidateQueryValue(t *testing.T) {
	got := instanceTypeCandidateQueryValue(stringSetFromSlice([]string{"ecs.b", "ecs.a", "ecs.c"}))
	if got != "ecs.a,ecs.b,ecs.c" {
		t.Fatalf("unexpected query value: %q", got)
	}
	large := make([]string, 0, darInstanceTypesMaxQuery+1)
	for i := 0; i < darInstanceTypesMaxQuery+1; i++ {
		large = append(large, "ecs.test."+strconv.Itoa(i))
	}
	if instanceTypeCandidateQueryValue(stringSetFromSlice(large)) != "" {
		t.Fatalf("expected empty query when candidate count exceeds max")
	}
}

func TestApplyInstanceTypeCandidatesToDARRequest(t *testing.T) {
	req := &ecsclient.DescribeAvailableResourceRequest{}
	req.Cores = dara.Int32(2)
	req.Memory = dara.Float32(4)
	applyInstanceTypeCandidatesToDARRequest(req, stringSetFromSlice([]string{"ecs.g6.large", "ecs.t6.large"}))
	if req.InstanceType == nil || *req.InstanceType != "ecs.g6.large,ecs.t6.large" {
		t.Fatalf("expected joined InstanceType, got %+v", req.InstanceType)
	}
	if req.Cores != nil || req.Memory != nil {
		t.Fatalf("expected Cores/Memory cleared when InstanceType is set, got Cores=%+v Memory=%+v", req.Cores, req.Memory)
	}
}

func TestParseAvailableInstancesFiltersImageArchitecture(t *testing.T) {
	req := httptestNewGetRequest(
		"/cloud/available-instances/?image_architecture=x86_64&image_id=m-123&container_image_id=859671040643174400",
	)
	f := parseAvailableInstancesFilters(req, "cn-hongkong", "cn-hongkong-b")
	if f.ImageArchitecture != "x86_64" {
		t.Fatalf("expected image_architecture, got %q", f.ImageArchitecture)
	}
	if f.CloudImageID != "m-123" {
		t.Fatalf("expected image_id, got %q", f.CloudImageID)
	}
}

func httptestNewGetRequest(rawURL string) *http.Request {
	return httptest.NewRequest(http.MethodGet, rawURL, nil)
}
