package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var aiProviderHTTP = &http.Client{Timeout: 30 * time.Second}

// aiProviderUserdataRetry 镜像市场瞬时抖动（502/503/504/网关超时）的有限重试参数。
// start-vm 首轮遇 502 不应被当成配置永久失败；重试耗尽后仍走现有 error SSE + failed binding。
var (
	aiProviderUserdataMaxRetries = 2
	aiProviderUserdataRetryDelay = 500 * time.Millisecond
)

func normalizePublicImageItem(item map[string]interface{}) {
	if item == nil {
		return
	}
	if v, ok := item["id"]; ok {
		item["id"] = stringifyID(v)
	}
	if vendor, ok := item["vendor"].(map[string]interface{}); ok {
		if v, ok := vendor["id"]; ok {
			vendor["id"] = stringifyID(v)
		}
	}
	if tpl, ok := item["userdata_template"].(map[string]interface{}); ok {
		if v, ok := tpl["id"]; ok {
			tpl["id"] = stringifyID(v)
		}
	}
	rawEnvs, ok := item["runtime_environments"].([]interface{})
	if !ok {
		return
	}
	for _, raw := range rawEnvs {
		env, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		if tpl, ok := env["userdata_template"].(map[string]interface{}); ok {
			if v, ok := tpl["id"]; ok {
				tpl["id"] = stringifyID(v)
			}
		}
	}
}

func aiProviderErrorDetail(raw []byte, statusCode int) error {
	var errBody map[string]interface{}
	if json.Unmarshal(raw, &errBody) == nil {
		if detail, ok := errBody["detail"].(string); ok && strings.TrimSpace(detail) != "" {
			return fmt.Errorf("%s", detail)
		}
	}
	return fmt.Errorf("镜像服务返回错误: %d", statusCode)
}

func fetchAIPublicJSONList(endpoint string) ([]map[string]interface{}, error) {
	resp, err := aiProviderHTTP.Get(endpoint)
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") || strings.Contains(err.Error(), "no such host") {
			return nil, fmt.Errorf("无法连接到镜像服务，请检查网络")
		}
		return nil, fmt.Errorf("镜像服务请求失败: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, aiProviderErrorDetail(raw, resp.StatusCode)
	}
	var arr []map[string]interface{}
	if len(raw) == 0 {
		return []map[string]interface{}{}, nil
	}
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, fmt.Errorf("镜像服务响应格式无效")
	}
	if arr == nil {
		return []map[string]interface{}{}, nil
	}
	for i := range arr {
		normalizePublicImageItem(arr[i])
	}
	return arr, nil
}

func fetchAIPublicCatalog() ([]map[string]interface{}, error) {
	return fetchAIPublicJSONList(cfg.AIProviderBaseURL + "/api/public/catalog/")
}

func fetchAIPublicDevCatalog(saasUserID string) ([]map[string]interface{}, error) {
	base := cfg.AIProviderBaseURL + "/api/public/vendor-development-catalog/"
	u, err := url.Parse(base)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("saas_user_id", saasUserID)
	u.RawQuery = q.Encode()
	return fetchAIPublicJSONList(u.String())
}

func fetchAIPublicUnsubmittedImage(vendorID, containerID string) (map[string]interface{}, error) {
	base := cfg.AIProviderBaseURL + "/api/public/unsubmitted-image/"
	u, err := url.Parse(base)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("vendor_id", vendorID)
	q.Set("container_id", containerID)
	u.RawQuery = q.Encode()
	resp, err := aiProviderHTTP.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("无法连接到镜像服务，请检查网络")
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, aiProviderErrorDetail(raw, resp.StatusCode)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("镜像服务响应格式无效")
	}
	normalizePublicImageItem(out)
	return out, nil
}

func fetchAIPublicImageRuntimeEnvironments(imageID string) ([]map[string]interface{}, error) {
	base := cfg.AIProviderBaseURL + "/api/public/image-runtime-environments/"
	u, err := url.Parse(base)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("image_id", imageID)
	u.RawQuery = q.Encode()
	return fetchAIPublicJSONList(u.String())
}

func fetchAIPublicRuntimeUserdataBody(imageID, cloudServerImageID, platformType, regionID string) (string, error) {
	platformType = strings.TrimSpace(platformType)
	regionID = strings.TrimSpace(regionID)
	if platformType == "" || regionID == "" {
		return "", fmt.Errorf("platform_type and region_id required for runtime userdata")
	}
	imageID = strings.TrimSpace(imageID)
	cloudServerImageID = strings.TrimSpace(cloudServerImageID)
	if imageID == "" && cloudServerImageID == "" {
		return "", fmt.Errorf("image_id or cloud_server_image_id required")
	}
	base := cfg.AIProviderBaseURL + "/api/public/runtime-userdata/"
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("platform_type", platformType)
	q.Set("region", regionID)
	if imageID != "" {
		q.Set("image_id", imageID)
	}
	if cloudServerImageID != "" {
		q.Set("cloud_server_image_id", cloudServerImageID)
	}
	u.RawQuery = q.Encode()

	var lastErr error
	for attempt := 0; attempt <= aiProviderUserdataMaxRetries; attempt++ {
		content, status, err := fetchAIPublicRuntimeUserdataOnce(u.String())
		if err == nil {
			return content, nil
		}
		lastErr = err
		if !isRetryableImageMarketError(status, err) {
			return "", err
		}
		if attempt < aiProviderUserdataMaxRetries {
			time.Sleep(aiProviderUserdataRetryDelay)
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("调用镜像服务失败")
	}
	return "", lastErr
}

func fetchAIPublicRuntimeUserdataOnce(endpoint string) (string, int, error) {
	resp, err := aiProviderHTTP.Get(endpoint)
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") || strings.Contains(err.Error(), "no such host") {
			return "", 0, fmt.Errorf("无法连接到镜像服务，请检查网络")
		}
		return "", 0, fmt.Errorf("镜像服务请求失败: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", resp.StatusCode, aiProviderErrorDetail(raw, resp.StatusCode)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", resp.StatusCode, fmt.Errorf("镜像服务响应格式无效")
	}
	content, _ := out["content"].(string)
	return strings.TrimSpace(content), resp.StatusCode, nil
}

// isRetryableImageMarketError 判定镜像市场调用失败是否属于瞬时抖动：502/503/504 网关类状态，
// 或连接层超时/重置/截断。4xx 配置类错误与连接性故障（拒绝/域名不存在）不重试。
func isRetryableImageMarketError(status int, err error) bool {
	if status == http.StatusBadGateway || status == http.StatusServiceUnavailable || status == http.StatusGatewayTimeout {
		return true
	}
	if err != nil {
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() {
			return true
		}
		msg := err.Error()
		return strings.Contains(msg, "connection reset by peer") || strings.Contains(msg, "EOF")
	}
	return false
}
