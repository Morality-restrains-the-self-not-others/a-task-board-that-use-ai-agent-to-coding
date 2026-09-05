package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var cloudHTTP = &http.Client{Timeout: 10 * time.Second}

func taskCloudServiceBaseURL() string {
	if v := strings.TrimSpace(os.Getenv("TASK_CLOUD_SERVICE_BASE_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://127.0.0.1:8018"
}

func cloudInternalHeaders() map[string]string {
	h := map[string]string{}
	if cfg.InternalSecret != "" {
		h["X-Internal-Secret"] = cfg.InternalSecret
	}
	return h
}

func lookupInstalledImage(tenantID, imageID, imageName string) (map[string]interface{}, error) {
	tenantID = strings.TrimSpace(tenantID)
	imageID = strings.TrimSpace(imageID)
	imageName = strings.TrimSpace(imageName)
	if tenantID == "" || imageID == "" {
		return nil, nil
	}
	q := url.Values{}
	q.Set("tenant_id", tenantID)
	q.Set("id", imageID)
	if imageName != "" {
		q.Set("name", imageName)
	}
	lookupURL := taskCloudServiceBaseURL() + "/api/internal/tenant-installed-images/lookup?" + q.Encode()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, lookupURL, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range cloudInternalHeaders() {
		req.Header.Set(k, v)
	}
	resp, err := cloudHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("installed image lookup status %d", resp.StatusCode)
	}
	var out map[string]interface{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func resolveInstalledImageName(tenantID, imageID string) string {
	img, err := lookupInstalledImage(tenantID, imageID, "")
	if err != nil || img == nil {
		return ""
	}
	return strings.TrimSpace(strField(img, "name"))
}
