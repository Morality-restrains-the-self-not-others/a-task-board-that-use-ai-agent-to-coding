package payload

import (
	"encoding/json"
	"fmt"
	"strconv"
)

func MapField(data map[string]interface{}, key string) map[string]interface{} {
	if data == nil {
		return nil
	}
	v, ok := data[key]
	if !ok || v == nil {
		return nil
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	return m
}

func StrField(data map[string]interface{}, key string) string {
	v, ok := data[key]
	if !ok || v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

func RequiredStrField(data map[string]interface{}, key string) (string, error) {
	v := StrField(data, key)
	if v == "" {
		return "", fmt.Errorf("missing %s", key)
	}
	return v, nil
}

func Int64Field(data map[string]interface{}, key string) (int64, error) {
	v, ok := data[key]
	if !ok || v == nil {
		return 0, fmt.Errorf("missing %s", key)
	}
	switch n := v.(type) {
	case float64:
		return int64(n), nil
	case int64:
		return n, nil
	case int:
		return int64(n), nil
	case json.Number:
		return n.Int64()
	case string:
		return strconv.ParseInt(n, 10, 64)
	default:
		return strconv.ParseInt(fmt.Sprint(v), 10, 64)
	}
}
