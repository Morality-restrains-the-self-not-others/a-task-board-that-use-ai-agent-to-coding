package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
)

func optionalInt64(v any) *int64 {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case float64:
		n := int64(t)
		return &n
	case int64:
		return &t
	case int:
		n := int64(t)
		return &n
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return nil
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil
		}
		return &n
	default:
		return nil
	}
}

func asInt64(v any) (int64, bool) {
	p := optionalInt64(v)
	if p == nil {
		return 0, false
	}
	return *p, true
}

func accessTokenFingerprint(token string) string {
	raw := strings.TrimSpace(token)
	if raw == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func sanitizeAuditDetail(raw map[string]any) map[string]any {
	if raw == nil {
		return map[string]any{}
	}
	out := map[string]any{}
	i := 0
	for k, v := range raw {
		if i >= 48 {
			break
		}
		i++
		ks := k
		if len(ks) > 128 {
			ks = ks[:128]
		}
		lower := strings.ToLower(ks)
		if strings.Contains(lower, "token") || strings.Contains(lower, "secret") || strings.Contains(lower, "authorization") {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				out[ks] = "<redacted>"
			} else {
				out[ks] = v
			}
			continue
		}
		switch t := v.(type) {
		case bool, nil:
			out[ks] = t
		case float64, int, int64:
			out[ks] = t
		case string:
			if len(t) > 2048 {
				t = t[:2048]
			}
			out[ks] = t
		default:
			b, err := json.Marshal(t)
			if err != nil {
				out[ks] = "<unserializable>"
			} else {
				s := string(b)
				if len(s) > 4096 {
					s = s[:4096]
				}
				out[ks] = s
			}
		}
	}
	blob, _ := json.Marshal(out)
	if len(blob) > 12000 {
		preview := string(blob)
		if len(preview) > 8000 {
			preview = preview[:8000]
		}
		return map[string]any{"truncated": true, "preview": preview}
	}
	return out
}
