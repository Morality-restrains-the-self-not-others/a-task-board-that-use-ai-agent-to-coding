package aliyun

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	aliyunAutoReleaseMinOffsetMinutes = 30
	aliyunAutoReleaseMaxOffsetDays    = 3 * 365
)

// ComputeAutoReleaseTimeISO mirrors Python _compute_auto_release_time_iso.
func ComputeAutoReleaseTimeISO(hardware map[string]interface{}) string {
	if hardware == nil {
		return ""
	}
	minutes := interfaceToFloatPtr(hardware["auto_release_minutes"])
	if minutes == nil {
		if legacy := interfaceToFloatPtr(hardware["auto_release_hours"]); legacy != nil {
			v := *legacy * 60
			minutes = &v
		}
	}
	if minutes == nil || *minutes <= 0 {
		return ""
	}
	now := time.Now().UTC()
	releaseAt := ceilToUTCMinute(now.Add(time.Duration(*minutes) * time.Minute))
	earliest := ceilToUTCMinute(now.Add(aliyunAutoReleaseMinOffsetMinutes * time.Minute))
	if releaseAt.Before(earliest) {
		releaseAt = earliest
	}
	latest := now.Add(aliyunAutoReleaseMaxOffsetDays * 24 * time.Hour)
	if releaseAt.After(latest) {
		releaseAt = latest.Truncate(time.Minute)
	}
	return releaseAt.Format("2006-01-02T15:04:05Z")
}

func ceilToUTCMinute(t time.Time) time.Time {
	t = t.UTC()
	if t.Second() == 0 && t.Nanosecond() == 0 {
		return t
	}
	return t.Truncate(time.Minute).Add(time.Minute)
}

func interfaceToFloatPtr(v interface{}) *float64 {
	if v == nil {
		return nil
	}
	switch n := v.(type) {
	case float64:
		return &n
	case float32:
		f := float64(n)
		return &f
	case int:
		f := float64(n)
		return &f
	case int64:
		f := float64(n)
		return &f
	default:
		s := strings.TrimSpace(fmt.Sprint(v))
		if s == "" {
			return nil
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil
		}
		return &f
	}
}
