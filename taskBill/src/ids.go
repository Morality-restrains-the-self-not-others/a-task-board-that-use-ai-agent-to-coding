package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// formatID 将 DB int64 转为 API 字符串（Snowflake 禁止以 JSON number 出站）。
func formatID(id int64) string {
	return strconv.FormatInt(id, 10)
}

// parseIDField 解析请求体中的 ID；仅接受 string / json.Number，拒绝 float64 防精度丢失。
func parseIDField(v interface{}) (int64, error) {
	switch t := v.(type) {
	case nil:
		return 0, fmt.Errorf("missing id field")
	case string:
		return strconv.ParseInt(strings.TrimSpace(t), 10, 64)
	case json.Number:
		return t.Int64()
	case float64:
		return 0, fmt.Errorf("id must be string, not number (json float loses precision)")
	case int64:
		return t, nil
	case int:
		return int64(t), nil
	default:
		return strconv.ParseInt(fmt.Sprintf("%v", t), 10, 64)
	}
}

// parseInt64Field 解析非 ID 的整数字段（如 points_delta），允许 json.Number。
func parseInt64Field(v interface{}) (int64, error) {
	switch t := v.(type) {
	case nil:
		return 0, fmt.Errorf("missing field")
	case string:
		return strconv.ParseInt(strings.TrimSpace(t), 10, 64)
	case json.Number:
		return t.Int64()
	case float64:
		return int64(t), nil
	case int64:
		return t, nil
	case int:
		return int64(t), nil
	default:
		return strconv.ParseInt(fmt.Sprintf("%v", t), 10, 64)
	}
}

func parseFloat64Field(v interface{}) (float64, error) {
	switch t := v.(type) {
	case nil:
		return 0, fmt.Errorf("missing field")
	case string:
		return strconv.ParseFloat(strings.TrimSpace(t), 64)
	case json.Number:
		return t.Float64()
	case float64:
		return t, nil
	case int64:
		return float64(t), nil
	case int:
		return float64(t), nil
	default:
		return strconv.ParseFloat(fmt.Sprintf("%v", t), 64)
	}
}

func boolField(body map[string]interface{}, key string) bool {
	v, ok := body[key]
	if !ok || v == nil {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		s := strings.TrimSpace(strings.ToLower(t))
		return s == "true" || s == "1" || s == "yes"
	case json.Number:
		n, _ := t.Int64()
		return n != 0
	case float64:
		return t != 0
	default:
		return false
	}
}

// stringField 读取已是字符串的业务 ID（task_id、user_id 等）。
func stringField(body map[string]interface{}, key string) string {
	v, ok := body[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	default:
		return fmt.Sprintf("%v", t)
	}
}
