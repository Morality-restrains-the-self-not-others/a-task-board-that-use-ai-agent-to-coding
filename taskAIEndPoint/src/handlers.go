package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"gatewaycors"
)

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "taskAIEndPoint"})
}

func handleLlmProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed"})
		return
	}

	match, ok := parseLlmProxyPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	ctx := r.Context()
	token := extractBearerToken(r.Header.Get("Authorization"))
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"detail": "missing proxy token"})
		return
	}
	if valid, detail := validateProxyToken(ctx, match.TenantID, match.WorkspaceID, match.TaskID, token); !valid {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"detail": detail})
		return
	}

	route, status, body := cloudResolveRoute(ctx, match.TenantID, match.WorkspaceID, match.TaskID, match.Provider)
	if status != http.StatusOK || route == nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		_, _ = w.Write(body)
		return
	}
	if !route.UseSubToken {
		writeJSON(w, http.StatusForbidden, map[string]string{"detail": "provider is not sub-token enabled"})
		return
	}

	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "failed to read body"})
		return
	}

	chatReq, chatMap, err := parseChatRequest(rawBody)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "invalid json body"})
		return
	}
	if chatReq.Model == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "model is required"})
		return
	}

	if route.BudgetEnabled {
		dec, bStatus, bBody := cloudBudgetGate(
			ctx,
			match.TenantID, match.WorkspaceID, match.TaskID,
			match.Provider, route.UpstreamBase, chatReq.Model,
		)
		if bStatus == http.StatusPaymentRequired || (dec != nil && !dec.Allowed) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusPaymentRequired)
			if dec != nil {
				_, _ = w.Write(budgetExhaustedPayload(dec))
			} else {
				_, _ = w.Write(bBody)
			}
			return
		}
		if bStatus != http.StatusOK {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(bStatus)
			_, _ = w.Write(bBody)
			return
		}
	}

	cred, cStatus, cBody := cloudUpstreamCredential(ctx, match.TenantID, match.Provider, route.UpstreamBase)
	if cStatus != http.StatusOK || cred == nil || strings.TrimSpace(cred.APIKey) == "" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(cStatus)
		_, _ = w.Write(cBody)
		return
	}

	upstreamBody := rawBody
	if chatReq.Stream {
		upstreamBody = injectStreamUsageOption(chatMap)
	}

	upstreamURL := fmtUpstreamURL(ensureUpstreamV1Base(route.UpstreamBase), match.SubPath)
	inTok, outTok := forwardUpstream(w, r, upstreamURL, cred.APIKey, upstreamBody, chatReq.Stream)

	if route.BudgetEnabled && (inTok > 0 || outTok > 0) {
		idem := strings.TrimSpace(r.Header.Get("X-Request-Id"))
		if idem == "" {
			idem = strings.TrimSpace(r.Header.Get("X-Trace-Id"))
		}
		cloudCommitUsage(ctx, commitUsageRequest{
			TenantID:       match.TenantID,
			WorkspaceID:    match.WorkspaceID,
			TaskID:         match.TaskID,
			Provider:       match.Provider,
			BaseURL:        route.UpstreamBase,
			ModelName:      chatReq.Model,
			InputTokens:    inTok,
			OutputTokens:   outTok,
			IdempotencyKey: idem,
			RequestID:      idem,
		})
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", gatewaycors.AllowHeadersWith("X-Request-Id"))
		w.Header().Set("Access-Control-Expose-Headers", gatewaycors.ExposeHeaders)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func mountRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/health/", handleHealth)
	mux.HandleFunc("/api/tenant/", handleLlmProxy)
}
