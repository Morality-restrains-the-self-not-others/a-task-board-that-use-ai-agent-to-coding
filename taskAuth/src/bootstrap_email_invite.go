package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

// bootstrapEmailInviteRecipient is the default email for the seeded invite token.
// When run without --email, the invitation is created for the bootstrap admin email.
const bootstrapEmailInviteRecipient = "author@example.com"

// runBootstrapEmailInvite idempotently creates a pending email registration invite
// for the given email (or the default bootstrap admin email if empty), then prints
// the invite URL to stdout so the operator can open it in a browser.
func runBootstrapEmailInvite(dsn string, repoRoot string, email string) error {
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

	if email == "" {
		email = bootstrapEmailInviteRecipient
	}

	// Check if email is already registered — refuse to invite an existing user.
	existingLM, err := findLoginMethodByEmail(email)
	if err != nil {
		return fmt.Errorf("lookup email: %w", err)
	}
	if existingLM != nil {
		return fmt.Errorf("邮箱 %s 已被注册（user_id=%s），无需邀请", email, existingLM.ObjectID)
	}

	// Check if a pending invite already exists for this email.
	var existingToken string
	var existingExpiresAt string
	err = db.QueryRow(`
		SELECT token, expires_at FROM auth_email_registration_invite
		WHERE email = ? AND status = 'pending' AND expires_at > ?
		ORDER BY created_at DESC LIMIT 1`,
		email, timeNowUTC(),
	).Scan(&existingToken, &existingExpiresAt)
	if err == nil {
		// A valid pending invite already exists — reuse it.
		frontendBase := cfg.FrontendBase
		if frontendBase == "" {
			frontendBase = "http://localhost:4000"
		}
		inviteURL := fmt.Sprintf("%s/auth/register/?invite_token=%s", frontendBase, existingToken)
		fmt.Printf("已存在有效邀请:\n")
		fmt.Printf("  邮箱:       %s\n", email)
		fmt.Printf("  过期时间:   %s\n", existingExpiresAt)
		fmt.Printf("  邀请链接:   %s\n\n", inviteURL)
		log.Printf("[taskAuth] bootstrap-email-invite: reused pending invite for %s", email)
		return nil
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("check existing invite: %w", err)
	}

	// Look up a superadmin to act as inviter.
	inviterUserID := findAnySuperAdmin()

	// Generate token and create the invitation.
	token, err := generateEmailInviteToken()
	if err != nil {
		return fmt.Errorf("generate token: %w", err)
	}

	id := generateSnowflakeID()
	now := timeNowUTC()
	expires := time.Now().UTC().Add(7 * 24 * time.Hour).Format("2006-01-02 15:04:05.000000")

	_, err = db.Exec(`
		INSERT INTO auth_email_registration_invite
		(id, email, token, inviter_user_id, status, expires_at, created_at, email_sent_at, email_send_attempts, delivery_status)
		VALUES (?, ?, ?, ?, 'pending', ?, ?, ?, 1, 'pending')`,
		id, email, token, inviterUserID, expires, now, now,
	)
	if err != nil {
		return fmt.Errorf("insert invite: %w", err)
	}

	frontendBase := cfg.FrontendBase
	if frontendBase == "" {
		frontendBase = "http://localhost:4000"
	}
	inviteURL := fmt.Sprintf("%s/auth/register/?invite_token=%s", frontendBase, token)

	fmt.Printf("✅ 邀请已创建:\n")
	fmt.Printf("  邮箱:       %s\n", email)
	fmt.Printf("  令牌:       %s\n", token)
	fmt.Printf("  过期时间:   %s\n", expires)
	fmt.Printf("  邀请链接:   %s\n\n", inviteURL)

	log.Printf("[taskAuth] bootstrap-email-invite: created invite for %s token=%s", email, token[:8]+"...")
	return nil
}

// findAnySuperAdmin returns the user ID of any superadmin, or "0" if none exist.
// This is used as the inviter_user_id when creating seed invites.
func findAnySuperAdmin() string {
	var userID string
	err := db.QueryRow(`
		SELECT object_id FROM auth_login_method
		WHERE method_type = 'email'
		ORDER BY id ASC LIMIT 1`,
	).Scan(&userID)
	if err == nil && userID != "" {
		// Verify this is actually a superadmin
		isSuper, _ := isSuperAdminUser(userID)
		if isSuper {
			return userID
		}
	}
	// Fallback: look up any user with is_superuser flag
	err = db.QueryRow(`
		SELECT id FROM auth_user
		WHERE is_superuser = 1
		ORDER BY id ASC LIMIT 1`,
	).Scan(&userID)
	if err == nil && userID != "" {
		return userID
	}
	return "0" // bootstrap sentinel — invitation created by the system
}
