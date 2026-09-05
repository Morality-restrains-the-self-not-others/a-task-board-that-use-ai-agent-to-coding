package main

import (
	"testing"
)

// 回归：conf 追加 OIDC client 后，data_migrate_log 已有 010_oidc_bootstrap_clients
// 仍须 INSERT。GitLab SH-1 SSO 曾因一次性 skip 返回 unauthorized_client / client not found。
func TestRunGoDataMigrateReseedsOidcClientsWhenStepAlreadyLogged(t *testing.T) {
	oldCfg := cfg
	t.Cleanup(func() { cfg = oldCfg })
	setupAuthTestDB(t)

	adminURIs := `["http://admin-locked.example.com/cb"]`
	cfg = Config{
		OidcBootstrapClients: []OidcBootstrapClient{
			{ClientID: "existing-logged-client", ClientSecret: "s1", RedirectURI: "http://existing.example.com/cb"},
			{ClientID: "appended-after-010", ClientSecret: "s2", RedirectURI: "http://new.example.com/cb"},
			{ClientID: "admin-locked-client", ClientSecret: "conf-secret", RedirectURI: "http://conf-would-overwrite.example.com/cb"},
		},
	}

	if err := ensureOidcClient("existing-logged-client", "s1", "existing-logged-client", `["http://existing.example.com/cb"]`); err != nil {
		t.Fatalf("pre-seed existing: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO auth_oidc_client (id, client_id, client_secret_hash, name, redirect_uris, managed_by, created_at, updated_at)
		VALUES ('9201', 'admin-locked-client', 'secret-hash', 'admin-locked-client', ?, 'admin', NOW(), NOW())`, adminURIs); err != nil {
		t.Fatalf("insert admin-managed row: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO data_migrate_log (step_key, applied_at, checksum)
		VALUES ('010_oidc_bootstrap_clients', NOW(), 'stale')`); err != nil {
		t.Fatalf("pre-insert migrate log: %v", err)
	}

	row, err := loadOidcClient("appended-after-010")
	if err != nil {
		t.Fatalf("precondition load: %v", err)
	}
	if row != nil {
		t.Fatalf("precondition: appended client must be absent, got %+v", row)
	}

	if err := RunGoDataMigrate(); err != nil {
		t.Fatalf("RunGoDataMigrate: %v", err)
	}
	if err := RunGoDataMigrate(); err != nil {
		t.Fatalf("second RunGoDataMigrate: %v", err)
	}

	appended, err := loadOidcClient("appended-after-010")
	if err != nil || appended == nil {
		t.Fatalf("expected INSERT of appended client: err=%v row=%v", err, appended)
	}
	if appended.RedirectURIs != `["http://new.example.com/cb"]` {
		t.Fatalf("appended redirect_uris: %s", appended.RedirectURIs)
	}

	existing, err := loadOidcClient("existing-logged-client")
	if err != nil || existing == nil {
		t.Fatalf("existing client dropped: err=%v row=%v", err, existing)
	}

	admin, err := loadOidcClient("admin-locked-client")
	if err != nil || admin == nil {
		t.Fatalf("admin client missing: err=%v row=%v", err, admin)
	}
	if admin.RedirectURIs != adminURIs {
		t.Fatalf("admin redirect_uris overwritten: %s", admin.RedirectURIs)
	}
	if admin.ManagedBy != "admin" {
		t.Fatalf("managed_by want admin, got %s", admin.ManagedBy)
	}

	var checksum string
	if err := db.QueryRow(`SELECT checksum FROM data_migrate_log WHERE step_key = ?`, "010_oidc_bootstrap_clients").Scan(&checksum); err != nil {
		t.Fatalf("checksum query: %v", err)
	}
	if checksum == "" || checksum == "stale" {
		t.Fatalf("checksum not refreshed: %q", checksum)
	}
	if want := oidcBootstrapClientChecksum(); checksum != want {
		t.Fatalf("checksum=%s want=%s", checksum, want)
	}
}

func TestOidcBootstrapClientChecksumIgnoresSecret(t *testing.T) {
	old := cfg
	t.Cleanup(func() { cfg = old })
	cfg = Config{OidcBootstrapClients: []OidcBootstrapClient{
		{ClientID: "b", ClientSecret: "secret-a", RedirectURI: "http://a.example/cb"},
		{ClientID: "a", ClientSecret: "secret-b", RedirectURI: "http://b.example/cb"},
	}}
	first := oidcBootstrapClientChecksum()
	if first == "" {
		t.Fatal("checksum empty")
	}
	cfg.OidcBootstrapClients[0].ClientSecret = "changed"
	if oidcBootstrapClientChecksum() != first {
		t.Fatal("checksum must ignore client secrets")
	}
	cfg.OidcBootstrapClients[0].ClientID = "z"
	if oidcBootstrapClientChecksum() == first {
		t.Fatal("checksum must change when client_id set changes")
	}
}
