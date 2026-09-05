package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"confload"
)

const (
	bootstrapAdminUserID           = "bootstrap-admin"
	bootstrapAdminEmailPlaceholder = "__BOOTSTRAP_ADMIN_EMAIL__"
)

func sqlSafeBootstrapAdminEmail(email string) (string, error) {
	email = strings.TrimSpace(email)
	if email == "" || !strings.Contains(email, "@") {
		return "", fmt.Errorf("bootstrapAdmin.email is empty or invalid")
	}
	if strings.ContainsAny(email, "'\\\"\n\r\x00;") {
		return "", fmt.Errorf("bootstrapAdmin.email contains SQL-unsafe characters")
	}
	return email, nil
}

func resolveBootstrapAdminEmail(repoRoot string) (string, error) {
	var block struct {
		BootstrapAdmin *struct {
			Email string `yaml:"email"`
		} `yaml:"bootstrapAdmin"`
	}
	if err := confload.ReadAppConfig(repoRoot, "auth/task-auth", &block); err != nil {
		return "", fmt.Errorf("read bootstrapAdmin.email: %w", err)
	}
	if block.BootstrapAdmin == nil {
		return "", fmt.Errorf("bootstrapAdmin.email is required in conf/auth/task-auth/config.yaml")
	}
	email := strings.TrimSpace(block.BootstrapAdmin.Email)
	return sqlSafeBootstrapAdminEmail(email)
}

func renderBootstrapAdminSQL(raw, repoRoot string) (string, error) {
	if !strings.Contains(raw, bootstrapAdminEmailPlaceholder) {
		return raw, nil
	}
	email, err := resolveBootstrapAdminEmail(repoRoot)
	if err != nil {
		return "", err
	}
	log.Printf("[taskAuth] dataMigrate: substituted %s", bootstrapAdminEmailPlaceholder)
	return strings.ReplaceAll(raw, bootstrapAdminEmailPlaceholder, email), nil
}

func applyBootstrapAdminEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" || !strings.Contains(email, "@") {
		return fmt.Errorf("bootstrapAdmin.email is empty or invalid")
	}
	existing, err := findEmailLoginMethodByUserID(bootstrapAdminUserID)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf(
			"bootstrap admin %q 没有 email 登录方式 — dataMigrate 可能未执行。请运行: apply_datamigrate.sh task_auth dataMigrate/taskAuth",
			bootstrapAdminUserID,
		)
	}
	if strings.EqualFold(existing.Identifier, email) {
		return nil
	}
	other, err := findLoginMethodByEmail(email)
	if err != nil {
		return err
	}
	if other != nil && other.ObjectID != bootstrapAdminUserID {
		return fmt.Errorf("bootstrapAdmin.email %q is already bound to user %s", email, other.ObjectID)
	}
	_, err = db.Exec(
		`UPDATE auth_login_method SET identifier = ?, updated_at = NOW() WHERE id = ?`,
		email, existing.ID,
	)
	if err != nil {
		return err
	}
	log.Printf("[taskAuth] bootstrap-admin: wrote email %s (user_id=%s)", email, bootstrapAdminUserID)
	return nil
}

// ensureBootstrapAdminSeeded verifies dataMigrate seeded the bootstrap superadmin,
// then writes conf bootstrapAdmin.email onto that user's email login method.
func ensureBootstrapAdminSeeded(dsn string, repoRoot string) error {
	if err := runDataMigrateFromDir(dsn, repoRoot); err != nil {
		return fmt.Errorf("migrations: %w", err)
	}
	if err := openDB(dsn); err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()
	if err := loadUserContentTypeID(); err != nil {
		return fmt.Errorf("content_type: %w", err)
	}

	email, err := resolveBootstrapAdminEmail(repoRoot)
	if err != nil {
		return err
	}
	if err := applyBootstrapAdminEmail(email); err != nil {
		return err
	}

	existing, err := findEmailLoginMethodByUserID(bootstrapAdminUserID)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf(
			"bootstrap admin %q 不存在 — dataMigrate 可能未执行。"+
				"请运行: apply_datamigrate.sh task_auth dataMigrate/taskAuth",
			bootstrapAdminUserID,
		)
	}

	if err := ensureBootstrapAdminUserActive(existing.ObjectID); err != nil {
		return err
	}
	if err := ensureSuperAdminRow(existing.ObjectID); err != nil {
		return err
	}

	log.Printf("[taskAuth] bootstrap-admin: %s verified (user_id=%s)", email, existing.ObjectID)
	return nil
}

func ensureBootstrapAdminUserActive(userID string) error {
	_, err := db.Exec(`
		UPDATE auth_user
		SET is_active = 1, is_superuser = 1, is_staff = 1
		WHERE id = ?`, userID)
	return err
}

func lookupBootstrapAdminUserID() (string, error) {
	lm, err := findEmailLoginMethodByUserID(bootstrapAdminUserID)
	if err != nil {
		return "", err
	}
	if lm == nil {
		return "", sql.ErrNoRows
	}
	return lm.ObjectID, nil
}
