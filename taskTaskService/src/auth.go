package main

import (
	"net/http"

	"gatewayauth"
)

func verifyJWT(r *http.Request) (userID string, tenantID string, err error) {
	return gatewayauth.VerifyWithTaskAuth(r, cfg.TaskAuthURL)
}

func resolveGatewayAuthUser(r *http.Request) string {
	return gatewayauth.UserFromGatewayHeaders(r, cfg.TaskGatewayInternalSecret)
}

func getAuthUser(r *http.Request) string { return r.Header.Get(gatewayauth.HeaderAuthUserID) }
func getAuthTenant(r *http.Request) string {
	return r.Header.Get(gatewayauth.HeaderAuthTenantID)
}
