package cloudconfig

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"snowflake"
)

// StartVMResult is persisted to task_cloud.db via taskCloudService.
type StartVMResult struct {
	Platform        string
	InstanceID      string
	SecurityGroupID string
	VSwitchID       string
	Region          string
	ZoneID          string
	PublicIP        string
	ServerURL       string
	LaunchRequestID string
	ClientToken     string
	CommentID       string
	CSCID           string
}

// ConfigRow is a minimal config row for stop / migrate-await processing.
type ConfigRow struct {
	CommentID         string
	InstanceID        string
	Region            string
	Platform          string
	ServerURL         string
	AuthorizationID   string
	WorkspaceID       string
	LastRuntimeStatus string
	// LaunchRequestID is set when RunInstances has been issued but no instance_id
	// has returned yet (OPT-20260817-027). A terminal task must mark such rows
	// Released so a late-arriving VM is not left as an orphan.
	LaunchRequestID string
}

// TaskRuntimeList is the list-by-task response: task-level counts + comment CSCs.
type TaskRuntimeList struct {
	RunningMachineCount   int
	RunningContainerCount int
	Comments              []ConfigRow
}

func serviceBase() string {
	if v := strings.TrimSpace(os.Getenv("TASK_CLOUD_SERVICE_BASE_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	// OPT-20260821-020: 同节点部署时 cloud 服务与 taskEvents 在同一内网主机，
	// 默认走 INFRA_HOST（与 conf/events/domain-events/docker-infra.yaml 的 ${INFRA_HOST:-10.2.150.68} 一致），
	// 禁止把跨进程连接写死成 loopback-only（拆节点/容器未监听时必然 connection refused）。
	host := strings.TrimSpace(os.Getenv("INFRA_HOST"))
	if host == "" {
		host = "10.2.150.68"
	}
	return "http://" + host + ":8018"
}

func postJSON(path string, payload map[string]interface{}) (int, map[string]interface{}, error) {
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, serviceBase()+path, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]interface{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	return resp.StatusCode, out, nil
}

func getJSON(path string) (int, map[string]interface{}, error) {
	req, err := http.NewRequest(http.MethodGet, serviceBase()+path, nil)
	if err != nil {
		return 0, nil, err
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]interface{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	return resp.StatusCode, out, nil
}

// UpsertAfterStart writes CloudServerConfig to taskCloudService SSOT.
func UpsertAfterStart(companyID int64, workspaceID, taskID, authID, verificationSecret string, in StartVMResult) error {
	ts := time.Now().UTC().Format(time.RFC3339)
	clientToken := strings.TrimSpace(in.ClientToken)
	if clientToken == "" {
		clientToken = "testToken"
	}
	cfgID := strings.TrimSpace(in.CSCID)
	if cfgID == "" {
		cfgID = fmt.Sprintf("csc_%d", snowflake.GenerateID())
	}
	lookupQ := fmt.Sprintf("/api/internal/cloud-server-config/lookup/?tenant_id=%s&workspace_id=%s&task_id=%s",
		fmt.Sprint(companyID), workspaceID, taskID)
	if cid := strings.TrimSpace(in.CommentID); cid != "" {
		lookupQ += "&comment_id=" + cid
	}
	if sid := strings.TrimSpace(in.CSCID); sid != "" {
		lookupQ += "&csc_id=" + sid
	}
	if status, body, lerr := getJSON(lookupQ); lerr == nil && status == http.StatusOK && body != nil {
		if id := strField(body, "id"); id != "" {
			cfgID = id
		}
	}
	row := map[string]interface{}{
		"id":                cfgID,
		"company_id":        fmt.Sprint(companyID),
		"workspace_id":      workspaceID,
		"task_id":           taskID,
		"comment_id":        strings.TrimSpace(in.CommentID),
		"platform":          in.Platform,
		"instance_id":       in.InstanceID,
		"security_group_id": in.SecurityGroupID,
		"vswitch_id":        in.VSwitchID,
		"region":            in.Region,
		"zone_id":           in.ZoneID,
		"authorization_id":  authID,
		"public_ip":         in.PublicIP,
		"server_url":        in.ServerURL,
		"launch_request_id": in.LaunchRequestID,
		"client_token":      clientToken,
		"updated_at":        ts,
		"created_at":        ts,
	}
	if strings.TrimSpace(verificationSecret) != "" {
		row["verification_secret"] = verificationSecret
	}
	status, _, err := postJSON("/api/internal/cloud-server-config/import/", map[string]interface{}{
		"configs": []map[string]interface{}{row},
	})
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("taskCloudService import config status %d", status)
	}
	return nil
}

// InsertHistoryAfterStart appends a history row in taskCloudService.
// runtimeSource is the start entry code (e.g. cloud_vm_auto_run); empty defaults to cloud_vm.
func InsertHistoryAfterStart(companyID int64, workspaceID, taskID, authID string, hw map[string]interface{}, in StartVMResult, runtimeSource string) error {
	cpu := intField(hw, "cpu_cores", 1)
	mem := intField(hw, "memory_gb", 1)
	disk := intField(hw, "storage_gb", 40)
	instType := strField(hw, "instance_type")
	src := strings.TrimSpace(runtimeSource)
	if src == "" {
		src = "cloud_vm"
	}
	ts := time.Now().UTC().Format(time.RFC3339)
	payload := map[string]interface{}{
		"id":                fmt.Sprintf("csh_%d", snowflake.GenerateID()),
		"company_id":        fmt.Sprint(companyID),
		"workspace_id":      workspaceID,
		"task_id":           taskID,
		"platform":          in.Platform,
		"platform_id":       1,
		"instance_id":       in.InstanceID,
		"instance_type_id":  instType,
		"security_group_id": in.SecurityGroupID,
		"vswitch_id":        in.VSwitchID,
		"region":            in.Region,
		"zone_id":           in.ZoneID,
		"authorization_id":  authID,
		"cpu_cores":         cpu,
		"memory_gb":         mem,
		"storage_gb":        disk,
		"launch_request_id": in.LaunchRequestID,
		"started_at":        ts,
		"created_at":        ts,
		"public_ip":         in.PublicIP,
		"server_url":        in.ServerURL,
		"runtime_source":    src,
	}
	status, _, err := postJSON("/api/internal/cloud-server-config/histories/", payload)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("taskCloudService create history status %d", status)
	}
	return nil
}

// LoadForTask reads config from taskCloudService for stop processing.
func LoadForTask(companyID int64, workspaceID, taskID string) (*ConfigRow, error) {
	q := fmt.Sprintf("/api/internal/cloud-server-config/lookup/?tenant_id=%s&workspace_id=%s&task_id=%s",
		strings.TrimSpace(fmt.Sprint(companyID)),
		strings.TrimSpace(workspaceID),
		strings.TrimSpace(taskID),
	)
	status, body, err := getJSON(q)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		return nil, fmt.Errorf("config not found")
	}
	if status >= 400 || body == nil {
		return nil, fmt.Errorf("taskCloudService lookup status %d", status)
	}
	row := configRowFromMap(body)
	return &row, nil
}

func configRowFromMap(body map[string]interface{}) ConfigRow {
	return ConfigRow{
		CommentID:         strField(body, "comment_id"),
		InstanceID:        strField(body, "instance_id"),
		Region:            strField(body, "region"),
		Platform:          strField(body, "platform"),
		ServerURL:         strField(body, "server_url"),
		AuthorizationID:   strField(body, "authorization_id"),
		WorkspaceID:       strField(body, "workspace_id"),
		LastRuntimeStatus: strField(body, "last_runtime_status"),
		LaunchRequestID:   strField(body, "launch_request_id"),
	}
}

// ListByTask reads all comment-scoped CSCs for a task (template row omitted).
func ListByTask(companyID int64, workspaceID, taskID string) (*TaskRuntimeList, error) {
	q := fmt.Sprintf("/api/internal/cloud-server-config/list-by-task/?tenant_id=%s&workspace_id=%s&task_id=%s",
		strings.TrimSpace(fmt.Sprint(companyID)),
		strings.TrimSpace(workspaceID),
		strings.TrimSpace(taskID),
	)
	status, body, err := getJSON(q)
	if err != nil {
		return nil, err
	}
	if status >= 400 || body == nil {
		return nil, fmt.Errorf("taskCloudService list-by-task status %d", status)
	}
	out := &TaskRuntimeList{
		RunningMachineCount:   intField(body, "running_machine_count", 0),
		RunningContainerCount: intField(body, "running_container_count", 0),
		Comments:              nil,
	}
	raw, _ := body["comments"].([]interface{})
	for _, item := range raw {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		out.Comments = append(out.Comments, configRowFromMap(m))
	}
	if out.Comments == nil {
		out.Comments = []ConfigRow{}
	}
	return out, nil
}

// MarkCommentTerminalReleased asks taskCloudService to mark a comment-scoped CSC
// terminal_released=1 + last_runtime_status='Released'. Used when a task goes
// terminal while RunInstances is still in flight (launch_request_id set, no
// instance_id yet) so a late-arriving VM is not left as an orphan (OPT-20260817-027).
func MarkCommentTerminalReleased(tenantID, workspaceID, taskID, commentID string) error {
	payload := map[string]interface{}{
		"tenant_id":    tenantID,
		"workspace_id": workspaceID,
		"task_id":      taskID,
		"comment_id":   commentID,
	}
	status, _, err := postJSON("/api/internal/cloud/compute/mark-comment-terminal-released/", payload)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("taskCloudService mark-comment-terminal-released status %d", status)
	}
	return nil
}

// ClearAfterStop asks taskCloudService to clear reachability natively.
// instanceID scopes history close / CSC clear to the deleted instance when non-empty.
// stopRequestID (CLOUD_SERVER_STOPPED 载荷) 非空时由 owner 侧落库为唯一键（OPT-20260818-015）。
func ClearAfterStop(tenantID, workspaceID, taskID, reason, instanceID, stopRequestID string) error {
	payload := map[string]interface{}{
		"tenant_id":   tenantID,
		"task_id":     taskID,
		"stop_reason": reason,
	}
	if strings.TrimSpace(workspaceID) != "" {
		payload["workspace_id"] = workspaceID
	}
	if id := strings.TrimSpace(instanceID); id != "" {
		payload["instance_id"] = id
	}
	if id := strings.TrimSpace(stopRequestID); id != "" {
		payload["stop_request_id"] = id
	}
	status, _, err := postJSON("/api/internal/cloud-server-config/clear-after-stop/", payload)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("taskCloudService clear-after-stop status %d", status)
	}
	return nil
}

// StopRequestProcessed checks whether a CLOUD_SERVER_STOPPED stop_request_id has already
// been processed by the owner DB (cloud_stop_request unique key). Fail-open: transient
// errors return false so a real stop is never skipped.
func StopRequestProcessed(stopRequestID string) (bool, error) {
	stopRequestID = strings.TrimSpace(stopRequestID)
	if stopRequestID == "" {
		return false, nil
	}
	status, body, err := getJSON("/api/internal/cloud/stops/processed?stop_request_id=" + stopRequestID)
	if err != nil {
		return false, err
	}
	if status >= 400 || body == nil {
		return false, fmt.Errorf("taskCloudService stop processed status %d", status)
	}
	processed, _ := body["processed"].(bool)
	return processed, nil
}

func strField(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}

func intField(m map[string]interface{}, key string, def int) int {
	if m == nil {
		return def
	}
	v, ok := m[key]
	if !ok || v == nil {
		return def
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		var out int
		if _, err := fmt.Sscanf(fmt.Sprintf("%v", v), "%d", &out); err == nil {
			return out
		}
	}
	return def
}
