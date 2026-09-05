package main

import (
	"snowflake"
	"fmt"
)

func createAccessKeyIAMAssociation(authID, accessKey, iamID string) error {
	authID = trim(authID)
	accessKey = trim(accessKey)
	iamID = trim(iamID)
	if authID == "" || accessKey == "" || iamID == "" {
		return fmt.Errorf("cloud_platform_auth_id, access_key and iam_id required")
	}
	_, err := importAccessKeyIAMAssociations([]map[string]interface{}{
		{
			"cloud_platform_auth_id": authID,
			"access_key":             accessKey,
			"iam_id":                 iamID,
		},
	})
	return err
}

func importAccessKeyIAMAssociations(rows []map[string]interface{}) (int, error) {
	count := 0
	for _, row := range rows {
		id := strField(row, "id")
		authID := strField(row, "cloud_platform_auth_id")
		if authID == "" {
			authID = strField(row, "auth_id")
		}
		accessKey := strField(row, "access_key")
		iamID := strField(row, "iam_id")
		if authID == "" || accessKey == "" || iamID == "" {
			continue
		}
		if id == "" {
			id = snowflake.GenerateIDString()
		}
		// OPT-20260818-015: REPLACE INTO 以新 snowflake 重复插行 → 改 INSERT ON DUPLICATE KEY
		// UPDATE，配合 027 迁移的 UNIQUE(cloud_platform_auth_id, access_key) 兜底：
		// 重放的授权事件更新既有关联行而非新增重复行（id 保留原行）。
		_, err := db.Exec(`INSERT INTO cloud_access_key_iam_associations
			(id, cloud_platform_auth_id, access_key, iam_id, created_at, updated_at)
			VALUES (?, ?, ?, ?, COALESCE(NULLIF(?, ''), CURRENT_TIMESTAMP), COALESCE(NULLIF(?, ''), CURRENT_TIMESTAMP))
			ON DUPLICATE KEY UPDATE
				iam_id = VALUES(iam_id),
				updated_at = CURRENT_TIMESTAMP`,
			id, authID, accessKey, iamID,
			strField(row, "created_at"), strField(row, "updated_at"),
		)
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}
