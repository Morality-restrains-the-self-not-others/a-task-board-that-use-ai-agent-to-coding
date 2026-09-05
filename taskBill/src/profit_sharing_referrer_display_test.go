package main

import (
	"testing"
	"time"
)

func TestReferrerProfitSharingDisplayWindows(t *testing.T) {
	now := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	paid := func(daysAgo int) string {
		return now.AddDate(0, 0, -daysAgo).Format(time.RFC3339)
	}
	cases := []struct {
		name    string
		status  string
		paidAt  string
		display string
		share   bool
	}{
		{"frozen 14d", psStatusPending, paid(14), referrerPSFrozen, false},
		{"shareable 15d", psStatusPending, paid(15), referrerPSShareable, true},
		{"shareable failed 20d", psStatusFailed, paid(20), referrerPSShareable, true},
		{"failed 5d not frozen", psStatusFailed, paid(5), referrerPSFailed, false},
		{"failed missing paid_at", psStatusFailed, "", referrerPSFailed, false},
		{"expired failed 30d", psStatusFailed, paid(30), referrerPSExpired, false},
		{"expired 30d", psStatusPending, paid(30), referrerPSExpired, false},
		{"shared ignores age", psStatusFinished, paid(3), referrerPSShared, false},
		{"processing", psStatusProcessing, paid(20), referrerPSProcessing, false},
		{"missing paid_at", psStatusPending, "", referrerPSFrozen, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, share := referrerProfitSharingDisplay(tc.status, tc.paidAt, now)
			if got != tc.display || share != tc.share {
				t.Fatalf("got %s share=%v want %s share=%v", got, share, tc.display, tc.share)
			}
		})
	}
}
