package main

import (
	"net/http"
	"strings"

	"taskGitOauth/infrastructure"
)

func catalogCompanyID(r *http.Request) string {
	if r == nil {
		return ""
	}
	if cid := strings.TrimSpace(r.URL.Query().Get("company_id")); cid != "" {
		return cid
	}
	for _, h := range []string{"X-Tenant-Id", "X-Auth-Tenant-Id"} {
		if cid := strings.TrimSpace(r.Header.Get(h)); cid != "" {
			return cid
		}
	}
	return ""
}

func providerCatalogEntry(pc infrastructure.ProviderConfig) map[string]any {
	label := pc.ProviderKey
	if pc.ServiceProvider != "" && pc.ServiceProvider != "default" {
		label = pc.Provider + ":" + pc.ServiceProvider
	}
	return map[string]any{
		"provider":         pc.Provider,
		"service_provider": pc.ServiceProvider,
		"provider_key":     pc.ProviderKey,
		"website":          pc.Website,
		"client_id":        pc.ClientID,
		"label":            label,
	}
}

func (a *App) mergeTenantProviderCatalog(providers []map[string]any, companyID string) []map[string]any {
	cid := strings.TrimSpace(companyID)
	if a == nil || a.DB == nil || cid == "" {
		return providers
	}
	row, err := a.DB.GetTenantGitLabConnection(cid)
	if err != nil || row == nil || !row.Active || strings.TrimSpace(row.BaseURL) == "" {
		return providers
	}
	pc, err := infrastructure.TenantRowToProviderConfig(row, nil)
	if err != nil || pc == nil || strings.TrimSpace(pc.Website) == "" {
		return providers
	}
	key := strings.ToLower(strings.TrimSpace(pc.ProviderKey))
	out := make([]map[string]any, 0, len(providers)+1)
	replaced := false
	for _, existing := range providers {
		existingKey, _ := existing["provider_key"].(string)
		if strings.ToLower(strings.TrimSpace(existingKey)) == key {
			out = append(out, providerCatalogEntry(*pc))
			replaced = true
			continue
		}
		out = append(out, existing)
	}
	if !replaced {
		out = append(out, providerCatalogEntry(*pc))
	}
	return out
}
