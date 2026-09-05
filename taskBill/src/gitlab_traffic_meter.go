package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	"tracelog"
)

type gitlabTrafficMeterReq struct {
	TenantID       int64
	Bytes          int64
	IsIntranet     bool
	FromCI         bool
	IdempotencyKey string
	Region         string
	RepoURL        string
	ProjectPath    string
	GitlabUsername string
	TaskID         string
	UserID         string
	WorkspaceID    string
	ProjectID      string
}

func gitlabHostFromURL(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	// git@host:group/repo.git — url.Parse 会把 :group 当成端口
	if strings.HasPrefix(s, "git@") && !strings.Contains(s, "://") {
		rest := strings.TrimPrefix(s, "git@")
		host, _, found := strings.Cut(rest, ":")
		if !found {
			return strings.ToLower(strings.TrimSpace(rest))
		}
		return strings.ToLower(strings.TrimSpace(host))
	}
	if !strings.Contains(s, "://") {
		s = "https://" + s
	}
	u, err := url.Parse(s)
	if err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(u.Hostname()))
}

func gitlabRegionSlugFromRepoURL(repoURL string) (string, error) {
	host := gitlabHostFromURL(repoURL)
	if host == "" {
		return "", nil
	}
	regions, err := listGitlabRegions()
	if err != nil {
		return "", err
	}
	for _, r := range regions {
		if gitlabHostFromURL(r.GitlabWebURL) == host {
			return r.Slug, nil
		}
	}
	return "", nil
}

func meterSkip(reason string, gb float64) map[string]interface{} {
	return map[string]interface{}{
		"skipped": true,
		"reason":  reason,
		"gb":      gb,
		"cost":    int64(0),
	}
}

func resolveGitlabTrafficMeterTenant(in gitlabTrafficMeterReq) int64 {
	if in.TenantID > 0 {
		return in.TenantID
	}
	if parsed, err := parseTenantIDFromGitlabProjectPath(in.ProjectPath); err == nil {
		return parsed
	}
	if in.FromCI || in.IsIntranet {
		return 0
	}
	return resolveTenantByUsername(gitlabTrafficGateRequest{
		ProjectPath:    in.ProjectPath,
		GitlabUsername: in.GitlabUsername,
		Region:         in.Region,
		FromCI:         in.FromCI,
		IsIntranet:     in.IsIntranet,
	}, in.Region)
}

func gitlabTrafficMeterReqFromJSON(body map[string]interface{}) (gitlabTrafficMeterReq, error) {
	in := gitlabTrafficMeterReq{
		IsIntranet:     boolField(body, "is_intranet") || boolField(body, "same_region_intranet"),
		FromCI:         boolField(body, "from_ci"),
		IdempotencyKey: stringField(body, "idempotency_key"),
		Region:         stringField(body, "region"),
		RepoURL:        stringField(body, "repo_url"),
		ProjectPath:    stringField(body, "project_path"),
		GitlabUsername: stringField(body, "gitlab_username"),
		TaskID:         stringField(body, "task_id"),
		UserID:         stringField(body, "user_id"),
		WorkspaceID:    stringField(body, "workspace_id"),
		ProjectID:      stringField(body, "project_id"),
	}
	if tid, err := parseIDField(body["tenant_id"]); err == nil {
		in.TenantID = tid
	}
	in.TenantID = resolveGitlabTrafficMeterTenant(in)
	if in.TenantID <= 0 && strings.TrimSpace(in.ProjectPath) == "" {
		return in, fmt.Errorf("须提供 tenant_id 或 project_path")
	}
	hasBytes := body["bytes"] != nil
	hasGB := body["gb"] != nil
	if !hasBytes && !hasGB {
		return in, fmt.Errorf("须提供 gb 或 bytes")
	}
	if hasBytes {
		n, e := parseNonNegInt64(body["bytes"], "bytes")
		if e != nil {
			return in, e
		}
		in.Bytes = n
	} else if v, e := parseFloat64Field(body["gb"]); e == nil {
		in.Bytes = gbToTrafficBytes(v)
	} else {
		return in, fmt.Errorf("gb 无效")
	}
	return in, nil
}

// meterGitlabOutboundTraffic increments traffic_used_gb by measured clone/fetch
// bytes (6 decimal GB). Does not ceil to 1 GB and does not require unit price.
func meterGitlabOutboundTraffic(ctx context.Context, in gitlabTrafficMeterReq) (map[string]interface{}, error) {
	if in.FromCI {
		return meterSkip("ci_skip", 0), nil
	}
	if in.IsIntranet {
		return meterSkip("same_region_intranet", 0), nil
	}
	if in.Bytes <= 0 {
		return meterSkip("zero_usage", 0), nil
	}
	if in.TenantID <= 0 {
		return meterSkip("unmapped_project", 0), nil
	}
	region := strings.TrimSpace(in.Region)
	repoURL := strings.TrimSpace(in.RepoURL)
	if repoURL != "" {
		slug, err := gitlabRegionSlugFromRepoURL(repoURL)
		if err != nil {
			return nil, err
		}
		if slug == "" {
			return meterSkip("not_platform_gitlab", 0), nil
		}
		region = slug
	}
	if region == "" {
		return meterSkip("missing_region", 0), nil
	}
	meterGB := diskUsedGBFromBytes(in.Bytes)
	if meterGB <= 0 {
		return meterSkip("zero_usage", 0), nil
	}
	if err := expireCreditLots(ctx, in.TenantID); err != nil {
		return nil, err
	}
	acc, _, err := getOrCreateBillingAccount(in.TenantID, true)
	if err != nil {
		return nil, err
	}
	res, resErr := getTenantGitlabResourceByRegion(in.TenantID, region)
	if resErr != nil {
		return nil, resErr
	}
	billed, _ := sumGitlabTrafficUsedGB(acc.ID)
	used := gitlabTrafficUsedGB(res, billed)
	if allowed, code := gitlabTrafficDownloadAllowed(res.TrafficPrepaidGB, used, false, false); !allowed {
		return nil, &TrafficQuotaExceededError{
			PrepaidGB: float64(res.TrafficPrepaidGB),
			UsedGB:    used,
			Code:      code,
		}
	}
	meterKey := ""
	if ik := strings.TrimSpace(in.IdempotencyKey); ik != "" {
		meterKey = "gitlab-traffic-meter:" + ik
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if meterKey != "" {
		var existing int64
		err := tx.QueryRow(`SELECT transaction_id FROM billing_idempotency_key WHERE `+"`"+`key`+"`"+` = ?`, meterKey).Scan(&existing)
		if err == nil {
			return meterSkip("idempotent", meterGB), nil
		}
		if err != sql.ErrNoRows {
			return nil, err
		}
	}
	now := utcNow()
	if _, err := tx.Exec(`
		INSERT INTO billing_tenant_gitlab_resource (
			tenant_id, region, disk_gb, traffic_prepaid_gb, disk_months, disk_expires_at,
			disk_used_bytes, traffic_used_gb, provisioning_status, created_at, updated_at
		) VALUES (?, ?, 0, 0, 0, '', 0, ?, 'not_purchased', ?, ?)
		ON DUPLICATE KEY UPDATE
			traffic_used_gb = traffic_used_gb + VALUES(traffic_used_gb),
			updated_at = VALUES(updated_at)`,
		in.TenantID, region, meterGB, now, now,
	); err != nil {
		return nil, err
	}
	eventID := generateSnowflakeID()
	if meterKey != "" {
		if _, err := tx.Exec(`
			INSERT INTO billing_idempotency_key (id, `+"`"+`key`+"`"+`, created_at, transaction_id)
			VALUES (?, ?, ?, ?)`, generateSnowflakeID(), meterKey, now, eventID); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	tracelog.LogForwardStage(ctx, "gitlab_traffic_metered", map[string]any{
		"tenant_id":       formatID(in.TenantID),
		"account_id":      formatID(acc.ID),
		"gb":              meterGB,
		"bytes":           in.Bytes,
		"region":          region,
		"task_id":         in.TaskID,
		"user_id":         in.UserID,
		"workspace_id":    in.WorkspaceID,
		"project_id":      in.ProjectID,
		"project_path":    in.ProjectPath,
		"idempotency_key": in.IdempotencyKey,
	})
	return map[string]interface{}{
		"gb":             meterGB,
		"bytes":          in.Bytes,
		"region":         region,
		"skipped":        false,
		"cost":           int64(0),
		"wallet_debited": false,
	}, nil
}

func gbToTrafficBytes(gb float64) int64 {
	if gb <= 0 {
		return 0
	}
	return int64(gb*bytesPerGiB + 0.5)
}

func chargeGitlabTrafficFromMeter(
	ctx context.Context,
	tenantID int64,
	gb float64,
	isIntranet bool,
	idempotencyKey, projectID, userID, workspaceID, taskID, region, repoURL string,
) (map[string]interface{}, error) {
	if gb < 0 {
		return nil, fmt.Errorf("gb must be >= 0")
	}
	return meterGitlabOutboundTraffic(ctx, gitlabTrafficMeterReq{
		TenantID:       tenantID,
		Bytes:          gbToTrafficBytes(gb),
		IsIntranet:     isIntranet,
		IdempotencyKey: idempotencyKey,
		Region:         region,
		RepoURL:        repoURL,
		TaskID:         taskID,
		UserID:         userID,
		WorkspaceID:    workspaceID,
		ProjectID:      projectID,
	})
}
