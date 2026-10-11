package main

import (
	"strings"
	"testing"

	dbload "dbload"
)

func TestRenderBootstrapAdminSQL(t *testing.T) {
	raw := "VALUES ('__BOOTSTRAP_ADMIN_EMAIL__')"
	out, err := renderBootstrapAdminSQL(raw, repoRoot())
	if err != nil {
		t.Fatalf("renderBootstrapAdminSQL: %v", err)
	}
	email := mustConfAdminEmail(t)
	if !strings.Contains(out, email) {
		t.Fatalf("rendered SQL missing conf email %q: %s", email, out)
	}
	if strings.Contains(out, bootstrapAdminEmailPlaceholder) {
		t.Fatal("placeholder still present after render")
	}
}

func TestSQLSafeBootstrapAdminEmailRejectsQuote(t *testing.T) {
	if _, err := sqlSafeBootstrapAdminEmail("bad'admin@example.com"); err == nil {
		t.Fatal("expected error for quote in email")
	}
}

func TestMigrateSeedsConfBootstrapAdminEmail(t *testing.T) {
	testDSN, cleanup, err := dbload.OpenTestMySQL("task-auth", repoRoot())
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}
	t.Cleanup(cleanup)
	if err := runDataMigrateFromDir(testDSN, repoRoot()); err != nil {
		t.Fatalf("runDataMigrateFromDir: %v", err)
	}
	if err := openDB(testDSN); err != nil {
		t.Fatalf("openDB: %v", err)
	}
	defer db.Close()
	lm, err := findEmailLoginMethodByUserID(bootstrapAdminUserID)
	if err != nil {
		t.Fatalf("findEmailLoginMethodByUserID: %v", err)
	}
	if lm == nil {
		t.Fatal("expected bootstrap-admin email login method after migrate")
	}
	want := mustConfAdminEmail(t)
	if !strings.EqualFold(lm.Identifier, want) {
		t.Fatalf("migrate identifier: got %q want conf %q", lm.Identifier, want)
	}
}

func TestResolveBootstrapAdminEmailFromConf(t *testing.T) {
	email, err := resolveBootstrapAdminEmail(repoRoot())
	if err != nil {
		t.Fatalf("resolveBootstrapAdminEmail: %v", err)
	}
	if !strings.Contains(email, "@") {
		t.Fatalf("expected conf bootstrapAdmin.email to contain @, got %q", email)
	}
}

func mustConfAdminEmail(t *testing.T) string {
	t.Helper()
	email, err := resolveBootstrapAdminEmail(repoRoot())
	if err != nil {
		t.Fatalf("resolveBootstrapAdminEmail: %v", err)
	}
	return email
}

func TestApplyBootstrapAdminEmailRejectsEmpty(t *testing.T) {
	if err := applyBootstrapAdminEmail(""); err == nil {
		t.Fatal("expected error for empty email")
	}
	if err := applyBootstrapAdminEmail("not-an-email"); err == nil {
		t.Fatal("expected error for identifier without @")
	}
}

func TestBootstrapAdminWritesConfiguredEmail(t *testing.T) {
	testDSN, cleanup, err := dbload.OpenTestMySQL("task-auth", repoRoot())
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}
	t.Cleanup(cleanup)
	if err := ensureBootstrapAdminSeeded(testDSN, repoRoot()); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if err := openDB(testDSN); err != nil {
		t.Fatalf("openDB: %v", err)
	}
	defer db.Close()

	want := "ops-init-admin@example.com"
	if err := applyBootstrapAdminEmail(want); err != nil {
		t.Fatalf("applyBootstrapAdminEmail: %v", err)
	}
	lm, err := findEmailLoginMethodByUserID(bootstrapAdminUserID)
	if err != nil {
		t.Fatalf("findEmailLoginMethodByUserID: %v", err)
	}
	if lm == nil {
		t.Fatal("expected bootstrap-admin email login method")
	}
	if !strings.EqualFold(lm.Identifier, want) {
		t.Fatalf("identifier: got %q want %q", lm.Identifier, want)
	}
}

func TestApplyBootstrapAdminEmailConflict(t *testing.T) {
	testDSN, cleanup, err := dbload.OpenTestMySQL("task-auth", repoRoot())
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}
	t.Cleanup(cleanup)
	if err := ensureBootstrapAdminSeeded(testDSN, repoRoot()); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if err := openDB(testDSN); err != nil {
		t.Fatalf("openDB: %v", err)
	}
	defer db.Close()
	if err := loadUserContentTypeID(); err != nil {
		t.Fatalf("content_type: %v", err)
	}

	taken := "taken-admin@example.com"
	if _, _, err := createUserWithEmailLogin(taken, "hash"); err != nil {
		t.Fatalf("create occupant: %v", err)
	}
	applyErr := applyBootstrapAdminEmail(taken)
	if applyErr == nil {
		t.Fatal("expected conflict error when email is already bound")
	}
	if !strings.Contains(applyErr.Error(), "already bound") {
		t.Fatalf("expected already bound error, got %v", applyErr)
	}
}

func TestBootstrapAdminIdempotent(t *testing.T) {
	testDSN, cleanup, err := dbload.OpenTestMySQL("task-auth", repoRoot())
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}
	t.Cleanup(cleanup)

	if err := ensureBootstrapAdminSeeded(testDSN, repoRoot()); err != nil {
		t.Fatalf("first bootstrap: %v", err)
	}
	if err := openDB(testDSN); err != nil {
		t.Fatalf("openDB after first bootstrap: %v", err)
	}
	userID1, err := lookupBootstrapAdminUserID()
	if err != nil {
		t.Fatalf("lookup user id: %v", err)
	}
	if userID1 == "" {
		t.Fatal("expected user id after bootstrap")
	}

	if err := ensureBootstrapAdminSeeded(testDSN, repoRoot()); err != nil {
		t.Fatalf("second bootstrap: %v", err)
	}
	if err := openDB(testDSN); err != nil {
		t.Fatalf("openDB after second bootstrap: %v", err)
	}
	userID2, err := lookupBootstrapAdminUserID()
	if err != nil {
		t.Fatalf("lookup user id again: %v", err)
	}
	if userID1 != userID2 {
		t.Fatalf("user id changed: %s -> %s", userID1, userID2)
	}

	if err := openDB(testDSN); err != nil {
		t.Fatalf("openDB: %v", err)
	}
	defer db.Close()

	var count int
	email := mustConfAdminEmail(t)
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM auth_login_method WHERE LOWER(identifier)=LOWER(?)`,
		email,
	).Scan(&count); err != nil {
		t.Fatalf("count login methods: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 login_method, got %d", count)
	}

	var isSuper, isActive bool
	if err := db.QueryRow(
		`SELECT is_superuser, is_active FROM auth_user WHERE id=?`, userID1,
	).Scan(&isSuper, &isActive); err != nil {
		t.Fatalf("load user: %v", err)
	}
	if !isSuper || !isActive {
		t.Fatalf("expected superuser active user, got super=%v active=%v", isSuper, isActive)
	}

	var superCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM auth_super_admin WHERE user_id = ?`, userID1).Scan(&superCount); err != nil {
		t.Fatalf("count super_admin: %v", err)
	}
	if superCount != 1 {
		t.Fatalf("expected 1 super_admin row, got %d", superCount)
	}
}

func TestBootstrapAdminCreatesFreshDB(t *testing.T) {
	testDSN, cleanup, err := dbload.OpenTestMySQL("task-auth", repoRoot())
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}
	t.Cleanup(cleanup)
	if err := ensureBootstrapAdminSeeded(testDSN, repoRoot()); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if err := openDB(testDSN); err != nil {
		t.Fatalf("openDB: %v", err)
	}
	defer db.Close()

	// On the fresh database, bootstrap must have created the schema and the
	// super admin user.
	userID, err := lookupBootstrapAdminUserID()
	if err != nil {
		t.Fatalf("lookup user id: %v", err)
	}
	if userID == "" {
		t.Fatal("expected bootstrap user id on fresh DB")
	}
	var count int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM auth_super_admin WHERE user_id = ?`, userID,
	).Scan(&count); err != nil {
		t.Fatalf("count super_admin: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 super_admin row, got %d", count)
	}
}

func TestNeedsBootstrapAdminPasswordRotation(t *testing.T) {
	if !needsBootstrapAdminPasswordRotation("") {
		t.Fatal("empty hash should rotate")
	}
	if !needsBootstrapAdminPasswordRotation(bootstrapAdminPasswordPending) {
		t.Fatal("pending sentinel should rotate")
	}
	if !needsBootstrapAdminPasswordRotation(legacySharedBootstrapPasswordHash) {
		t.Fatal("legacy shared hash should rotate")
	}
	hash, err := hashPassword("unique-ops-password-NotShared1!")
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	if needsBootstrapAdminPasswordRotation(hash) {
		t.Fatal("already-rotated unique hash must not rotate again")
	}
}

func TestGenerateRandomAdminPasswordUnique(t *testing.T) {
	a, err := generateRandomAdminPassword()
	if err != nil {
		t.Fatalf("a: %v", err)
	}
	b, err := generateRandomAdminPassword()
	if err != nil {
		t.Fatalf("b: %v", err)
	}
	if a == "" || b == "" || a == b {
		t.Fatalf("expected distinct non-empty passwords, got %q / %q", a, b)
	}
	if len(a) < 32 {
		t.Fatalf("password too short: %d", len(a))
	}
}

func TestBootstrapAdminGeneratesRandomPassword(t *testing.T) {
	testDSN, cleanup, err := dbload.OpenTestMySQL("task-auth", repoRoot())
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}
	t.Cleanup(cleanup)
	if err := ensureBootstrapAdminSeeded(testDSN, repoRoot()); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if err := openDB(testDSN); err != nil {
		t.Fatalf("openDB: %v", err)
	}
	defer db.Close()

	lm, err := findEmailLoginMethodByUserID(bootstrapAdminUserID)
	if err != nil || lm == nil {
		t.Fatalf("find login method: %v lm=%v", err, lm)
	}
	if lm.PasswordHash == "" || lm.PasswordHash == bootstrapAdminPasswordPending {
		t.Fatalf("password still pending/empty: %q", lm.PasswordHash)
	}
	if lm.PasswordHash == legacySharedBootstrapPasswordHash {
		t.Fatal("password still legacy shared hash")
	}
	for _, weak := range knownWeakBootstrapPasswords {
		if checkPasswordHash(weak, lm.PasswordHash) {
			t.Fatalf("password still weak %q", weak)
		}
	}
	must, err := getUserMustChangePassword(bootstrapAdminUserID)
	if err != nil {
		t.Fatalf("must_change_password: %v", err)
	}
	if !must {
		t.Fatal("must_change_password should stay 1 after random password seed")
	}

	// 幂等：再次 bootstrap 不得改写已生成的随机哈希
	firstHash := lm.PasswordHash
	if err := ensureBootstrapAdminSeeded(testDSN, repoRoot()); err != nil {
		t.Fatalf("second bootstrap: %v", err)
	}
	if err := openDB(testDSN); err != nil {
		t.Fatalf("reopen: %v", err)
	}
	lm2, err := findEmailLoginMethodByUserID(bootstrapAdminUserID)
	if err != nil || lm2 == nil {
		t.Fatalf("re-find: %v", err)
	}
	if lm2.PasswordHash != firstHash {
		t.Fatalf("idempotent bootstrap changed hash:\n  first=%s\n  second=%s", firstHash, lm2.PasswordHash)
	}
}
