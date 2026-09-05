package main

import (
	"context"
	"log/slog"
)

func loadUserIsTester(userID string) (bool, error) {
	var v int
	err := db.QueryRow(`SELECT COALESCE(is_tester, 0) FROM auth_user WHERE id = ?`, userID).Scan(&v)
	if err != nil {
		return false, err
	}
	return v != 0, nil
}

func testerForcesTenant(body map[string]interface{}) bool {
	v, ok := body["is_tester"].(bool)
	return ok && v
}

func applyCreatedUserRoleFlags(ctx context.Context, userID string, body map[string]interface{}) {
	if isStaff, ok := body["is_staff"].(bool); ok && isStaff {
		if _, err := db.Exec(`UPDATE auth_user SET is_staff = 1 WHERE id = ?`, userID); err != nil {
			slog.ErrorContext(ctx, "admin_create_user_staff_failed",
				"level", "error", "user_id", userID, "error", err.Error())
		} else {
			ensurePlatformRoleRows(userID, "role-employee")
		}
	}
	if isTenant, ok := body["is_tenant"].(bool); ok && isTenant {
		if _, err := db.Exec(`UPDATE auth_user SET is_tenant = 1 WHERE id = ?`, userID); err != nil {
			slog.ErrorContext(ctx, "admin_create_user_tenant_failed",
				"level", "error", "user_id", userID, "error", err.Error())
		}
	}
	if isTester, ok := body["is_tester"].(bool); ok {
		if err := setUserTesterFlag(ctx, userID, isTester); err != nil {
			slog.ErrorContext(ctx, "admin_create_user_tester_failed",
				"level", "error", "user_id", userID, "error", err.Error())
			return
		}
		publishUserTesterFlagChanged(ctx, userID, isTester)
	}
}

func setUserTesterFlag(ctx context.Context, userID string, isTester bool) error {
	if isTester {
		_, err := db.Exec(`UPDATE auth_user SET is_tester = 1, is_tenant = 1 WHERE id = ?`, userID)
		if err != nil {
			return err
		}
		slog.InfoContext(ctx, "user_tester_flag_set",
			"level", "info", "user_id", userID, "is_tester", true)
		return nil
	}
	_, err := db.Exec(`UPDATE auth_user SET is_tester = 0 WHERE id = ?`, userID)
	if err != nil {
		return err
	}
	slog.InfoContext(ctx, "user_tester_flag_set",
		"level", "info", "user_id", userID, "is_tester", false)
	return nil
}

func maybePublishTesterFlagChanged(ctx context.Context, userID string, prevTester bool, body map[string]interface{}) {
	isTester, ok := body["is_tester"].(bool)
	if !ok || isTester == prevTester {
		return
	}
	publishUserTesterFlagChanged(ctx, userID, isTester)
}
