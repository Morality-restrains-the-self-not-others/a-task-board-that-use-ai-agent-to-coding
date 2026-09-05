package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"taskAuth/domain"
)

// handleSystemAdminCreateUser creates a new user from the system admin panel.
// POST /api/system-admin/users/create/
func handleSystemAdminCreateUser(w http.ResponseWriter, r *http.Request) {
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorDetail(w, r, 400, "invalid json")
		return
	}

	if rejectUnauthorizedSuperuserGrant(w, r, body) {
		return
	}

	email := strField(body, "email")
	password := strField(body, "password")
	phone := strings.TrimSpace(strField(body, "phone"))
	username := strField(body, "username")
	if email == "" || password == "" {
		writeErrorDetail(w, r, 400, "email and password required")
		return
	}

	if _, _, err := checkAdminLoginIdentifiers("", phone, ""); err != nil {
		writeAdminIdentifierError(w, r, "", err)
		return
	}

	userID, _, err := createUserWithEmailLogin(email, password)
	if err != nil {
		writeErrorDetail(w, r, 400, err.Error())
		return
	}
	if err := applyAdminLoginIdentifiers(r.Context(), userID, phone, ""); err != nil {
		_ = deleteUser(userID)
		writeAdminIdentifierError(w, r, userID, err)
		return
	}
	if username != "" {
		if err := upsertUserProfile(userID, username); err != nil {
			_ = deleteUser(userID)
			writeErrorDetail(w, r, 500, "db error")
			return
		}
	}

	// Optionally set superuser flag
	if isSuperuser, ok := body["is_superuser"].(bool); ok && isSuperuser {
		_, _ = db.Exec(`INSERT IGNORE INTO auth_super_admin (user_id) VALUES (?)`, userID)
		_, _ = db.Exec(`UPDATE auth_user SET is_superuser = 1 WHERE id = ?`, userID)
		// v63 RBAC 双写：确保 /api/auth/user-roles/ 平台角色一致
		ensurePlatformRoleRows(userID, "role-super-admin")
	}

	applyCreatedUserRoleFlags(r.Context(), userID, body)

	payload, err := buildUserDetailJSON(userID)
	if err != nil {
		writeErrorDetail(w, r, 500, "db error")
		return
	}
	writeJSON(w, 201, payload)
}

// patchUserAsAdmin updates user fields with superuser auth.
func patchUserAsAdmin(w http.ResponseWriter, r *http.Request, userID string) {
	body, _ := readJSONBody(r)
	if rejectUnauthorizedSuperuserGrant(w, r, body) {
		return
	}

	phone := strings.TrimSpace(strField(body, "phone"))
	email := strings.TrimSpace(strField(body, "email"))
	_, hasPhone := body["phone"]
	_, hasUsername := body["username"]
	newPassword := strField(body, "password")
	if err := applyAdminLoginIdentifiers(r.Context(), userID, phone, email); err != nil {
		writeAdminIdentifierError(w, r, userID, err)
		return
	}
	if hasPhone && phone == "" {
		if err := voidPhoneLoginMethodsForUser(userID); err != nil {
			slog.ErrorContext(r.Context(), "system_admin_user_phone_unbind_failed",
				"user_id", userID, "err", err.Error())
			writeErrorDetail(w, r, 500, "db error")
			return
		}
		slog.InfoContext(r.Context(), "system_admin_user_phone_unbound", "user_id", userID)
	}
	if hasUsername {
		if err := upsertUserProfile(userID, strField(body, "username")); err != nil {
			writeErrorDetail(w, r, 500, "db error")
			return
		}
	}
	if newPassword != "" {
		if err := applyAdminPassword(r.Context(), userID, newPassword); err != nil {
			writeErrorDetail(w, r, 500, "db error")
			return
		}
	}

	allowedFields := map[string]bool{
		"is_active":    true,
		"is_superuser": true,
		"is_staff":     true,
		"is_tenant":    true,
		"is_tester":    true,
		"is_archived":  true,
	}

	prevTester, _ := loadUserIsTester(userID)

	setClauses := []string{}
	args := []interface{}{}
	forceTenant := testerForcesTenant(body)
	testerInBody := false
	testerVal := false
	if v, ok := body["is_tester"].(bool); ok {
		testerInBody = true
		testerVal = v
	}
	for key := range allowedFields {
		if key == "is_tenant" && forceTenant {
			continue
		}
		if key == "is_tester" {
			continue
		}
		val, ok := body[key]
		if !ok {
			continue
		}
		boolVal, isBool := val.(bool)
		if !isBool {
			continue
		}
		setClauses = append(setClauses, key+" = ?")
		args = append(args, boolVal)
	}
	if forceTenant {
		setClauses = append(setClauses, "is_tenant = ?")
		args = append(args, true)
	}

	if len(setClauses) == 0 && !testerInBody && phone == "" && !hasPhone && email == "" && !hasUsername && newPassword == "" {
		writeErrorDetail(w, r, 400, "no valid fields to update")
		return
	}

	if len(setClauses) > 0 {
		args = append(args, userID)
		query := "UPDATE auth_user SET " + strings.Join(setClauses, ", ") + " WHERE id = ?"
		_, err := db.Exec(query, args...)
		if err != nil {
			writeErrorDetail(w, r, 500, "db error")
			return
		}
	}
	if testerInBody {
		if err := setUserTesterFlag(r.Context(), userID, testerVal); err != nil {
			writeErrorDetail(w, r, 500, "db error")
			return
		}
	}

	// v63 RBAC 双写：is_superuser/is_staff 标志变更时同步平台角色行，
	// 避免 /api/auth/user-roles/ 与 /me/ 平台角色不一致。
	if isSuper, ok := body["is_superuser"].(bool); ok && isSuper {
		ensurePlatformRoleRows(userID, "role-super-admin")
	}
	if isStaff, ok := body["is_staff"].(bool); ok && isStaff {
		hasSuper, _ := hasPlatformRole(userID, "super_admin")
		if !hasSuper {
			ensurePlatformRoleRows(userID, "role-employee")
		}
	}

	maybePublishTesterFlagChanged(r.Context(), userID, prevTester, body)

	payload, err := buildUserDetailJSON(userID)
	if err == sql.ErrNoRows {
		writeErrorDetail(w, r, 404, "user not found")
		return
	}
	if err != nil {
		writeErrorDetail(w, r, 500, "db error")
		return
	}
	writeJSON(w, 200, payload)
}

var errAdminIdentifierInvalid = errors.New("invalid identifier")

// checkAdminLoginIdentifiers validates phone/email occupancy without writing.
func checkAdminLoginIdentifiers(userID, phone, email string) (cc, national string, err error) {
	if phone != "" {
		canonical := canonicalPhoneForSMSAndLogin(phone)
		cc, national = splitCountryCallingCodeAndNational(canonical)
		if cc == "" || national == "" {
			return "", "", fmt.Errorf("%w: 手机号格式无效", errAdminIdentifierInvalid)
		}
		d, _, err := phoneBindDecision(userID, cc, national)
		if err != nil {
			return "", "", err
		}
		if d == domain.PhoneShareLimit {
			return "", "", errPhoneBindLimit
		}
		if d == domain.PhoneShareTaken {
			return "", "", errPhoneTaken
		}
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if email != "" {
		if !strings.Contains(email, "@") || strings.HasSuffix(email, "@sso.invalid") {
			return "", "", fmt.Errorf("%w: 邮箱格式无效", errAdminIdentifierInvalid)
		}
		lm, err := findLoginMethodByEmail(email)
		if err != nil {
			return "", "", err
		}
		if lm != nil && lm.ObjectID != userID {
			return "", "", errEmailTaken
		}
	}
	return cc, national, nil
}

// applyAdminLoginIdentifiers binds phone/email from the system-admin edit form.
// Occupied identifiers return errPhoneTaken / errEmailTaken / errPhoneBindLimit
// so the handler can 409 instead of silently keeping the previous binding.
// Empty phone is a no-op here; patchUserAsAdmin voids the live phone binding
// when the phone key is present and empty.
func applyAdminLoginIdentifiers(ctx context.Context, userID, phone, email string) error {
	cc, national, err := checkAdminLoginIdentifiers(userID, phone, email)
	if err != nil {
		return err
	}
	if phone != "" {
		if err := upsertPhoneLoginMethod(userID, cc, national); err != nil {
			return err
		}
		slog.InfoContext(ctx, "system_admin_user_phone_updated", "user_id", userID)
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if email != "" {
		if err := upsertEmailLoginMethod(userID, email); err != nil {
			return err
		}
		slog.InfoContext(ctx, "system_admin_user_email_updated", "user_id", userID)
	}
	return nil
}

func applyAdminPassword(ctx context.Context, userID, password string) error {
	lm, err := findEmailLoginMethodByUserID(userID)
	if err != nil {
		return err
	}
	if lm == nil {
		return fmt.Errorf("email login method not found")
	}
	if err := updatePasswordHashOnly(lm.ID, password); err != nil {
		return err
	}
	slog.InfoContext(ctx, "system_admin_user_password_updated", "user_id", userID)
	return nil
}

func writeAdminIdentifierError(w http.ResponseWriter, r *http.Request, userID string, err error) {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeErrorDetail(w, r, http.StatusNotFound, "user not found")
	case errors.Is(err, errPhoneTaken):
		slog.InfoContext(r.Context(), "system_admin_user_phone_taken", "user_id", userID)
		writeErrorMap(w, r, http.StatusConflict, map[string]interface{}{
			"error":  "该手机号已被其他用户使用",
			"detail": "该手机号已被其他用户使用",
			"code":   "phone_taken",
		})
	case errors.Is(err, errPhoneBindLimit):
		slog.InfoContext(r.Context(), "system_admin_user_phone_bind_limit", "user_id", userID)
		writeErrorMap(w, r, http.StatusConflict, map[string]interface{}{
			"error":  "该手机号已达绑定上限",
			"detail": "该手机号已达绑定上限，无法再绑定到此用户",
			"code":   "phone_bind_limit",
		})
	case errors.Is(err, errEmailTaken):
		slog.InfoContext(r.Context(), "system_admin_user_email_taken", "user_id", userID)
		writeErrorMap(w, r, http.StatusConflict, map[string]interface{}{
			"error":  "该邮箱已被其他用户使用",
			"detail": "该邮箱已被其他用户使用",
			"code":   "email_taken",
		})
	case errors.Is(err, errAdminIdentifierInvalid):
		msg := err.Error()
		if unwrapped := errors.Unwrap(err); unwrapped != nil {
			msg = strings.TrimPrefix(msg, unwrapped.Error()+": ")
		}
		if msg == "" || msg == errAdminIdentifierInvalid.Error() {
			msg = "标识格式无效"
		}
		writeErrorDetail(w, r, http.StatusBadRequest, msg)
	default:
		slog.ErrorContext(r.Context(), "system_admin_user_identifier_update_failed",
			"user_id", userID, "err", err.Error())
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
	}
}
