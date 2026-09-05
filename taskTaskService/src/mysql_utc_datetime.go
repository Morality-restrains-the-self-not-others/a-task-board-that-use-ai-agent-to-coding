package main

import (
	"regexp"
	"strings"
	"time"
)

var mysqlDateTimePrefix = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})[ T](\d{2}:\d{2}:\d{2})`)

const mysqlUTCDateTimeLayout = "2006-01-02 15:04:05"

// formatMySQLUTCDateTime writes naive DATETIME digits in UTC so loc=Local DSN
// cannot convert the wall clock back to Asia/Shanghai before INSERT.
func formatMySQLUTCDateTime(t time.Time) string {
	return t.UTC().Format(mysqlUTCDateTimeLayout)
}

// parseMySQLUTCDateTime treats persisted DATETIME wall clock as UTC, including
// driver strings that attach a local offset to the same digits.
func parseMySQLUTCDateTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	m := mysqlDateTimePrefix.FindStringSubmatch(s)
	if m == nil {
		if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
			return t.UTC()
		}
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return t.UTC()
		}
		return time.Time{}
	}
	t, err := time.ParseInLocation(mysqlUTCDateTimeLayout, m[1]+" "+m[2], time.UTC)
	if err != nil {
		return time.Time{}
	}
	return t
}
