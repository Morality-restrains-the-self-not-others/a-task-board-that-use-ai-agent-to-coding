package main

import (
	"errors"
	"log"
	"log/slog"
	"net/http"
	"strings"

	"taskAuth/domain"
)

// handleBindPhone POST /api/accounts/users/bind_phone/
// 以及资料页 POST /api/accounts/users/profile/bind-phone/ 与 replace-phone/。
// 登录态绑定手机号（v64 S2 场景: 微信/邮箱登录账号补手机号）。
// 语义为「绑定」而非「注册」— 验证码确认后 upsert 到当前账号；
// 手机号已被其他活跃账号使用 → 409（code=phone_taken，reclaim_available）。
// reclaim=true 且短信校验通过后，作废对方绑定并转移到本账号（持有 SIM 即所有权）。
func handleBindPhone(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, err := resolveTokenUserIDFromRequest(r)
	if err != nil {
		writeErrorDetail(w, r, http.StatusUnauthorized, "未登录")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	phone := strings.TrimSpace(strField(body, "phone"))
	code := strings.TrimSpace(strField(body, "code"))
	reclaim := boolField(body, "reclaim")
	if phone == "" || code == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "必须提供手机号与验证码")
		return
	}

	codeID, err := findUnusedPhoneVerificationCodeID(phone, code)
	if err != nil {
		slog.ErrorContext(r.Context(), "bind_phone_verify_db_error", "user_id", userID, "err", err.Error())
		writeErrorDetail(w, r, http.StatusInternalServerError, "内部错误")
		return
	}
	if codeID == 0 {
		writeErrorDetail(w, r, http.StatusBadRequest, "验证码无效或已过期")
		return
	}

	canonical := canonicalPhoneForSMSAndLogin(phone)
	cc, national := splitCountryCallingCodeAndNational(canonical)
	if cc == "" || national == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "手机号格式无效")
		return
	}

	d, otherCount, err := phoneBindDecision(userID, cc, national)
	if err != nil {
		slog.ErrorContext(r.Context(), "bind_phone_taken_lookup_error", "user_id", userID, "err", err.Error())
		writeErrorDetail(w, r, http.StatusInternalServerError, "绑定失败")
		return
	}
	if (d == domain.PhoneShareLimit || d == domain.PhoneShareTaken) && !reclaim {
		slog.InfoContext(r.Context(), "phone_bind_rejected", "user_id", userID, "reason", string(d), "holder_count", otherCount)
		writePhoneBindConflict(w, r, d)
		return
	}

	if err := consumeVerificationCodeID(codeID, userID); err != nil {
		slog.ErrorContext(r.Context(), "bind_phone_consume_code_error", "user_id", userID, "err", err.Error())
		writeErrorDetail(w, r, http.StatusInternalServerError, "内部错误")
		return
	}

	if reclaim {
		if err := reclaimPhoneLoginMethod(userID, cc, national); err != nil {
			if errors.Is(err, errPhoneTaken) {
				writeErrorDetail(w, r, http.StatusConflict, "该手机号已绑定其他账号")
				return
			}
			slog.ErrorContext(r.Context(), "bind_phone_reclaim_error", "user_id", userID, "err", err.Error())
			writeErrorDetail(w, r, http.StatusInternalServerError, "绑定失败")
			return
		}
	} else if err := upsertPhoneLoginMethod(userID, cc, national); err != nil {
		if errors.Is(err, errPhoneBindLimit) {
			writePhoneBindConflict(w, r, domain.PhoneShareLimit)
			return
		}
		if errors.Is(err, errPhoneTaken) {
			writePhoneBindConflict(w, r, domain.PhoneShareTaken)
			return
		}
		log.Printf("[taskAuth] bind phone upsert user=%s: %v", userID, err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "绑定失败")
		return
	}

	holderCount := otherCount + 1
	if reclaim {
		holderCount = 1
	}
	slog.InfoContext(r.Context(), "phone_bound", "user_id", userID, "reclaim", reclaim, "shared", holderCount > 1, "holder_count", holderCount)
	maybeEvaluateKycAfterPhoneVerified(withRequestTrace(r).Context(), userID)
	writeJSON(w, http.StatusOK, map[string]interface{}{"bound": true, "phone": national})
}

// writePhoneBindConflict 写出绑定 409：占用判定与 upsert 失败共用同一错误体。
func writePhoneBindConflict(w http.ResponseWriter, r *http.Request, d domain.PhoneShareDecision) {
	switch d {
	case domain.PhoneShareLimit:
		writeErrorMap(w, r, http.StatusConflict, map[string]interface{}{
			"error":             "该手机号已达绑定上限",
			"detail":            "该手机号已绑定 5 个账号，无法再绑定。请解绑其中一个账号，或改用邮箱/用户名登录。",
			"code":              "phone_bind_limit",
			"reclaim_available": false,
			"limit":             domain.MaxSharedPhoneBindings,
		})
	case domain.PhoneShareTaken:
		writeErrorMap(w, r, http.StatusConflict, map[string]interface{}{
			"error":             "该手机号已绑定其他账号",
			"detail":            "该手机号已绑定其他账号。若这是您本人正在使用的号码，请确认转移到本账号；原账号将自动解除该号码。",
			"code":              "phone_taken",
			"reclaim_available": true,
		})
	default:
		slog.ErrorContext(r.Context(), "phone_bind_conflict_unexpected_decision", "decision", string(d))
		writeErrorDetail(w, r, http.StatusInternalServerError, "绑定失败")
	}
}
