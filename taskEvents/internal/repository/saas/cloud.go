package saas

import (
	"bytes"
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

const (
	CloudEventTypeStart  = "start"
	CloudEventPending    = "pending"
	CloudEventProcessing = "processing"
	CloudEventSuccess    = "success"
	CloudEventError      = "error"
)

// CloudAuthorization holds AK/SK for Aliyun calls.
// ID 为平台 SSOT 的字符串授权 ID（cpa_<snowflake>，见 cloud_platform_authorizations.id）；
// 历史代码曾按 int64 解析导致 "cpa_-3077015452416368750" 解析失败 → 误报"缺少 authorization_id"
// （OPT-20260809-026：task_15370673744378765865 停止服务器失败）。
type CloudAuthorization struct {
	ID           string
	PlatformType string
	SecretID     string
	SecretKey    string
}

func taskCloudServiceBase() string {
	if v := strings.TrimSpace(os.Getenv("TASK_CLOUD_SERVICE_BASE_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://127.0.0.1:8018"
}

func cloudInternalSecret(r *Repository) string {
	if v := strings.TrimSpace(os.Getenv("SHARED_INTERNAL_SECRET")); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("TASK_CLOUD_INTERNAL_SECRET")); v != "" {
		return v
	}
	if r != nil && strings.TrimSpace(r.secret) != "" {
		return strings.TrimSpace(r.secret)
	}
	return taskEventsSecret()
}

func (r *Repository) doCloudJSON(method, path string, payload interface{}, out interface{}) (int, error) {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return 0, err
		}
		body = bytes.NewReader(raw)
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	req, err := http.NewRequest(method, taskCloudServiceBase()+path, body)
	if err != nil {
		return 0, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if sec := cloudInternalSecret(r); sec != "" {
		req.Header.Set("X-Internal-Secret", sec)
	}
	client := r.client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if out != nil && len(raw) > 0 && resp.StatusCode < 500 {
		_ = json.Unmarshal(raw, out)
	}
	if resp.StatusCode >= 400 {
		var detail struct {
			Detail string `json:"detail"`
			Error  string `json:"error"`
			Code   string `json:"code"`
		}
		_ = json.Unmarshal(raw, &detail)
		msg := detail.Detail
		if msg == "" {
			msg = detail.Error
		}
		if msg == "" {
			msg = strings.TrimSpace(string(raw))
		}
		if msg == "" {
			msg = fmt.Sprintf("taskCloudService %s status %d", path, resp.StatusCode)
		}
		return resp.StatusCode, fmt.Errorf("%s", msg)
	}
	return resp.StatusCode, nil
}

func fetchCloudAuthorizationFromGo(authID string) (*CloudAuthorization, error) {
	// lookup 端点支持字符串 ID（cpa_<snowflake>）：tenantID 为空时走 loadCloudAuthByID(authID)。
	url := fmt.Sprintf("%s/api/internal/cloud-platform-authorizations/lookup?id=%s", taskCloudServiceBase(), url.QueryEscape(authID))
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("go lookup status %d", resp.StatusCode)
	}
	raw, _ := io.ReadAll(resp.Body)
	var body map[string]interface{}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	a := &CloudAuthorization{ID: authID}
	if v, ok := body["platform_type"].(string); ok {
		a.PlatformType = v
	}
	if v, ok := body["secret_id"].(string); ok {
		a.SecretID = v
	}
	if v, ok := body["secret_key"].(string); ok {
		a.SecretKey = v
	}
	if a.SecretID == "" || a.SecretKey == "" {
		return nil, fmt.Errorf("incomplete authorization from go")
	}
	return a, nil
}

func (r *Repository) CloudAuthorizationByID(authID string) (*CloudAuthorization, error) {
	_ = r
	authID = strings.TrimSpace(authID)
	if authID == "" {
		return nil, fmt.Errorf("authorization id empty")
	}
	a, err := fetchCloudAuthorizationFromGo(authID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, fmt.Errorf("authorization %s not found", authID)
	}
	return a, nil
}

// CreateAccessKeyIAMAssociation inserts access_key_iam_associations via taskCloudService.
func (r *Repository) CreateAccessKeyIAMAssociation(authID string, accessKey, iamID string) error {
	_, err := r.doCloudJSON(http.MethodPost, "/api/internal/access-key-iam-associations/", map[string]interface{}{
		"auth_id":    authID,
		"access_key": accessKey,
		"iam_id":     iamID,
	}, nil)
	return err
}

// PendingStartEvent is the latest pending start CloudServerEvent row.
type PendingStartEvent struct {
	ID     int64
	Status string
}

func (r *Repository) LatestPendingStartEvent(companyID int64, taskID string) (*PendingStartEvent, error) {
	q := url.Values{
		"company_id": {fmt.Sprint(companyID)},
		"task_id":    {taskID},
	}
	var body map[string]interface{}
	_, err := r.doCloudJSON(http.MethodGet, "/api/internal/cloud-server-events/latest-pending-start?"+q.Encode(), nil, &body)
	if err != nil {
		return nil, err
	}
	id, err := asInt64(body["id"])
	if err != nil {
		return nil, err
	}
	status, _ := body["status"].(string)
	return &PendingStartEvent{ID: id, Status: status}, nil
}

func (r *Repository) UpdateCloudServerEventStatus(eventID int64, status, errMsg string) error {
	_, err := r.doCloudJSON(http.MethodPost, "/api/internal/cloud-server-events/update-status", map[string]interface{}{
		"event_id":      fmt.Sprint(eventID),
		"status":        status,
		"error_message": errMsg,
	}, nil)
	return err
}

// ClaimCloudServerStartEvent 原子地 claim 启动事件（pending→processing）。
// taskCloudService 侧带 from_status 守卫做状态迁移：仅当行仍为 pending 时成功。
// 返回 claimed=false（err=nil）表示事件已被并发消费者抢先 claim / 推进，
// 调用方应把「丢失 claim」当作幂等跳过，而不是重试或投 DLT。
func (r *Repository) ClaimCloudServerStartEvent(eventID int64) (bool, error) {
	code, err := r.doCloudJSON(http.MethodPost, "/api/internal/cloud-server-events/update-status", map[string]interface{}{
		"event_id":    fmt.Sprint(eventID),
		"status":      CloudEventProcessing,
		"from_status": CloudEventPending,
	}, nil)
	if err != nil {
		if code == http.StatusConflict {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CloudServerEventStatusByID 按 event_id 查启动事件状态（OPT-20260818-015）。
// LatestPendingStartEvent 未命中（如重放已完成事件）时，据此识别「已成功处理」以幂等跳过，
// 避免把已完成事件误投 DLT。
func (r *Repository) CloudServerEventStatusByID(eventID string) (string, error) {
	var body map[string]interface{}
	_, err := r.doCloudJSON(http.MethodGet, "/api/internal/cloud-server-events/status?event_id="+url.QueryEscape(eventID), nil, &body)
	if err != nil {
		return "", err
	}
	status, _ := body["status"].(string)
	if status == "" {
		return "", fmt.Errorf("cloud server event %s status empty", eventID)
	}
	return status, nil
}

func (r *Repository) UpdateCloudServerEventData(eventID int64, data map[string]interface{}) error {
	_, err := r.doCloudJSON(http.MethodPost, "/api/internal/cloud-server-events/update-data", map[string]interface{}{
		"event_id":   fmt.Sprint(eventID),
		"event_data": data,
	}, nil)
	return err
}

// PersistJobExecutionEvent writes a container job-stream step/lifecycle row via Cloud internal API.
func (r *Repository) PersistJobExecutionEvent(_ context.Context, rec any) error {
	_, err := r.doCloudJSON(http.MethodPost, "/api/internal/cloud/job-execution-events/", rec, nil)
	return err
}

// InitTenantFeatureParamsDirect initializes TenantFeatureParams for a new company
// via taskCloudService directly (bypasses Django). Idempotent — returns the existing
// row if one already exists for this company.
func (r *Repository) InitTenantFeatureParamsDirect(companyID string) (map[string]interface{}, error) {
	var body map[string]interface{}
	_, err := r.doCloudJSON(http.MethodPost, "/api/internal/cloud/init-tenant-feature-params/", map[string]string{
		"company_id": companyID,
	}, &body)
	if err != nil {
		return nil, err
	}
	return body, nil
}
