package infrastructure

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ParseHS256JWT verifies a compact HS256 JWT and returns claims.
// Validates optional exp/iat/nbf, iss, aud, and required typ claim when wantTyp != "".
func ParseHS256JWT(token, secret, wantIss, wantAud, wantTyp string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid jwt format")
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("bad jwt header")
	}
	var header map[string]any
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, fmt.Errorf("bad jwt header json")
	}
	if alg, _ := header["alg"].(string); !strings.EqualFold(alg, "HS256") {
		return nil, fmt.Errorf("unexpected alg")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(parts[0] + "." + parts[1]))
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("bad jwt sig")
	}
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return nil, fmt.Errorf("bad jwt signature")
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("bad jwt payload")
	}
	var claims map[string]any
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, fmt.Errorf("bad jwt claims")
	}
	now := time.Now().Unix()
	if wantTyp != "" {
		exp, ok := claimInt64(claims["exp"])
		if !ok {
			return nil, fmt.Errorf("exp required")
		}
		if now >= exp {
			return nil, fmt.Errorf("jwt expired")
		}
		if _, ok := claimInt64(claims["iat"]); !ok {
			return nil, fmt.Errorf("iat required")
		}
		if strings.TrimSpace(fmt.Sprint(claims["sub"])) == "" || fmt.Sprint(claims["sub"]) == "<nil>" {
			return nil, fmt.Errorf("sub required")
		}
		if typ, _ := claims["typ"].(string); typ != wantTyp {
			return nil, fmt.Errorf("bad typ")
		}
	} else if exp, ok := claimInt64(claims["exp"]); ok && now >= exp {
		return nil, fmt.Errorf("jwt expired")
	}
	if nbf, ok := claimInt64(claims["nbf"]); ok && now < nbf {
		return nil, fmt.Errorf("jwt not yet valid")
	}
	if wantIss != "" {
		if iss, _ := claims["iss"].(string); iss != wantIss {
			return nil, fmt.Errorf("bad iss")
		}
	}
	if wantAud != "" && !audienceOK(claims["aud"], wantAud) {
		return nil, fmt.Errorf("bad aud")
	}
	return claims, nil
}

func claimInt64(v any) (int64, bool) {
	switch t := v.(type) {
	case float64:
		return int64(t), true
	case json.Number:
		n, err := t.Int64()
		return n, err == nil
	case int64:
		return t, true
	case int:
		return int64(t), true
	default:
		return 0, false
	}
}

func audienceOK(aud any, want string) bool {
	switch t := aud.(type) {
	case string:
		return t == want
	case []any:
		for _, v := range t {
			if s, ok := v.(string); ok && s == want {
				return true
			}
		}
	}
	return false
}
