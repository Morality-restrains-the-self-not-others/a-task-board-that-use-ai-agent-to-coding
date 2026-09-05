package main

import (
	"log"
	"net/http"
	"strings"
)

func handleSendVerificationCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	phone := strField(body, "phone")
	email := strField(body, "email")
	if phone == "" && email == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "必须提供手机号或邮箱")
		return
	}
	if phone != "" {
		out, err := sendPhoneVerificationCode(r.Context(), phone, smsKindVerification)
		if err != nil {
			msg := err.Error()
			status := http.StatusBadRequest
			if strings.Contains(msg, "store code") {
				status = http.StatusInternalServerError
				log.Printf("[taskAuth] send verification store error: %v", err)
			}
			writeErrorDetail(w, r, status, msg)
			return
		}
		resp := map[string]interface{}{"message": out.Message}
		if out.RequestID != "" {
			resp["request_id"] = out.RequestID
		}
		if out.SMSProvider != "" {
			resp["sms_provider"] = out.SMSProvider
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}
	out, err := sendEmailVerificationCode(r.Context(), email, smsKindVerification)
	if err != nil {
		msg := err.Error()
		status := http.StatusBadRequest
		if strings.Contains(msg, "store code") {
			status = http.StatusInternalServerError
			log.Printf("[taskAuth] send email verification store error: %v", err)
		} else if strings.Contains(msg, "邮件发送失败") {
			// 上游 Kafka/SMTP 投递失败：502，便于前端与网关区分参数错误 vs 投递故障
			status = http.StatusBadGateway
			log.Printf("[taskAuth] send email verification delivery error: %v", err)
		}
		writeErrorDetail(w, r, status, msg)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"message": out.Message})
}

func handleInternalVerifyVerificationCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		writeErrorDetail(w, r, http.StatusForbidden, "forbidden")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	phone := strField(body, "phone")
	email := strField(body, "email")
	code := strField(body, "code")
	userID := strField(body, "user_id")
	var ok bool
	if phone != "" {
		ok, err = verifyPhoneVerificationCode(phone, code, userID)
	} else if email != "" {
		ok, err = verifyEmailVerificationCode(email, code, userID)
	} else {
		writeErrorDetail(w, r, http.StatusBadRequest, "必须提供手机号或邮箱")
		return
	}
	if err != nil {
		log.Printf("[taskAuth] verify code db error: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"valid": ok})
}
