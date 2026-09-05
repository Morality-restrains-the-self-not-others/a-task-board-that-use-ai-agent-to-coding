package main

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"taskAiProvider/domain"
	"taskAiProvider/infrastructure"
)

type App struct {
	Cfg    *infrastructure.Config
	DB     *infrastructure.DB
	OIDC   *oidcStore
	JWKS   *infrastructure.JWKSCache
	Docs   domain.VendorDocStore
	Events domain.EventBus
	Path   *infrastructure.VendorDocsPathState
	// AutoRunExtractor pulls /app/autoRunStep.md from OCI. Tests inject a stub.
	AutoRunExtractor autoRunStepsExtractor
	// SkillsExtractor pulls /app/imageSkills.yaml from OCI. Tests inject a stub.
	SkillsExtractor imageSkillsExtractor
	// AutoRunAndSkillsExtractor walks layers once for both files (OPT-20260821-018).
	// Tests inject a stub; production falls back to the default single-pass walk.
	AutoRunAndSkillsExtractor autoRunAndSkillsExtractor
	// ExtractSync runs extract on the request goroutine (tests). Production is async.
	ExtractSync bool
}

func NewApp(cfg *infrastructure.Config, db *infrastructure.DB) *App {
	path := infrastructure.NewVendorDocsPathState(cfg)
	return &App{
		Cfg:    cfg,
		DB:     db,
		OIDC:   newOIDCStore(),
		JWKS:   infrastructure.NewJWKSCache(infrastructure.OIDCJWKSURL(cfg.OIDCRpIssuer)),
		Path:   path,
		Docs:   infrastructure.NewVendorDocStore(cfg, path, nil),
		Events: infrastructure.LogEventBus{},
	}
}

func (a *App) docStore() domain.VendorDocStore {
	if a != nil && a.Docs != nil {
		return a.Docs
	}
	root := ""
	if a != nil && a.Cfg != nil {
		root = a.Cfg.VendorDocsDir
	}
	return &infrastructure.LocalVendorDocStore{Root: root}
}

func (a *App) eventBus() domain.EventBus {
	if a != nil && a.Events != nil {
		return a.Events
	}
	return infrastructure.LogEventBus{}
}

func (a *App) vendorDocsBackend() string {
	if a != nil && a.Docs != nil {
		return a.Docs.Backend()
	}
	if a != nil && a.Cfg != nil && a.Cfg.VendorDocsBackend != "" {
		return a.Cfg.VendorDocsBackend
	}
	return domain.VendorDocBackendLocal
}

func (a *App) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/health/", a.handleHealth)
	mux.HandleFunc("/api/health", a.handleHealth)
	mux.HandleFunc("/api/schema/", a.handleSchema)
	mux.HandleFunc("/api/swagger/", a.handleSwagger)

	// Convention: /api/ai-provider/{funcName}/... — taskGateway forwards these
	// prefixes to this service without path rewrite (taskGateway/routes/routes.yaml).
	mux.HandleFunc("/api/ai-provider/sso-exchange/", a.handleSSOExchange)
	mux.HandleFunc("/api/ai-provider/oidc-authorize/", a.handleOIDCAuthorize)
	mux.HandleFunc("/api/ai-provider/oidc-callback/", a.handleOIDCCallback)

	mux.HandleFunc("/api/ai-provider/vendor-auth-register/", a.handleSSOOnly)
	mux.HandleFunc("/api/ai-provider/vendor-auth-login/", a.handleSSOOnly)
	mux.HandleFunc("/api/ai-provider/vendor-me/", a.handleVendorMe)
	mux.HandleFunc("/api/ai-provider/vendor-status/", a.handleVendorStatus)
	mux.HandleFunc("/api/ai-provider/vendor-application/upload-url/", a.handleVendorApplicationUploadURL)
	mux.HandleFunc("/api/ai-provider/vendor-application/upload-complete/", a.handleVendorApplicationUploadComplete)
	mux.HandleFunc("/api/ai-provider/vendor-application/local-put/", a.handleVendorApplicationLocalPut)
	mux.HandleFunc("/api/ai-provider/vendor-application/upload/", a.handleVendorApplicationUpload)
	mux.HandleFunc("/api/ai-provider/vendor-application/phone-status/", a.handleVendorApplicationPhoneStatus)
	mux.HandleFunc("/api/ai-provider/vendor-application/send-sms/", a.handleVendorApplicationSendSMS)
	mux.HandleFunc("/api/ai-provider/vendor-application/verify-phone/", a.handleVendorApplicationVerifyPhone)
	mux.HandleFunc("/api/ai-provider/vendor-application/", a.handleVendorApplication)
	mux.HandleFunc("/api/ai-provider/admin-vendor-docs-storage/", a.handleAdminVendorDocsStorage)
	mux.HandleFunc("/api/ai-provider/marketplace-settings/", a.handleMarketplaceSettings)
	mux.HandleFunc("/api/ai-provider/admin-marketplace-settings/", a.handleAdminMarketplaceSettings)
	mux.HandleFunc("/api/ai-provider/admin-auth-login/", a.handleSSOOnly)
	mux.HandleFunc("/api/ai-provider/admin-me/", a.handleStaffMe)

	mux.HandleFunc("/api/ai-provider/public-catalog/", a.handlePublicCatalog)
	mux.HandleFunc("/api/ai-provider/public-userdata-templates/", a.handlePublicUserDataTemplates)
	mux.HandleFunc("/api/ai-provider/public-image-runtime-envs/", a.handleImageRuntimeEnvs)
	mux.HandleFunc("/api/ai-provider/public-runtime-userdata/", a.handleRuntimeUserdata)
	mux.HandleFunc("/api/ai-provider/public-vendor-dev-catalog/", a.handleVendorDevCatalog)
	mux.HandleFunc("/api/ai-provider/public-unsubmitted-image/", a.handleUnsubmittedImage)

	mux.HandleFunc("/api/ai-provider/admin-vendors/", a.handleAdminVendors)
	mux.HandleFunc("/api/ai-provider/admin-userdata-verify/", a.handleUserdataVerify)

	mux.HandleFunc("/api/ai-provider/vendor-cloud-credentials/", a.handleCredentialProxy)
	mux.HandleFunc("/api/ai-provider/vendor-cloud-regions/", a.handleCloudQueryProxy)
	mux.HandleFunc("/api/ai-provider/vendor-cloud-images/", a.handleCloudQueryProxy)
	mux.HandleFunc("/api/ai-provider/vendor-cloud-instance-types/", a.handleCloudQueryProxy)

	mux.HandleFunc("/api/ai-provider/vendor-container-images/", a.handleVendorContainerImages)
	mux.HandleFunc("/api/ai-provider/vendor-image-groups/", a.handleVendorImageGroups)
	mux.HandleFunc("/api/ai-provider/public-image-groups/", a.handlePublicImageGroupIcon)
	mux.HandleFunc("/api/ai-provider/vendor-cloud-server-images/", a.handleVendorCloudServerImages)
	mux.HandleFunc("/api/ai-provider/vendor-userdata-templates/", a.handleVendorUserDataTemplates)

	mux.HandleFunc("/api/ai-provider/admin-container-images/", a.handleAdminContainerImages)
	mux.HandleFunc("/api/ai-provider/admin-cloud-server-images/", a.handleAdminCloudServerImages)
	mux.HandleFunc("/api/ai-provider/admin-userdata-templates/", a.handleAdminUserDataTemplates)

	// Legacy audience-prefixed paths — still the contract documented in
	// openapi.go and consumed by the frontend (frontend/src/views/*, api.js)
	// and the e2e scripts. Commit 41d83e7 renamed these away without updating
	// those consumers, 404ing the whole API contract (nightly regression);
	// keep both registrations until every consumer has migrated.
	mux.HandleFunc("/api/auth/sso/exchange/", a.handleSSOExchange)
	mux.HandleFunc("/api/auth/oidc/authorize/", a.handleOIDCAuthorize)
	mux.HandleFunc("/api/auth/oidc/callback/", a.handleOIDCCallback)

	mux.HandleFunc("/api/vendor/auth/register/", a.handleSSOOnly)
	mux.HandleFunc("/api/vendor/auth/login/", a.handleSSOOnly)
	mux.HandleFunc("/api/vendor/auth/me/", a.handleVendorMe)
	mux.HandleFunc("/api/vendor/status/", a.handleVendorStatus)
	mux.HandleFunc("/api/vendor/application/upload-url/", a.handleVendorApplicationUploadURL)
	mux.HandleFunc("/api/vendor/application/upload-complete/", a.handleVendorApplicationUploadComplete)
	mux.HandleFunc("/api/vendor/application/local-put/", a.handleVendorApplicationLocalPut)
	mux.HandleFunc("/api/vendor/application/upload/", a.handleVendorApplicationUpload)
	mux.HandleFunc("/api/vendor/application/phone-status/", a.handleVendorApplicationPhoneStatus)
	mux.HandleFunc("/api/vendor/application/send-sms/", a.handleVendorApplicationSendSMS)
	mux.HandleFunc("/api/vendor/application/verify-phone/", a.handleVendorApplicationVerifyPhone)
	mux.HandleFunc("/api/vendor/application/", a.handleVendorApplication)
	mux.HandleFunc("/api/admin/vendor-docs-storage/", a.handleAdminVendorDocsStorage)
	mux.HandleFunc("/api/marketplace/settings/", a.handleMarketplaceSettings)
	mux.HandleFunc("/api/admin/marketplace-settings/", a.handleAdminMarketplaceSettings)
	mux.HandleFunc("/api/admin/auth/login/", a.handleSSOOnly)
	mux.HandleFunc("/api/admin/auth/me/", a.handleStaffMe)

	mux.HandleFunc("/api/public/catalog/", a.handlePublicCatalog)
	mux.HandleFunc("/api/public/image-groups/", a.handlePublicImageGroupIcon)
	mux.HandleFunc("/api/public/userdata-templates/", a.handlePublicUserDataTemplates)
	mux.HandleFunc("/api/public/image-runtime-environments/", a.handleImageRuntimeEnvs)
	mux.HandleFunc("/api/public/runtime-userdata/", a.handleRuntimeUserdata)
	mux.HandleFunc("/api/public/vendor-development-catalog/", a.handleVendorDevCatalog)
	mux.HandleFunc("/api/public/unsubmitted-image/", a.handleUnsubmittedImage)

	mux.HandleFunc("/api/admin/vendors/", a.handleAdminVendors)
	mux.HandleFunc("/api/admin/cloud-server-images/userdata-verify/", a.handleUserdataVerify)

	mux.HandleFunc("/api/vendor/cloud-platform-credentials/", a.handleCredentialProxy)
	mux.HandleFunc("/api/vendor/cloud-server-images/regions/", a.handleCloudQueryProxy)
	mux.HandleFunc("/api/vendor/cloud-server-images/images/", a.handleCloudQueryProxy)
	mux.HandleFunc("/api/vendor/cloud-server-images/instance-types/", a.handleCloudQueryProxy)

	mux.HandleFunc("/api/vendor/container-images/", a.handleVendorContainerImages)
	mux.HandleFunc("/api/vendor/image-groups/", a.handleVendorImageGroups)
	mux.HandleFunc("/api/vendor/cloud-server-images/", a.handleVendorCloudServerImages)
	mux.HandleFunc("/api/vendor/userdata-templates/", a.handleVendorUserDataTemplates)

	mux.HandleFunc("/api/admin/container-images/", a.handleAdminContainerImages)
	mux.HandleFunc("/api/admin/cloud-server-images/", a.handleAdminCloudServerImages)
	mux.HandleFunc("/api/admin/userdata-templates/", a.handleAdminUserDataTemplates)

	mux.HandleFunc("/saas-machine-container.md", a.handleSaasMachineContainerSkill)
	mux.HandleFunc("/api/ai-provider/saas-inbound-skill-versions/", a.handleSaasInboundSkillVersions)
	mux.HandleFunc("/", a.handleSPA)
}

func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"detail": "method not allowed"})
		return
	}
	ms, err := a.DB.PingOK()
	db := map[string]any{"ok": err == nil, "latency_ms": ms}
	if err != nil {
		db["error"] = err.Error()
	}
	ok := err == nil
	status := http.StatusOK
	if !ok {
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, map[string]any{
		"service": "ai-provider",
		"ok":      ok,
		"checks":  map[string]any{"database": db},
	})
}

func (a *App) handleSPA(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		writeJSON(w, http.StatusNotFound, map[string]any{"detail": "not found"})
		return
	}
	dist := a.Cfg.FrontendDistDir
	reqPath := r.URL.Path
	if reqPath == "/" || reqPath == "" {
		http.ServeFile(w, r, filepath.Join(dist, "index.html"))
		return
	}
	full := filepath.Join(dist, filepath.Clean("/"+reqPath))
	if !strings.HasPrefix(full, dist) {
		http.NotFound(w, r)
		return
	}
	if st, err := os.Stat(full); err == nil && !st.IsDir() {
		http.ServeFile(w, r, full)
		return
	}
	http.ServeFile(w, r, filepath.Join(dist, "index.html"))
}

func (a *App) handleSchema(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, openAPIDocument())
}

func (a *App) handleSwagger(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, openAPIDocument())
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
