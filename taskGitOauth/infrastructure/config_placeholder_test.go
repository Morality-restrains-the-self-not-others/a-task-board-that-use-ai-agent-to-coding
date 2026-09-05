package infrastructure

import (
	"strings"
	"testing"
)

func TestDomainPlaceholderREMatchesCamelCaseSubdomain(t *testing.T) {
	raw := "${scheme}://${subdomains.gitlabTencentSh1}/oauth"
	found := map[string]bool{}
	for _, m := range domainPlaceholderRE.FindAllStringSubmatch(raw, -1) {
		if len(m) > 1 {
			found[m[1]] = true
		}
	}
	if !found["scheme"] {
		t.Fatal("expected scheme placeholder")
	}
	if !found["subdomains.gitlabTencentSh1"] {
		t.Fatalf("camelCase subdomain must resolve, got %v", found)
	}
}

func TestLoadConfigIncludesTencentSh1GitLab(t *testing.T) {
	root, err := FindMonorepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	pc := ProviderConfigByKey(cfg, "gitlab:tencent-sh-1")
	if pc == nil {
		t.Fatal("missing gitlab:tencent-sh-1 provider YAML")
	}
	if strings.Contains(pc.Website, "${") {
		t.Fatalf("website still has unresolved placeholder: %q", pc.Website)
	}
	host := hostnameOf(pc.Website)
	if !strings.Contains(host, "gitlab-tencent-sh-1") {
		t.Fatalf("website host=%q want *gitlab-tencent-sh-1* (website=%q)", host, pc.Website)
	}
	if strings.Contains(pc.RedirectURI, "${") {
		t.Fatalf("redirect_uri still has unresolved placeholder: %q", pc.RedirectURI)
	}
	if !strings.Contains(pc.RedirectURI, "/redirect/gitsite/"+host+"/oauth/callback/") {
		t.Fatalf("redirect_uri=%q host=%q", pc.RedirectURI, host)
	}
	if cfg.ResolveProviderByGitsite(host) == nil {
		t.Fatal("ResolveProviderByGitsite must find tencent-sh-1")
	}
}
