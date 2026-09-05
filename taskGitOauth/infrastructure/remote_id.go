package infrastructure

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// FormatRemoteUserID converts OAuth profile id fields to a stable decimal string.
// JSON numbers decode as float64 in map[string]any; fmt.Sprint then yields
// scientific notation (e.g. 1.321779e+06) which Django normalize_github_user_id rejects.
func FormatRemoteUserID(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return ""
		}
		if _, err := strconv.ParseInt(s, 10, 64); err == nil {
			return s
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return formatFloatID(f)
		}
		return s
	case json.Number:
		return strings.TrimSpace(t.String())
	case int:
		return strconv.Itoa(t)
	case int32:
		return strconv.FormatInt(int64(t), 10)
	case int64:
		return strconv.FormatInt(t, 10)
	case uint64:
		return strconv.FormatUint(t, 10)
	case float32:
		return formatFloatID(float64(t))
	case float64:
		return formatFloatID(t)
	default:
		s := strings.TrimSpace(fmt.Sprint(t))
		if s == "" || s == "<nil>" {
			return ""
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return formatFloatID(f)
		}
		return s
	}
}

func formatFloatID(f float64) string {
	if math.IsNaN(f) {
		return ""
	}
	if f == float64(int64(f)) && f >= -9e15 && f <= 9e15 {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}
