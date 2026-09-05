package main

import (
	"taskGitOauth/infrastructure"
	"testing"
)

func TestProviderCatalogClientID(t *testing.T) {
	cfg, err := infrastructure.LoadConfig("../../../..")
	if err != nil {
		t.Skipf("cannot load config from test: %v", err)
	}
	all := cfg.GetAllProviderConfigs()
	if len(all) == 0 {
		t.Skip("no provider configs found")
	}
	for _, pc := range all {
		t.Logf("provider_key=%s website=%q client_id=%q label=%s",
			pc.ProviderKey, pc.Website, pc.ClientID, pc.ProviderKey)
		if pc.ClientID == "" {
			t.Errorf("ProviderConfig %s has empty ClientID", pc.ProviderKey)
		}
	}
}
