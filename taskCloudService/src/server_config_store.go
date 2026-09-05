package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type CloudServerConfig struct {
	ID                  string
	CompanyID           string
	WorkspaceID         string
	TaskID              string
	CommentID           string // 空=任务硬件模板（非运行实例）；非空=评论级独立实例
	Platform            string
	InstanceID          string
	SecurityGroupID     string
	VswitchID           string
	Region              string
	ZoneID              string
	AuthorizationID     string
	PublicIP            string
	ServerURL           string
	BusinessAPIEndpoint string
	ContainerVscodeURL  string
	ErrorReason         string
	LaunchRequestID     string
	ClientToken         string
	LastRuntimeStatus   string
	ImageInvokerUserID  string // 评论人员 / 镜像调用人员（审计；闲置复用已下线）
	LastHeartbeatAt     string // 最近一次容器心跳时间（用于冷打开 UI 判断双向已连接）
	VerificationSecret  string // UserData 一次性验证 secret
	UserdataRunVerified string // UserData 验证通过时间
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type CloudServerConfigHistory struct {
	ID                  string
	CompanyID           string
	WorkspaceID         string
	TaskID              string
	Platform            string
	PlatformID          int
	InstanceID          string
	InstanceTypeID      string
	SecurityGroupID     string
	VswitchID           string
	Region              string
	ZoneID              string
	AuthorizationID     string
	PublicIP            string
	ServerURL           string
	BusinessAPIEndpoint string
	ErrorReason         string
	StopReason          string
	RuntimeSource       string
	StartReason         string // 业务语义（manual_start / task_reuse / auto_run / mention / migrate）OPT-20260719-015
	LaunchRequestID     string
	CpuCores            int
	MemoryGB            int
	StorageGB           int
	StartedAt           time.Time
	StoppedAt           *time.Time
	CreatedAt           time.Time
}

// loadCloudServerConfig 加载任务硬件模板行（comment_id=”）。
// 仅用于复制 platform/region/auth 等到评论 CSC；禁止当作运行实例。
func loadCloudServerConfig(companyID, workspaceID, taskID string) (*CloudServerConfig, error) {
	return loadCloudServerConfigForComment(companyID, workspaceID, taskID, "")
}

// loadNewestCloudServerConfigWithInstance 返回任务下最新带非空 instance_id 的**评论级** CSC。
func loadNewestCloudServerConfigWithInstance(companyID, workspaceID, taskID string) (*CloudServerConfig, error) {
	return loadNewestCloudServerConfigForTaskFiltered(companyID, workspaceID, taskID, true)
}

// loadNewestCloudServerConfigForTask 返回任务下最新 CSC（含评论级；instance 可为空）。
// 用于评论容器已 ensure CSC、但云实例尚未回填 instance_id 时，避免 runtime 误报「未找到服务器配置记录」。
func loadNewestCloudServerConfigForTask(companyID, workspaceID, taskID string) (*CloudServerConfig, error) {
	return loadNewestCloudServerConfigForTaskFiltered(companyID, workspaceID, taskID, false)
}

func loadNewestCloudServerConfigForTaskFiltered(companyID, workspaceID, taskID string, requireInstance bool) (*CloudServerConfig, error) {
	companyID = trim(companyID)
	workspaceID = trim(workspaceID)
	taskID = trim(taskID)
	if taskID == "" {
		return nil, sql.ErrNoRows
	}
	q := `SELECT id, company_id, workspace_id, task_id, COALESCE(comment_id,''), platform, instance_id,
		COALESCE(security_group_id,''), COALESCE(vswitch_id,''), region, zone_id,
		authorization_id, public_ip, server_url, business_api_endpoint,
		COALESCE(container_vscode_url,''), COALESCE(error_reason,''), COALESCE(launch_request_id,''), client_token,
		COALESCE(last_runtime_status,''), COALESCE(image_invoker_user_id,''),
		COALESCE(verification_secret,''), COALESCE(userdata_run_verified,''),
		created_at, updated_at
		FROM cloud_server_configs
		WHERE task_id=?`
	args := []interface{}{taskID}
	if requireInstance {
		q += ` AND TRIM(COALESCE(instance_id,'')) != ''`
	}
	q += ` AND TRIM(COALESCE(comment_id,'')) != ''`
	if workspaceID != "" {
		q += ` AND workspace_id=?`
		args = append(args, workspaceID)
	}
	if companyID != "" {
		q += ` AND company_id=?`
		args = append(args, companyID)
	}
	q += ` ORDER BY updated_at DESC LIMIT 1`
	return scanCloudServerConfig(db.QueryRow(q, args...))
}

func loadCloudServerConfigByID(id string) (*CloudServerConfig, error) {
	id = trim(id)
	if id == "" {
		return nil, sql.ErrNoRows
	}
	if cloudServerConfigByIDLoadHook != nil {
		cloudServerConfigByIDLoadHook(id)
	}
	row := db.QueryRow(`SELECT `+cloudServerConfigSelectColumns+`
		FROM cloud_server_configs WHERE id=?`, id)
	return scanCloudServerConfig(row)
}

// cloudServerConfigByIDLoadHook 仅测试注入：统计单条加载次数，证明 list JSON 不再 N+1。
var cloudServerConfigByIDLoadHook func(id string)

// cloudServerConfigsByIDsLoadHook 仅测试注入：记录批量 IN 查询的去重 id 数。
var cloudServerConfigsByIDsLoadHook func(ids []string)

const cloudServerConfigSelectColumns = `id, company_id, workspace_id, task_id, COALESCE(comment_id,''), platform, instance_id,
		COALESCE(security_group_id,''), COALESCE(vswitch_id,''), region, zone_id,
		authorization_id, public_ip, server_url, business_api_endpoint,
		COALESCE(container_vscode_url,''), COALESCE(error_reason,''), COALESCE(launch_request_id,''), client_token,
		COALESCE(last_runtime_status,''), COALESCE(image_invoker_user_id,''),
		COALESCE(verification_secret,''), COALESCE(userdata_run_verified,''),
		created_at, updated_at`

func uniqueNonEmptyIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = trim(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// loadCloudServerConfigsByIDs 一次 IN 查询加载多条 CSC，避免 list binding 时逐条 QueryRow。
func loadCloudServerConfigsByIDs(ids []string) (map[string]*CloudServerConfig, error) {
	uniq := uniqueNonEmptyIDs(ids)
	if cloudServerConfigsByIDsLoadHook != nil {
		cloudServerConfigsByIDsLoadHook(uniq)
	}
	out := make(map[string]*CloudServerConfig, len(uniq))
	if len(uniq) == 0 {
		return out, nil
	}
	placeholders := make([]string, len(uniq))
	args := make([]interface{}, len(uniq))
	for i, id := range uniq {
		placeholders[i] = "?"
		args[i] = id
	}
	rows, err := db.Query(
		`SELECT `+cloudServerConfigSelectColumns+`
		FROM cloud_server_configs WHERE id IN (`+strings.Join(placeholders, ",")+`)`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		cfg, err := scanCloudServerConfigFields(rows)
		if err != nil {
			return nil, err
		}
		if cfg == nil || trim(cfg.ID) == "" {
			continue
		}
		out[cfg.ID] = cfg
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// loadCloudServerConfigByInstanceForTask 按 instance_id 加载运行行；同实例多行时优先评论级。
func loadCloudServerConfigByInstanceForTask(companyID, workspaceID, taskID, instanceID string) (*CloudServerConfig, error) {
	companyID = trim(companyID)
	workspaceID = trim(workspaceID)
	taskID = trim(taskID)
	instanceID = trim(instanceID)
	if taskID == "" || instanceID == "" {
		return nil, sql.ErrNoRows
	}
	q := `SELECT id, company_id, workspace_id, task_id, COALESCE(comment_id,''), platform, instance_id,
		COALESCE(security_group_id,''), COALESCE(vswitch_id,''), region, zone_id,
		authorization_id, public_ip, server_url, business_api_endpoint,
		COALESCE(container_vscode_url,''), COALESCE(error_reason,''), COALESCE(launch_request_id,''), client_token,
		COALESCE(last_runtime_status,''), COALESCE(image_invoker_user_id,''),
		COALESCE(verification_secret,''), COALESCE(userdata_run_verified,''),
		created_at, updated_at
		FROM cloud_server_configs WHERE task_id=? AND TRIM(COALESCE(instance_id,''))=?`
	args := []interface{}{taskID, instanceID}
	if workspaceID != "" {
		q += ` AND workspace_id=?`
		args = append(args, workspaceID)
	}
	if companyID != "" {
		q += ` AND company_id=?`
		args = append(args, companyID)
	}
	q += ` ORDER BY CASE WHEN TRIM(COALESCE(comment_id,''))='' THEN 1 ELSE 0 END, updated_at DESC LIMIT 1`
	return scanCloudServerConfig(db.QueryRow(q, args...))
}

// loadCloudServerConfigByPublicIPForTask 按公网 IP 解析评论级 CSC。
// 存量 onlineServiceJS 的 register-reachability / 心跳可能不带 comment_id，
// 但会带 public_ip 或可从 server_url 解析出主机。
func loadCloudServerConfigByPublicIPForTask(companyID, workspaceID, taskID, publicIP string) (*CloudServerConfig, error) {
	companyID = trim(companyID)
	workspaceID = trim(workspaceID)
	taskID = trim(taskID)
	publicIP = trim(publicIP)
	if taskID == "" || publicIP == "" {
		return nil, sql.ErrNoRows
	}
	q := `SELECT id, company_id, workspace_id, task_id, COALESCE(comment_id,''), platform, instance_id,
		COALESCE(security_group_id,''), COALESCE(vswitch_id,''), region, zone_id,
		authorization_id, public_ip, server_url, business_api_endpoint,
		COALESCE(container_vscode_url,''), COALESCE(error_reason,''), COALESCE(launch_request_id,''), client_token,
		COALESCE(last_runtime_status,''), COALESCE(image_invoker_user_id,''),
		COALESCE(verification_secret,''), COALESCE(userdata_run_verified,''),
		created_at, updated_at
		FROM cloud_server_configs
		WHERE task_id=? AND TRIM(COALESCE(comment_id,'')) != '' AND TRIM(COALESCE(public_ip,''))=?`
	args := []interface{}{taskID, publicIP}
	if workspaceID != "" {
		q += ` AND workspace_id=?`
		args = append(args, workspaceID)
	}
	if companyID != "" {
		q += ` AND company_id=?`
		args = append(args, companyID)
	}
	q += ` ORDER BY updated_at DESC LIMIT 1`
	return scanCloudServerConfig(db.QueryRow(q, args...))
}

// loadUniqueCommentCloudServerConfigWithInstance 仅当任务下恰好一条带 instance 的评论级 CSC 时返回。
func loadUniqueCommentCloudServerConfigWithInstance(companyID, workspaceID, taskID string) (*CloudServerConfig, int, error) {
	companyID = trim(companyID)
	workspaceID = trim(workspaceID)
	taskID = trim(taskID)
	if taskID == "" {
		return nil, 0, sql.ErrNoRows
	}
	countQ := `SELECT COUNT(*) FROM cloud_server_configs
		WHERE task_id=? AND TRIM(COALESCE(comment_id,'')) != '' AND TRIM(COALESCE(instance_id,'')) != ''`
	countArgs := []interface{}{taskID}
	if workspaceID != "" {
		countQ += ` AND workspace_id=?`
		countArgs = append(countArgs, workspaceID)
	}
	if companyID != "" {
		countQ += ` AND company_id=?`
		countArgs = append(countArgs, companyID)
	}
	var n int
	if err := db.QueryRow(countQ, countArgs...).Scan(&n); err != nil {
		return nil, 0, err
	}
	if n != 1 {
		return nil, n, nil
	}
	cfg, err := loadNewestCloudServerConfigWithInstance(companyID, workspaceID, taskID)
	return cfg, n, err
}

func cscHasReachableEndpoint(csc *CloudServerConfig) bool {
	if csc == nil {
		return false
	}
	return trim(csc.ServerURL) != "" || trim(csc.BusinessAPIEndpoint) != ""
}

// loadCloudServerConfigForComment 按 workspace_id + task_id + comment_id 加载。
// comment_id 空串仅表示任务硬件模板行，禁止当作运行实例。
func loadCloudServerConfigForComment(companyID, workspaceID, taskID, commentID string) (*CloudServerConfig, error) {
	companyID = trim(companyID)
	workspaceID = trim(workspaceID)
	taskID = trim(taskID)
	commentID = trim(commentID)
	if taskID == "" {
		return nil, sql.ErrNoRows
	}
	q := `SELECT id, company_id, workspace_id, task_id, COALESCE(comment_id,''), platform, instance_id,
		COALESCE(security_group_id,''), COALESCE(vswitch_id,''), region, zone_id,
		authorization_id, public_ip, server_url, business_api_endpoint,
		COALESCE(container_vscode_url,''), COALESCE(error_reason,''), COALESCE(launch_request_id,''), client_token,
		COALESCE(last_runtime_status,''), COALESCE(image_invoker_user_id,''),
		COALESCE(verification_secret,''), COALESCE(userdata_run_verified,''),
		created_at, updated_at
		FROM cloud_server_configs WHERE task_id=? AND COALESCE(comment_id,'')=?`
	args := []interface{}{taskID, commentID}
	if workspaceID != "" {
		q += ` AND workspace_id=?`
		args = append(args, workspaceID)
	}
	if companyID != "" {
		q += ` AND company_id=?`
		args = append(args, companyID)
	}
	q += ` ORDER BY updated_at DESC LIMIT 1`

	row := db.QueryRow(q, args...)
	return scanCloudServerConfig(row)
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanCloudServerConfig(row *sql.Row) (*CloudServerConfig, error) {
	return scanCloudServerConfigFields(row)
}

func scanCloudServerConfigFields(row rowScanner) (*CloudServerConfig, error) {
	var c CloudServerConfig
	var created, updated string
	err := row.Scan(
		&c.ID, &c.CompanyID, &c.WorkspaceID, &c.TaskID, &c.CommentID, &c.Platform, &c.InstanceID,
		&c.SecurityGroupID, &c.VswitchID, &c.Region, &c.ZoneID, &c.AuthorizationID, &c.PublicIP, &c.ServerURL,
		&c.BusinessAPIEndpoint, &c.ContainerVscodeURL, &c.ErrorReason, &c.LaunchRequestID, &c.ClientToken,
		&c.LastRuntimeStatus, &c.ImageInvokerUserID,
		&c.VerificationSecret, &c.UserdataRunVerified,
		&created, &updated,
	)
	if err != nil {
		return nil, err
	}
	c.CreatedAt = parseCloudUTCDateTime(created)
	c.UpdatedAt = parseCloudUTCDateTime(updated)
	return &c, nil
}

func cloudServerConfigToJSON(c *CloudServerConfig) map[string]interface{} {
	if c == nil {
		return nil
	}
	return map[string]interface{}{
		"id":                    c.ID,
		"company_id":            c.CompanyID,
		"workspace_id":          c.WorkspaceID,
		"task_id":               c.TaskID,
		"comment_id":            c.CommentID,
		"platform":              c.Platform,
		"instance_id":           c.InstanceID,
		"security_group_id":     c.SecurityGroupID,
		"vswitch_id":            c.VswitchID,
		"region":                c.Region,
		"zone_id":               c.ZoneID,
		"authorization_id":      c.AuthorizationID,
		"public_ip":             c.PublicIP,
		"server_url":            c.ServerURL,
		"business_api_endpoint": c.BusinessAPIEndpoint,
		"container_vscode_url":  c.ContainerVscodeURL,
		"error_reason":          c.ErrorReason,
		"launch_request_id":     c.LaunchRequestID,
		"client_token":          c.ClientToken,
		"last_runtime_status":   c.LastRuntimeStatus,
		"image_invoker_user_id": c.ImageInvokerUserID,
		"created_at":            formatCloudUTCJSON(c.CreatedAt),
		"updated_at":            formatCloudUTCJSON(c.UpdatedAt),
	}
}

func setCloudServerConfigVerificationSecret(companyID, workspaceID, taskID, secret string) error {
	companyID = trim(companyID)
	workspaceID = trim(workspaceID)
	taskID = trim(taskID)
	secret = trim(secret)
	if taskID == "" || secret == "" {
		return fmt.Errorf("task_id and secret required")
	}
	_, err := db.Exec(`
		UPDATE cloud_server_configs
		SET verification_secret = ?, updated_at = CURRENT_TIMESTAMP
		WHERE company_id = ? AND workspace_id = ? AND task_id = ? AND COALESCE(comment_id,'') = ''
	`, secret, companyID, workspaceID, taskID)
	return err
}

func verifyCloudServerUserdata(companyID, workspaceID, taskID, secret string) (bool, error) {
	companyID = trim(companyID)
	workspaceID = trim(workspaceID)
	taskID = trim(taskID)
	secret = trim(secret)
	if taskID == "" || secret == "" {
		return false, fmt.Errorf("task_id and secret required")
	}
	res, err := db.Exec(`
		UPDATE cloud_server_configs
		SET userdata_run_verified = CURRENT_TIMESTAMP,
		    verification_secret = '',
		    updated_at = CURRENT_TIMESTAMP
		WHERE company_id = ? AND workspace_id = ? AND task_id = ?
		  AND verification_secret = ? AND verification_secret != ''
		  AND userdata_run_verified IS NULL
	`, companyID, workspaceID, taskID, secret)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func upsertCloudServerConfig(c CloudServerConfig) error {
	now := time.Now().UTC()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	c.UpdatedAt = now
	_, err := db.Exec(
		`INSERT INTO cloud_server_configs(
			id, company_id, workspace_id, task_id, comment_id, platform, instance_id,
			security_group_id, vswitch_id, region, zone_id, authorization_id,
			public_ip, server_url, business_api_endpoint, container_vscode_url,
			error_reason, launch_request_id, client_token, last_runtime_status,
			image_invoker_user_id, created_at, updated_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON DUPLICATE KEY UPDATE
			company_id=VALUES(company_id),
			workspace_id=VALUES(workspace_id),
			task_id=VALUES(task_id),
			comment_id=VALUES(comment_id),
			platform=VALUES(platform),`+cloudServerRuntimeUpsertPreserve+`
			security_group_id=VALUES(security_group_id),
			vswitch_id=VALUES(vswitch_id),
			region=VALUES(region),
			zone_id=VALUES(zone_id),
			authorization_id=VALUES(authorization_id),
			business_api_endpoint=VALUES(business_api_endpoint),
			container_vscode_url=VALUES(container_vscode_url),
			error_reason=VALUES(error_reason),
			launch_request_id=VALUES(launch_request_id),
			client_token=VALUES(client_token),
			image_invoker_user_id=CASE
				WHEN TRIM(COALESCE(VALUES(image_invoker_user_id),'')) != ''
				THEN VALUES(image_invoker_user_id)
				ELSE cloud_server_configs.image_invoker_user_id
			END,
			updated_at=VALUES(updated_at)`,
		c.ID, c.CompanyID, c.WorkspaceID, c.TaskID, c.CommentID, c.Platform, c.InstanceID,
		c.SecurityGroupID, c.VswitchID, c.Region, c.ZoneID, c.AuthorizationID,
		c.PublicIP, c.ServerURL, c.BusinessAPIEndpoint, c.ContainerVscodeURL,
		c.ErrorReason, c.LaunchRequestID, c.ClientToken, c.LastRuntimeStatus,
		c.ImageInvokerUserID, formatMySQLUTCDateTime(c.CreatedAt), formatMySQLUTCDateTime(c.UpdatedAt),
	)
	return err
}

func trim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
