package infrastructure

import "testing"

func TestTenantGitLabHostMatchesPathAIP(t *testing.T) {
	base := "http://115.29.110.74"
	if !tenantGitLabHostMatches(base, "115.29.110.74") {
		t.Fatal("bare IP must match Path A base_url")
	}
	if !tenantGitLabHostMatches(base, "http://115.29.110.74") {
		t.Fatal("URL host must match Path A base_url")
	}
	if tenantGitLabHostMatches(base, "gitlab.daydaymoney.com") {
		t.Fatal("platform GitLab host must not match Path A IP")
	}
}

func TestResolveProviderByGitsiteWithDBNilDBMissesPathA(t *testing.T) {
	cfg := &Config{Providers: map[string][]ProviderConfig{}}
	pc, err := cfg.ResolveProviderByGitsiteWithDB(nil, nil, "115.29.110.74")
	if err != nil {
		t.Fatal(err)
	}
	if pc != nil {
		t.Fatalf("YAML-empty + nil DB must not invent a provider, got %+v", pc)
	}
}
