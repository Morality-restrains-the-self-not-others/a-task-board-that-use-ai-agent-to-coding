package main

import (
	"testing"
	"time"

	"taskGitOauth/domain"
	"taskGitOauth/infrastructure"
)

func TestProviderConfigForRepoURLUsesTenantPathAHost(t *testing.T) {
	app := testApp(t)
	tid := "877397588196749312"
	cipher, err := app.Fernet.Encrypt("sekrit")
	if err != nil {
		t.Fatal(err)
	}
	_, err = app.DB.UpsertTenantGitLabConnection(&infrastructure.TenantGitLabOAuthConnectionRow{
		CompanyID: tid, BaseURL: "http://115.29.110.74", ClientID: "path-a-client",
		ClientSecretEnc: cipher, RedirectURI: domain.DefaultRedirectURI(app.Cfg.PublicBaseURL, tid),
		Scope: domain.DefaultTenantGitLabScope, Active: true, UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	pc := app.providerConfigForRepoURL("http://115.29.110.74/example-user/somanyad.git")
	if pc == nil {
		t.Fatal("expected Path A provider from DB")
	}
	if pc.ProviderKey != domain.ProviderKeyForCompany(tid) {
		t.Fatalf("provider_key=%s", pc.ProviderKey)
	}
}
