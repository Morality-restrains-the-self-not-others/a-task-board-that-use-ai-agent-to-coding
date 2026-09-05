package main

import (
	"bytes"
	"compress/zlib"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"
)

// ---- direct DB session resolve (removes HTTP dependency on saas-backend) ---- //

// resolveUserIDFromDjangoSessionCookie resolves a Django sessionid cookie to a user_id.
// Strategy: try direct MySQL read of saas.django_session first; fall back to HTTP on error.
func resolveUserIDFromDjangoSessionCookie(sessionKey string) (string, error) {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return "", fmt.Errorf("empty sessionid")
	}

	// Primary path: direct MySQL read (no HTTP, no Django dependency)
	if uid, err := resolveUserIDFromDjangoDBSession(sessionKey); err == nil {
		return uid, nil
	}

	// saas-backend 已退役（2026-07-30，OPT-049）：HTTP fallback 路径移除，
	// 直连 MySQL 读取失败即报错。
	return "", fmt.Errorf("session resolve failed")
}

// resolveUserIDFromDjangoDBSession reads the django_session table directly from MySQL.
// This eliminates the taskAuth → Django HTTP dependency (OPT-20260729-022).
func resolveUserIDFromDjangoDBSession(sessionKey string) (string, error) {
	if db == nil {
		return "", fmt.Errorf("db not available")
	}

	var sessionData string
	var expireDate time.Time
	err := db.QueryRow(
		`SELECT session_data, expire_date FROM saas.django_session
		 WHERE session_key = ? AND expire_date > NOW()`,
		sessionKey,
	).Scan(&sessionData, &expireDate)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("session not found")
	}
	if err != nil {
		return "", fmt.Errorf("session db query: %w", err)
	}

	session, err := decodeDjangoSessionData(sessionData)
	if err != nil {
		return "", fmt.Errorf("session decode: %w", err)
	}

	uid, _ := session["_auth_user_id"].(string)
	if v, ok := session["_auth_user_id"].(float64); ok {
		uid = fmt.Sprintf("%.0f", v)
	}
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return "", fmt.Errorf("session missing _auth_user_id")
	}
	return uid, nil
}

// decodeDjangoSessionData decodes Django's signed session_data format.
//
// Django 4.2 SessionBase.encode() produces:
//
//	Timestamper.sign(  "."?  base64url(json or zlib(json))  )
//	                  flag   payload
//
// TimestampSigner wraps as:  payload + ":" + base64(ts) + ":" + base64(HMAC-SHA256)
//
// We skip signature verification (safe for internal DB reads) and extract only the payload.
func decodeDjangoSessionData(sessionData string) (map[string]interface{}, error) {
	s := strings.TrimSpace(sessionData)
	if s == "" {
		return nil, fmt.Errorf("empty session_data")
	}

	// 1. Split off signature (last ":" segment)
	lastColon := strings.LastIndex(s, ":")
	if lastColon < 0 {
		return nil, fmt.Errorf("malformed session_data: no separator")
	}
	withoutSig := s[:lastColon]

	// 2. Split off timestamp (second-to-last ":" segment)
	tsColon := strings.LastIndex(withoutSig, ":")
	if tsColon < 0 {
		// No timestamp? Treat entire string as payload (older Django format)
		return decodeSessionPayload(s)
	}
	payload := withoutSig[:tsColon]

	return decodeSessionPayload(payload)
}

// decodeSessionPayload decodes the inner payload (base64url JSON or zlib-compressed).
func decodeSessionPayload(payload string) (map[string]interface{}, error) {
	var raw []byte

	// Check for compression flag (Django prepends '.' to indicate zlib)
	if strings.HasPrefix(payload, ".") {
		compressed, err := b64DecodeDjango(strings.TrimPrefix(payload, "."))
		if err != nil {
			return nil, fmt.Errorf("base64 decode compressed: %w", err)
		}
		r, err := zlib.NewReader(bytes.NewReader(compressed))
		if err != nil {
			return nil, fmt.Errorf("zlib reader: %w", err)
		}
		defer r.Close()
		raw, err = io.ReadAll(r)
		if err != nil {
			return nil, fmt.Errorf("zlib decompress: %w", err)
		}
	} else {
		var err error
		raw, err = b64DecodeDjango(payload)
		if err != nil {
			return nil, fmt.Errorf("base64 decode: %w", err)
		}
	}

	var session map[string]interface{}
	if err := json.Unmarshal(raw, &session); err != nil {
		// Try the raw bytes as a plain string (non-JSON serializer)
		return nil, fmt.Errorf("json parse session: %w", err)
	}
	return session, nil
}

// b64DecodeDjango mirrors Django's django.core.signing.b64_decode:
//
//	base64.urlsafe_b64decode(s + padding)
func b64DecodeDjango(s string) ([]byte, error) {
	// Django uses URL-safe base64 without padding.
	missing := (4 - len(s)%4) % 4
	s += strings.Repeat("=", missing)
	return base64.URLEncoding.DecodeString(s)
}

// ---- helpers ---- //

func sessionIDFromCookieHeader(cookieHeader string) string {
	cookieHeader = strings.TrimSpace(cookieHeader)
	if cookieHeader == "" {
		return ""
	}
	parts := strings.Split(cookieHeader, ";")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, "sessionid=") {
			return strings.TrimSpace(strings.TrimPrefix(p, "sessionid="))
		}
	}
	return ""
}

func isPrintableSessionKey(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < 33 || r > 126 {
			return false
		}
	}
	return true
}

// logSessionResolvePath records which path was used for session resolution (DB vs HTTP).
func logSessionResolvePath(path string, sessionKeyPrefix string, elapsed time.Duration) {
	if path != "db" {
		log.Printf("[taskAuth] session resolve path=%s key=%s... elapsed=%v", path, sessionKeyPrefix, elapsed)
	}
}
