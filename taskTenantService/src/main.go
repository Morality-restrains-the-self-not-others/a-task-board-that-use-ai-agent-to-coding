package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"daydaymoneymeta"
	"tracelog"
)

func cleanPath(r *http.Request, prefix string) []string {
	p := strings.TrimPrefix(r.URL.Path, prefix)
	p = strings.TrimSuffix(p, "/")
	if p == "" {
		return nil
	}
	return strings.SplitN(p, "/", 12)
}

func main() {
	repoRoot, err := findMonorepoRoot()
	if err != nil {
		log.Fatalf("[taskTenantService] monorepo root: %v", err)
	}
	loadConfig(repoRoot)
	initLogging("task-tenant-service")
	tracelog.Init("task-tenant-service")
	if meta, err := daydaymoneymeta.LoadNearDir(filepath.Join(repoRoot, "taskTenantService")); err == nil {
		tracelog.SetDaydaymoneyMeta(meta.ServiceID, meta.Tags)
	}

	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if err := openDB(cfg.DBPath); err != nil {
			log.Fatalf("[taskTenantService] migration failed: %v", err)
		}
		if err := runDataMigrate(repoRoot); err != nil {
			log.Fatalf("[taskTenantService] dataMigrate: %v", err)
		}
		log.Println("[taskTenantService] migration complete")
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "backfill-member-name" {
		runMemberNameBackfill(os.Args[2:])
		return
	}

	if err := openDB(cfg.DBPath); err != nil {
		log.Fatalf("[taskTenantService] db: %v", err)
	}
	defer db.Close()
	initMemberAvatarMediaRoot(repoRoot)

	mux := http.NewServeMux()
	mountRoutes(mux)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	logInfo("listening on "+addr, "")
	log.Printf("[taskTenantService] listening on %s (db=%s)", addr, cfg.DBPath)
	if err := tracelog.ListenAndServe(addr, tracelog.Middleware(corsMiddleware(gatewayUserMiddleware(mux)))); err != nil {
		log.Fatal(err)
	}
}

func mountRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/health/", handleHealth)
	mux.HandleFunc("/api/live", handleHealth)
	mux.HandleFunc("/api/live/", handleHealth)

	mux.HandleFunc("/api/tenant/", func(w http.ResponseWriter, r *http.Request) {
		parts := cleanPath(r, "/api/tenant/")
		switch parts[0] {
		case "member-role":
			handleMemberRoleRoute(w, r, parts)
			return
		case "group-role":
			handleGroupRoleRoute(w, r, parts)
			return
		case "group-admin":
			handleGroupAdminRoute(w, r, parts)
			return
		case "resource-group":
			handleResourceGroupRoute(w, r, parts)
			return
		}
		if len(parts) < 3 || parts[1] != "accounts" {
			writeError(w, r, 404, "not found")
			return
		}
		tenantID := parts[0]
		r.Header.Set("X-Auth-Tenant-Id", tenantID)
		resource := parts[2]
		rest := parts[3:]
		switch resource {
		case "members":
			handleMembersRoute(w, r, tenantID, rest)
		case "groups":
			handleGroupsRoute(w, r, tenantID, rest)
		case "companies":
			handleCompaniesRoute(w, r, tenantID, rest)
		default:
			writeError(w, r, 404, "not found")
		}
	})

	mux.HandleFunc("/api/internal/tenant/authz-state", handleInternalAuthzState)
	mux.HandleFunc("/api/internal/tenant/authz-state/", handleInternalAuthzState)
	mux.HandleFunc("/api/internal/tenant/members", handleInternalMembers)
	mux.HandleFunc("/api/internal/tenant/members/", handleInternalMembers)
	mux.HandleFunc("/api/internal/tenant/users/", handleInternalTenantUsersRouter)
	mux.HandleFunc("/api/internal/tenant/users", handleInternalTenantUsersRouter)
	mux.HandleFunc("/api/internal/tenant/account-deletion/cleanup-invites", handleInternalCleanupStaleInvites)
	mux.HandleFunc("/api/internal/tenant/account-deletion/cleanup-invites/", handleInternalCleanupStaleInvites)
	mux.HandleFunc("/api/internal/tenant/invitations/import", handleImportInvitations)
	mux.HandleFunc("/api/internal/tenant/invitations/import/", handleImportInvitations)
	mux.HandleFunc("/api/internal/tenant/invitations/delivery-callback/{id}", handleInvitationDeliveryCallback)
	mux.HandleFunc("/api/internal/tenant/invitations/delivery-callback/{id}/", handleInvitationDeliveryCallback)
	mux.HandleFunc("/api/internal/tenant/groups/import", handleImportGroups)
	mux.HandleFunc("/api/internal/tenant/groups/import/", handleImportGroups)
	mux.HandleFunc("/api/internal/tenant/groups", handleInternalGroups)
	mux.HandleFunc("/api/internal/tenant/groups/", handleInternalGroups)
	mux.HandleFunc("/api/internal/tenant/companies", handleInternalCompanies)
	mux.HandleFunc("/api/internal/tenant/companies/", handleInternalCompanies)

	mux.HandleFunc("/api/system_admin/accounts/admin/tenant-options/", handleAdminTenantOptions)
	mux.HandleFunc("/api/system_admin/accounts/admin/tenant-options", handleAdminTenantOptions)
	mux.HandleFunc("/api/system-admin/accounts/admin/tenant-options/", handleAdminTenantOptions)
	mux.HandleFunc("/api/system-admin/accounts/admin/tenant-options", handleAdminTenantOptions)

	mux.HandleFunc("/api/system_admin/accounts/admin/tenants/{id}/", handleAdminTenantGet)
	mux.HandleFunc("/api/system_admin/accounts/admin/tenants/{id}", handleAdminTenantGet)
	mux.HandleFunc("/api/system-admin/accounts/admin/tenants/{id}/", handleAdminTenantGet)
	mux.HandleFunc("/api/system-admin/accounts/admin/tenants/{id}", handleAdminTenantGet)
	mux.HandleFunc("/api/system_admin/accounts/admin/tenants/", handleAdminTenants)
	mux.HandleFunc("/api/system_admin/accounts/admin/tenants", handleAdminTenants)
	mux.HandleFunc("/api/system-admin/accounts/admin/tenants/", handleAdminTenants)
	mux.HandleFunc("/api/system-admin/accounts/admin/tenants", handleAdminTenants)
}
