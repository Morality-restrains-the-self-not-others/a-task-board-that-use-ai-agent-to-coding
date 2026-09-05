package main

import (
	"database/sql"
	"time"
)

const liveLoginMethodJoin = `
FROM auth_login_method lm
INNER JOIN auth_user u ON u.id = lm.object_id AND COALESCE(u.is_archived, 0) = 0`

const loginMethodSelectCols = `
SELECT lm.id, lm.object_id, lm.method_type, lm.identifier, COALESCE(lm.password_hash,''), lm.is_verified
`

func scanLoginMethodRow(row *sql.Row) (*LoginMethodRow, error) {
	var lm LoginMethodRow
	err := row.Scan(&lm.ID, &lm.ObjectID, &lm.MethodType, &lm.Identifier, &lm.PasswordHash, &lm.IsVerified)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &lm, nil
}

func findLoginMethodByIdentifier(identifier string) (*LoginMethodRow, error) {
	row := db.QueryRow(loginMethodSelectCols+liveLoginMethodJoin+`
		WHERE LOWER(lm.identifier) = LOWER(?) AND lm.binding_voided_at IS NULL
		LIMIT 1`, identifier)
	return scanLoginMethodRow(row)
}

func findLoginMethodByEmail(email string) (*LoginMethodRow, error) {
	row := db.QueryRow(loginMethodSelectCols+liveLoginMethodJoin+`
		WHERE LOWER(lm.identifier) = LOWER(?) AND lm.method_type = 'email' AND lm.binding_voided_at IS NULL
		LIMIT 1`, email)
	return scanLoginMethodRow(row)
}

func findEmailLoginMethodByUserID(userID string) (*LoginMethodRow, error) {
	row := db.QueryRow(loginMethodSelectCols+`
		FROM auth_login_method lm
		WHERE lm.object_id = ? AND lm.method_type = 'email' AND lm.binding_voided_at IS NULL
		LIMIT 1`, userID)
	return scanLoginMethodRow(row)
}

func findLoginMethodForPasswordReset(phone, email string) (*LoginMethodRow, error) {
	if phone != "" {
		return findLoginMethodByCanonicalPhone(canonicalPhoneForSMSAndLogin(phone))
	}
	if email != "" {
		return findLoginMethodByEmail(email)
	}
	return nil, nil
}

// findLoginMethodByCanonicalPhone 按规范化手机号（E.164 或国内号）查找 phone login_method。
// 供密码登录（handleLogin）与密码重置（findLoginMethodForPasswordReset）共用，避免
// 登录/重置各自复制一套规范化+国家码拆分逻辑（OPT-20260819-034）。
func findLoginMethodByCanonicalPhone(canonical string) (*LoginMethodRow, error) {
	if canonical == "" {
		return nil, nil
	}
	cc, nat := splitCountryCallingCodeAndNational(canonical)
	if cc == "" || nat == "" {
		return nil, nil
	}
	return findLoginMethodByPhone(cc, nat)
}

func findLoginMethodByPhone(countryCode, national string) (*LoginMethodRow, error) {
	methods, err := listLiveLoginMethodsByPhone(countryCode, national)
	if err != nil {
		return nil, err
	}
	if len(methods) == 0 {
		return nil, nil
	}
	lm := methods[0]
	return &lm, nil
}

func listLiveLoginMethodsByPhone(countryCode, national string) ([]LoginMethodRow, error) {
	rows, err := db.Query(loginMethodSelectCols+liveLoginMethodJoin+`
		WHERE lm.phone_country_calling_code = ? AND lm.identifier = ? AND lm.method_type = 'phone'
		AND lm.binding_voided_at IS NULL
		ORDER BY lm.id ASC
		LIMIT 32`, countryCode, national)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LoginMethodRow
	for rows.Next() {
		var lm LoginMethodRow
		if err := rows.Scan(&lm.ID, &lm.ObjectID, &lm.MethodType, &lm.Identifier, &lm.PasswordHash, &lm.IsVerified); err != nil {
			return nil, err
		}
		out = append(out, lm)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func voidStalePhoneLoginMethods(countryCode, national string) (int64, error) {
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	res, err := db.Exec(`
		UPDATE auth_login_method lm
		LEFT JOIN auth_user u ON u.id = lm.object_id
		SET lm.binding_voided_at = ?, lm.updated_at = ?
		WHERE lm.method_type = 'phone'
		  AND lm.phone_country_calling_code = ?
		  AND lm.identifier = ?
		  AND lm.binding_voided_at IS NULL
		  AND (u.id IS NULL OR COALESCE(u.is_archived, 0) = 1)`,
		now, now, countryCode, national)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func voidStaleEmailLoginMethods(email string) (int64, error) {
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	res, err := db.Exec(`
		UPDATE auth_login_method lm
		LEFT JOIN auth_user u ON u.id = lm.object_id
		SET lm.binding_voided_at = ?, lm.updated_at = ?
		WHERE lm.method_type = 'email'
		  AND LOWER(lm.identifier) = LOWER(?)
		  AND lm.binding_voided_at IS NULL
		  AND (u.id IS NULL OR COALESCE(u.is_archived, 0) = 1)`,
		now, now, email)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
