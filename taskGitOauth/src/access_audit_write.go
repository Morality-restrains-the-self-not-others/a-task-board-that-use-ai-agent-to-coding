package main

import (
	"taskGitOauth/infrastructure"
)

func (a *App) accessAuditSite(providerKey string) string {
	if a == nil || a.Cfg == nil {
		return ""
	}
	return infrastructure.AuditSiteForProviderKey(a.Cfg, providerKey)
}

func (a *App) insertAccessAuditForProvider(providerKey, userID, action, fp string, detail map[string]any) {
	site := a.accessAuditSite(providerKey)
	if site == "" {
		logWarn("access audit skip: site unresolved provider_key=%s", providerKey)
		return
	}
	if err := a.DB.InsertAccessAudit(site, userID, nil, nil, action, fp, detail); err != nil {
		logWarn("access audit insert: %v", err)
	}
}
