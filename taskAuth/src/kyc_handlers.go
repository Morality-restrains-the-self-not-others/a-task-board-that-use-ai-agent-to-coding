package main

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"tracelog"
)

func withRequestTrace(r *http.Request) *http.Request {
	tid := tracelog.ResolveTraceID(r.Context(), r.Header.Get("X-Trace-Id"))
	if tid == "" {
		tid = tracelog.ResolveTraceID(r.Context(), r.Header.Get("X-Request-Id"))
	}
	if tid == "" {
		return r
	}
	return r.WithContext(tracelog.ContextWithTraceID(r.Context(), tid))
}

func handleInternalKycProfile(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorDetail(w, r, http.StatusForbidden, "forbidden")
		return
	}
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID := strings.TrimSpace(r.URL.Query().Get("user_id"))
	profile, err := getOrCreateKycProfile(userID)
	if err != nil {
		if errors.Is(err, errKycUserRequired) {
			writeErrorDetail(w, r, http.StatusBadRequest, "user_id required")
			return
		}
		log.Printf("[taskAuth] kyc profile: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func handleInternalKycEvaluate(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorDetail(w, r, http.StatusForbidden, "forbidden")
		return
	}
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	r = withRequestTrace(r)
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	userID := strField(body, "user_id")
	profile, err := evaluateKyc(r.Context(), userID)
	if err != nil {
		if errors.Is(err, errKycUserRequired) {
			writeErrorDetail(w, r, http.StatusBadRequest, "user_id required")
			return
		}
		log.Printf("[taskAuth] kyc evaluate: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func handleInternalKycAdminOverride(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorDetail(w, r, http.StatusForbidden, "forbidden")
		return
	}
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	r = withRequestTrace(r)
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	userID := strField(body, "user_id")
	tier := strField(body, "tier")
	status := strField(body, "status")
	reasonCode := strField(body, "reason_code")
	reasonDetail := strField(body, "reason_detail")
	actorID := strField(body, "actor_id")
	if !isValidKycTier(tier) {
		writeErrorDetail(w, r, http.StatusBadRequest, "invalid tier")
		return
	}
	if !isValidKycStatus(status) {
		writeErrorDetail(w, r, http.StatusBadRequest, "invalid status")
		return
	}
	profile, err := adminOverrideKyc(r.Context(), userID, tier, status, reasonCode, reasonDetail, actorID)
	if err != nil {
		if errors.Is(err, errKycUserRequired) {
			writeErrorDetail(w, r, http.StatusBadRequest, "user_id required")
			return
		}
		log.Printf("[taskAuth] kyc admin-override: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func handleInternalKycAudit(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorDetail(w, r, http.StatusForbidden, "forbidden")
		return
	}
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID := strings.TrimSpace(r.URL.Query().Get("user_id"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := listKycAudit(userID, limit)
	if err != nil {
		if errors.Is(err, errKycUserRequired) {
			writeErrorDetail(w, r, http.StatusBadRequest, "user_id required")
			return
		}
		log.Printf("[taskAuth] kyc audit: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if items == nil {
		items = []kycAuditEntry{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"results": items})
}

func handleInternalKycAmlScreening(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorDetail(w, r, http.StatusForbidden, "forbidden")
		return
	}
	switch r.Method {
	case http.MethodGet:
		userID := strings.TrimSpace(r.URL.Query().Get("user_id"))
		if userID == "" {
			writeErrorDetail(w, r, http.StatusBadRequest, "user_id required")
			return
		}
		rec, err := latestAmlScreeningRecord(userID)
		if err != nil {
			log.Printf("[taskAuth] kyc aml-screening get: %v", err)
			writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
			return
		}
		if rec == nil {
			writeJSON(w, http.StatusOK, map[string]interface{}{"latest": nil})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"latest": rec})
		return
	case http.MethodPost:
		// continue below
	default:
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	r = withRequestTrace(r)
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	rec, err := recordAmlScreening(
		r.Context(),
		strField(body, "user_id"),
		strField(body, "result"),
		strField(body, "provider"),
		strField(body, "notes"),
		strField(body, "actor_id"),
		strField(body, "screening_ref"),
	)
	if err != nil {
		if errors.Is(err, errKycUserRequired) {
			writeErrorDetail(w, r, http.StatusBadRequest, "user_id required")
			return
		}
		if errors.Is(err, errAmlInvalidResult) {
			writeErrorDetail(w, r, http.StatusBadRequest, "invalid result")
			return
		}
		log.Printf("[taskAuth] kyc aml-screening: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

func handleInternalKycLimitPolicy(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorDetail(w, r, http.StatusForbidden, "forbidden")
		return
	}
	switch r.Method {
	case http.MethodGet:
		items, err := listKycLimitPolicies()
		if err != nil {
			log.Printf("[taskAuth] kyc limit-policy get: %v", err)
			writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
			return
		}
		if items == nil {
			items = []kycLimitPolicy{}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"results": items})
	case http.MethodPut:
		body, err := readJSONBody(r)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid json")
			return
		}
		updated, err := applyLimitPolicyUpdates(body)
		if err != nil {
			if errors.Is(err, errKycInvalidTier) {
				writeErrorDetail(w, r, http.StatusBadRequest, "invalid tier")
				return
			}
			log.Printf("[taskAuth] kyc limit-policy put: %v", err)
			writeErrorDetail(w, r, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"results": updated})
	default:
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func applyLimitPolicyUpdates(body map[string]interface{}) ([]kycLimitPolicy, error) {
	var specs []map[string]interface{}
	if raw, ok := body["policies"]; ok {
		switch t := raw.(type) {
		case []interface{}:
			for _, item := range t {
				if m, ok := item.(map[string]interface{}); ok {
					specs = append(specs, m)
				}
			}
		}
	} else if strField(body, "tier") != "" {
		specs = append(specs, body)
	} else {
		return nil, errors.New("tier or policies required")
	}
	var out []kycLimitPolicy
	for _, spec := range specs {
		tier := strField(spec, "tier")
		maxSingle := intFromBody(spec, "max_single_yuan")
		maxDaily := intFromBody(spec, "max_daily_yuan")
		allowed := true
		if v, ok := spec["recharge_allowed"]; ok && v != nil {
			switch t := v.(type) {
			case bool:
				allowed = t
			case float64:
				allowed = t != 0
			default:
				s := strField(spec, "recharge_allowed")
				allowed = s == "true" || s == "1"
			}
		}
		p, err := upsertKycLimitPolicy(tier, maxSingle, maxDaily, allowed)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func intFromBody(body map[string]interface{}, key string) int {
	v, ok := body[key]
	if !ok || v == nil {
		return 0
	}
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	default:
		n, _ := strconv.Atoi(strField(body, key))
		return n
	}
}

func handleInternalKycRechargeGate(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorDetail(w, r, http.StatusForbidden, "forbidden")
		return
	}
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID := strings.TrimSpace(r.URL.Query().Get("user_id"))
	amountRaw := strings.TrimSpace(r.URL.Query().Get("amount_yuan"))
	amount, err := strconv.ParseInt(amountRaw, 10, 64)
	if err != nil {
		writeErrorDetail(w, r, http.StatusBadRequest, "amount_yuan required")
		return
	}
	result, err := checkKycRechargeGate(userID, amount)
	if err != nil {
		if errors.Is(err, errKycUserRequired) {
			writeErrorDetail(w, r, http.StatusBadRequest, "user_id required")
			return
		}
		log.Printf("[taskAuth] kyc recharge-gate: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, result)
}
