package infrastructure

import (
	"net/http"
	"strings"
	"testing"
)

type panicRegistryClient struct{}

func (panicRegistryClient) Do(*http.Request) (*http.Response, error) {
	panic("must not fetch private registry")
}

func TestResolveContainerImageMetadataRejectsAliyunVPCWithoutFetch(t *testing.T) {
	_, err := resolveContainerImageMetadata(
		panicRegistryClient{},
		"registry-vpc.cn-qingdao.aliyuncs.com/ruandao/task2app-trae:x86_64-latest",
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "无法触及") {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(err.Error(), "registry.cn-qingdao.aliyuncs.com") {
		t.Fatalf("missing public mapping err=%v", err)
	}
}
