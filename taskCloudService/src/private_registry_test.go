package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"registryhost"
	"strings"
	"testing"
	"time"
)

func TestResolveContainerImageMetadataRejectsAliyunVPC(t *testing.T) {
	start := time.Now()
	_, err := resolveContainerImageMetadata("registry-vpc.cn-qingdao.aliyuncs.com/ruandao/task2app-trae:x86_64-latest")
	if time.Since(start) > 2*time.Second {
		t.Fatalf("must fail-fast, took %s", time.Since(start))
	}
	if err == nil {
		t.Fatal("expected error")
	}
	var pr *registryhost.ErrPrivateRegistry
	if !errors.As(err, &pr) {
		t.Fatalf("want ErrPrivateRegistry got %T %v", err, err)
	}
	if !strings.Contains(err.Error(), "无法触及") {
		t.Fatalf("err=%v", err)
	}
}

func TestResolveTargetArchitecturesRejectsAliyunVPC(t *testing.T) {
	setupCloudTestDB(t)
	body := `{"image_url":"registry-vpc.cn-qingdao.aliyuncs.com/ruandao/task2app-trae:x86_64-latest"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/installed-images/resolve-target-architectures/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Trace-Id", "trace-private-registry")
	rec := httptest.NewRecorder()
	start := time.Now()
	handleInstalledImageResolveTargetArchitectures(rec, req)
	if time.Since(start) > 2*time.Second {
		t.Fatalf("must fail-fast, took %s", time.Since(start))
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "无法触及") {
		t.Fatalf("missing hint: %s", rec.Body.String())
	}
}
