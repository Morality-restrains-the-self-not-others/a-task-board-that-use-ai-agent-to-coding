package main

import (
	"database/sql"
	"strings"
)

func peekPersonalFeatureParamsOwnerImpl(configID string) (ownerID string, found bool, err error) {
	configID = strings.TrimSpace(configID)
	if configID == "" || db == nil {
		return "", false, nil
	}
	var userID string
	qErr := db.QueryRow(
		`SELECT user_id FROM cloud_personal_feature_params_config WHERE id = ?`,
		configID,
	).Scan(&userID)
	if qErr == sql.ErrNoRows {
		return "", false, nil
	}
	if qErr != nil {
		// Table may not exist yet during transitional boot; treat as not found.
		if strings.Contains(qErr.Error(), "no such table") {
			return "", false, nil
		}
		return "", false, qErr
	}
	return strings.TrimSpace(userID), true, nil
}
