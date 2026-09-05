package main

import (
	"regexp"
	"time"
)

var cloudDateTimePrefix = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})[ T](\d{2}:\d{2}:\d{2})`)

const mysqlUTCDateTimeLayout = "2006-01-02 15:04:05"

// parseCloudUTCDateTime interprets timestamps this service persists as UTC wall
// clock in MySQL DATETIME (no zone). The driver may return a naive string or
// RFC3339 with a local offset attached to the same digits; always take YYYY-MM-DD
// HH:MM:SS as UTC so list JSON matches SSE browser-local display after conversion.
func parseCloudUTCDateTime(s string) time.Time {
	s = trim(s)
	if s == "" {
		return time.Time{}
	}
	m := cloudDateTimePrefix.FindStringSubmatch(s)
	if m == nil {
		return time.Time{}
	}
	t, err := time.ParseInLocation(mysqlUTCDateTimeLayout, m[1]+" "+m[2], time.UTC)
	if err != nil {
		return time.Time{}
	}
	return t
}

func formatMySQLUTCDateTime(t time.Time) string {
	return t.UTC().Format(mysqlUTCDateTimeLayout)
}

func formatCloudUTCJSON(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
