package main

import (
	"database/sql"
	"net/http"
	"strings"
	"time"
)

type cloudAuthRecord struct {
	ID                string
	PlatformType      string
	AuthorizationType string
	SecretID          string
	SecretKey         string
	Remark            string
	CompanyID         string
	Active            bool
	OAuthTokenID      string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func loadCloudAuthByID(authID string) (*cloudAuthRecord, error) {
	row := db.QueryRow(`SELECT id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active,oauth_token_id,created_at,updated_at
		FROM cloud_platform_authorizations WHERE id=?`, authID)
	var rec cloudAuthRecord
	var active int
	if err := row.Scan(&rec.ID, &rec.PlatformType, &rec.AuthorizationType, &rec.SecretID, &rec.SecretKey,
		&rec.Remark, &rec.CompanyID, &active, &rec.OAuthTokenID, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	rec.Active = active != 0
	return &rec, nil
}

func loadCloudAuth(tenantID, authID string) (*cloudAuthRecord, error) {
	if tenantID == "" {
		return loadCloudAuthByID(authID)
	}
	row := db.QueryRow(`SELECT id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active,oauth_token_id,created_at,updated_at
		FROM cloud_platform_authorizations WHERE id=? AND company_id=?`, authID, tenantID)
	var rec cloudAuthRecord
	var active int
	if err := row.Scan(&rec.ID, &rec.PlatformType, &rec.AuthorizationType, &rec.SecretID, &rec.SecretKey,
		&rec.Remark, &rec.CompanyID, &active, &rec.OAuthTokenID, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	rec.Active = active != 0
	return &rec, nil
}

func loadFirstCloudAuthForCompany(tenantID, platformType string) (*cloudAuthRecord, error) {
	if tenantID == "" || platformType == "" {
		return nil, nil
	}
	row := db.QueryRow(`SELECT id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active,oauth_token_id,created_at,updated_at
		FROM cloud_platform_authorizations WHERE company_id=? AND platform_type=? ORDER BY created_at ASC LIMIT 1`, tenantID, platformType)
	var rec cloudAuthRecord
	var active int
	if err := row.Scan(&rec.ID, &rec.PlatformType, &rec.AuthorizationType, &rec.SecretID, &rec.SecretKey,
		&rec.Remark, &rec.CompanyID, &active, &rec.OAuthTokenID, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	rec.Active = active != 0
	return &rec, nil
}

func loadFirstCloudAuthForCompanyAnyPlatform(tenantID string) (*cloudAuthRecord, error) {
	if tenantID == "" {
		return nil, nil
	}
	row := db.QueryRow(`SELECT id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active,oauth_token_id,created_at,updated_at
		FROM cloud_platform_authorizations WHERE company_id=? ORDER BY created_at ASC LIMIT 1`, tenantID)
	var rec cloudAuthRecord
	var active int
	if err := row.Scan(&rec.ID, &rec.PlatformType, &rec.AuthorizationType, &rec.SecretID, &rec.SecretKey,
		&rec.Remark, &rec.CompanyID, &active, &rec.OAuthTokenID, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	rec.Active = active != 0
	return &rec, nil
}

// resolveCloudAuthForStartVm mirrors Django start_vm_auto authorization selection.
func resolveCloudAuthForStartVm(tenantID string, body map[string]interface{}) (*cloudAuthRecord, string) {
	if tenantID == "" {
		return nil, "无法获取租户ID"
	}
	authID := normalizeAuthorizationIDField(body, "authorization_id")
	if authID != "" {
		rec, err := loadCloudAuth(tenantID, authID)
		if err != nil {
			return nil, "查询云平台授权失败"
		}
		if rec != nil {
			return rec, ""
		}
		// Explicit authorization_id missing (e.g. stale SQLite, numeric JSON precision) — fall through.
	}
	platformID := strField(body, "cloud_platform_id")
	var platformType string
	if platformID != "" {
		switch platformID {
		case "1":
			platformType = "aliyun"
		}
	}
	if platformType != "" {
		rec, err := loadFirstCloudAuthForCompany(tenantID, platformType)
		if err != nil {
			return nil, "查询云平台授权失败"
		}
		if rec == nil {
			return nil, "未配置云平台授权"
		}
		return rec, ""
	}
	rec, err := loadFirstCloudAuthForCompanyAnyPlatform(tenantID)
	if err != nil {
		return nil, "查询云平台授权失败"
	}
	if rec == nil {
		return nil, "未配置云平台授权"
	}
	return rec, ""
}

func writeCloudAuthNotFound(w http.ResponseWriter) {
	writeErrorMapJSON(w, nil, 404, map[string]interface{}{
		"status":  "error",
		"message": "未找到指定租户下的云平台授权",
	})
}

func validatePlatformTypeQuery(w http.ResponseWriter, r *http.Request, auth *cloudAuthRecord) bool {
	pt := r.URL.Query().Get("platform_type")
	if pt != "" && pt != auth.PlatformType {
		writeErrorMapJSON(w, r, 400, map[string]interface{}{
			"status":  "error",
			"message": "platform_type 与授权不匹配",
		})
		return false
	}
	return true
}

// filterChinaRegions keeps only regions whose id starts with "cn-".
// This is used by the vendor portal to limit displayed regions to China mainland regions.
func filterChinaRegions(regions []map[string]string) []map[string]string {
	if regions == nil {
		return nil
	}
	out := []map[string]string{}
	for _, r := range regions {
		id := r["id"]
		if id == "" {
			id = r["region_id"]
		}
		if strings.HasPrefix(id, "cn-") {
			out = append(out, r)
		}
	}
	return out
}

func formatRegionsIDName(regions []map[string]string) []map[string]string {
	out := []map[string]string{}
	for _, r := range regions {
		rid := r["region_id"]
		if rid == "" {
			rid = r["id"]
		}
		rname := r["region_name"]
		if rname == "" {
			rname = r["name"]
		}
		if rid != "" {
			if rname == "" {
				rname = rid
			}
			out = append(out, map[string]string{"id": rid, "name": rname})
		}
	}
	return out
}

func formatZonesIDName(zones []map[string]string) []map[string]string {
	out := []map[string]string{}
	for _, z := range zones {
		zid := z["zone_id"]
		if zid == "" {
			zid = z["id"]
		}
		zname := z["zone_name"]
		if zname == "" {
			zname = z["name"]
		}
		if zid != "" {
			if zname == "" {
				zname = zid
			}
			out = append(out, map[string]string{"id": zid, "name": zname})
		}
	}
	return out
}
