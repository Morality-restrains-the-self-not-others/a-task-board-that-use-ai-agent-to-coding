package main

import (
	"testing"
	"time"
)

func TestParseCloudUTCDateTimeTreatsWallClockAsUTC(t *testing.T) {
	want := time.Date(2026, 8, 13, 15, 30, 59, 0, time.UTC)
	cases := []string{
		"2026-08-13 15:30:59",
		"2026-08-13T15:30:59Z",
		"2026-08-13T15:30:59+08:00",
		"2026-08-13T15:30:59.662783363+08:00",
	}
	for _, in := range cases {
		got := parseCloudUTCDateTime(in)
		if !got.Equal(want) {
			t.Fatalf("parseCloudUTCDateTime(%q)=%s want %s", in, got.UTC().Format(time.RFC3339Nano), want.Format(time.RFC3339))
		}
	}
}

func TestFormatCloudUTCJSON(t *testing.T) {
	in := time.Date(2026, 8, 13, 15, 30, 59, 0, time.UTC)
	got := formatCloudUTCJSON(in)
	if got != "2026-08-13T15:30:59Z" {
		t.Fatalf("formatCloudUTCJSON=%q", got)
	}
	if formatCloudUTCJSON(time.Time{}) != "" {
		t.Fatalf("zero time should serialize empty")
	}
}
