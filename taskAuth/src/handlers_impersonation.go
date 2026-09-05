package main

import (
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"authz"
	"taskAuth/domain"

	"tracelog"
)

const impersonatorRestoreCookie = "impersonatorRestore"

func requestAuthToken(r *http.Request) string {
	tok := strings.TrimSpace(tokenFromRequest(r))
	if tok != "" {
		return tok
	}
	if c, err := r.Cookie("token"); err == nil {
		return strings.TrimSpace(c.Value)
	}
	return ""
}

func handleStartImpersonation(w http.ResponseWriter, r *http.Request, targetUserID string) {
	targetUserID = strings.TrimSpace(targetUserID)
	traceID := tracelog.TraceIDFromContext(r.Context())
	incoming := requestAuthToken(r)

	actorID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	if !authz.RequirePlatformPerm(w, r, authz.PermUserImpersonate) {
		slog.WarnContext(r.Context(), "impersonation denied",
			"event", "impersonation_denied",
			"reason", "missing_perm",
			"actor_user_id", actorID,
			"target_user_id", targetUserID,
			"trace_id", traceID)
		return
	}

	exists, err := userExists(targetUserID)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if !exists {
		writeErrorDetail(w, r, http.StatusNotFound, "user not found")
		return
	}

	idemKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if existing, eerr := findOpenImpersonationByIdempotency(actorID, idemKey); eerr == nil && existing != nil {
		if existing.TargetUserID != targetUserID {
			writeErrorDetail(w, r, http.StatusConflict, "idempotency key reused for a different user")
			return
		}
		writeImpersonationStartResponse(w, r, existing, incoming)
		return
	} else if eerr != nil && eerr != sql.ErrNoRows {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}

	reason, rerr := parseImpersonationReasonBody(r)
	if rerr != nil {
		writeErrorDetail(w, r, http.StatusBadRequest, rerr.Error())
		return
	}

	if handled := resumeOrRejectOpenImpersonation(w, r, actorID, targetUserID, incoming, traceID); handled {
		return
	}
	if err := endStaleOpenImpersonationForOtherTarget(r, actorID, targetUserID, incoming, traceID); err != nil {
		slog.ErrorContext(r.Context(), "impersonation stale session close failed",
			"event", "impersonation_error", "err", err.Error(), "trace_id", traceID)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}

	targetActive, err := userIsActive(targetUserID)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	targetAdmin, err := userIsPlatformAdmin(targetUserID)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}

	decErr := domain.ValidateStart(domain.StartImpersonationDecision{
		ActorUserID:               actorID,
		TargetUserID:              targetUserID,
		ActorHasPlatformManage:    authz.HasPlatformPerm(r, authz.PermPlatformManage),
		TargetIsPlatformAdmin:     targetAdmin,
		ActorAlreadyImpersonating: isImpersonationToken(incoming),
		TargetActive:              targetActive,
		IdempotencyKey:            idemKey,
		Reason:                    reason,
	})
	if decErr != nil {
		writeImpersonationDomainError(w, r, decErr)
		return
	}

	token, err := generateImpersonationToken()
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "token error")
		return
	}
	sess, msgID, err := insertImpersonationSession(actorID, targetUserID, token, idemKey, reason, impersonationTTL)
	if err != nil {
		slog.ErrorContext(r.Context(), "impersonation insert failed",
			"event", "impersonation_error", "err", err.Error(), "trace_id", traceID)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}

	slog.InfoContext(r.Context(), "impersonation started",
		"event", "impersonation_started",
		"actor_user_id", actorID,
		"target_user_id", targetUserID,
		"session_id", sess.ID,
		"reason_len", utf8.RuneCountInString(reason),
		"trace_id", traceID)
	go publishUserImpersonationStarted(r.Context(), actorID, targetUserID, sess.ID, utf8.RuneCountInString(reason))
	go publishUserInboxMessageCreated(r.Context(), msgID, targetUserID, actorID, sess.ID)

	writeImpersonationStartResponse(w, r, sess, incoming)
}

func writeImpersonationDomainError(w http.ResponseWriter, r *http.Request, err error) {
	switch err {
	case domain.ErrMissingIdempotencyKey, domain.ErrActorOrTargetRequired, domain.ErrMissingReason, domain.ErrReasonTooLong, domain.ErrReasonIsPromptText:
		writeErrorDetail(w, r, http.StatusBadRequest, err.Error())
	case domain.ErrSelfImpersonation:
		writeErrorDetail(w, r, http.StatusConflict, err.Error())
	case domain.ErrNestedImpersonation:
		writeErrorDetail(w, r, http.StatusConflict, domain.NestedImpersonationClientMessage)
	case domain.ErrTargetInactive:
		writeErrorDetail(w, r, http.StatusUnprocessableEntity, err.Error())
	case domain.ErrPrivilegeEscalation:
		writeErrorDetail(w, r, http.StatusForbidden, err.Error())
	default:
		writeErrorDetail(w, r, http.StatusBadRequest, err.Error())
	}
}

func parseImpersonationReasonBody(r *http.Request) (string, error) {
	if r.Body == nil {
		return "", domain.ErrMissingReason
	}
	var body struct {
		Reason string `json:"reason"`
	}
	dec := json.NewDecoder(io.LimitReader(r.Body, 16<<10))
	if err := dec.Decode(&body); err != nil {
		return "", domain.ErrMissingReason
	}
	return domain.NormalizeImpersonationReason(body.Reason), nil
}

func writeImpersonationStartResponse(w http.ResponseWriter, r *http.Request, sess *impersonationSessionRow, actorToken string) {
	if strings.TrimSpace(actorToken) != "" && !isImpersonationToken(actorToken) {
		http.SetCookie(w, &http.Cookie{
			Name:     impersonatorRestoreCookie,
			Value:    actorToken,
			Path:     "/",
			Domain:   ssoCookieDomain(),
			MaxAge:   int(impersonationTTL.Seconds()),
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			Secure:   false,
		})
	}
	resp := buildLoginResponse(sess.TargetUserID, sess.TokenKey)
	if user, ok := resp["user"].(map[string]interface{}); ok {
		resp["redirect_url"] = impersonationLandingRedirect(user)
	}
	resp["impersonation"] = map[string]interface{}{
		"actor_user_id":  sess.ActorUserID,
		"target_user_id": sess.TargetUserID,
		"expires_at":     sess.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z"),
		"session_id":     strconv.FormatInt(sess.ID, 10),
	}
	writeJSON(w, http.StatusOK, resp)
}

func handleStopImpersonation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	tok := requestAuthToken(r)
	sess, err := lookupImpersonationByToken(tok)
	if err == sql.ErrNoRows || sess == nil {
		writeErrorDetail(w, r, http.StatusConflict, domain.ErrNotImpersonating.Error())
		return
	}
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}

	justEnded := false
	if !sess.EndedAt.Valid {
		if err := endImpersonationSession(tok); err != nil {
			writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
			return
		}
		justEnded = true
	}

	restore := ""
	if c, err := r.Cookie(impersonatorRestoreCookie); err == nil {
		restore = strings.TrimSpace(c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     impersonatorRestoreCookie,
		Value:    "",
		Path:     "/",
		Domain:   ssoCookieDomain(),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	if justEnded {
		traceID := tracelog.TraceIDFromContext(r.Context())
		slog.InfoContext(r.Context(), "impersonation stopped",
			"event", "impersonation_stopped",
			"actor_user_id", sess.ActorUserID,
			"target_user_id", sess.TargetUserID,
			"session_id", sess.ID,
			"trace_id", traceID)
		go publishUserImpersonationStopped(r.Context(), sess.ActorUserID, sess.TargetUserID, sess.ID)
	}

	if restore == "" {
		restore, _ = getOrCreateToken(sess.ActorUserID, cfg.UserContentTypeID, resolveClientIP(r))
	}
	resp := buildLoginResponse(sess.ActorUserID, restore)
	resp["redirect_url"] = "/system-admin/users/"
	writeJSON(w, http.StatusOK, resp)
}

func handleImpersonationStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if _, ok := requireAuthenticatedUser(w, r); !ok {
		return
	}
	sess, err := lookupOpenImpersonationByToken(requestAuthToken(r))
	if err != nil || sess == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"impersonating": false})
		return
	}
	uname := sess.TargetUserID
	landing := "/onboarding/"
	if detail, derr := buildUserDetailJSON(sess.TargetUserID); derr == nil {
		if s, ok := detail["username"].(string); ok && s != "" {
			uname = s
		}
		landing = impersonationLandingRedirect(detail)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"impersonating":   true,
		"actor_user_id":   sess.ActorUserID,
		"target_user_id":  sess.TargetUserID,
		"target_username": uname,
		"expires_at":      sess.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z"),
		"session_id":      strconv.FormatInt(sess.ID, 10),
		"redirect_url":    landing,
	})
}

func resumeOrRejectOpenImpersonation(w http.ResponseWriter, r *http.Request, actorID, targetUserID, incoming, traceID string) bool {
	if sess, err := lookupOpenImpersonationByToken(incoming); err == nil && sess != nil {
		if sess.TargetUserID == targetUserID {
			slog.InfoContext(r.Context(), "impersonation resumed",
				"event", "impersonation_resumed",
				"actor_user_id", sess.ActorUserID,
				"target_user_id", targetUserID,
				"session_id", sess.ID,
				"trace_id", traceID)
			writeImpersonationStartResponse(w, r, sess, incoming)
			return true
		}
		slog.WarnContext(r.Context(), "impersonation denied",
			"event", "impersonation_denied",
			"reason", "nested",
			"actor_user_id", sess.ActorUserID,
			"target_user_id", targetUserID,
			"session_id", sess.ID,
			"trace_id", traceID)
		writeImpersonationDomainError(w, r, domain.ErrNestedImpersonation)
		return true
	} else if err != nil && err != sql.ErrNoRows {
		slog.ErrorContext(r.Context(), "impersonation lookup failed",
			"event", "impersonation_error", "err", err.Error(), "trace_id", traceID)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return true
	}

	open, err := findOpenImpersonationForActor(actorID)
	if err == sql.ErrNoRows {
		return false
	}
	if err != nil {
		slog.ErrorContext(r.Context(), "impersonation lookup failed",
			"event", "impersonation_error", "err", err.Error(), "trace_id", traceID)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return true
	}
	if open != nil && open.TargetUserID == targetUserID {
		slog.InfoContext(r.Context(), "impersonation resumed",
			"event", "impersonation_resumed",
			"actor_user_id", open.ActorUserID,
			"target_user_id", targetUserID,
			"session_id", open.ID,
			"trace_id", traceID)
		writeImpersonationStartResponse(w, r, open, incoming)
		return true
	}
	return false
}

func endStaleOpenImpersonationForOtherTarget(r *http.Request, actorID, targetUserID, incoming, traceID string) error {
	if isImpersonationToken(incoming) {
		return nil
	}
	open, err := findOpenImpersonationForActor(actorID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if open == nil || open.TargetUserID == targetUserID {
		return nil
	}
	if err := endImpersonationSession(open.TokenKey); err != nil {
		return err
	}
	slog.InfoContext(r.Context(), "impersonation superseded",
		"event", "impersonation_superseded",
		"actor_user_id", actorID,
		"previous_target_user_id", open.TargetUserID,
		"target_user_id", targetUserID,
		"session_id", open.ID,
		"trace_id", traceID)
	go publishUserImpersonationStopped(r.Context(), actorID, open.TargetUserID, open.ID)
	return nil
}
