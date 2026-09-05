package main

import (
	"strings"
	"time"
)

const (
	profitSharingFreezeDays = 15
	profitSharingExpireDays = 30

	referrerPSFrozen     = "frozen"
	referrerPSShareable  = "shareable"
	referrerPSExpired    = "expired"
	referrerPSShared     = "shared"
	referrerPSProcessing = "processing"
	referrerPSFailed     = "failed"
)

func parseFlexibleTime(raw string) (time.Time, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return time.Time{}, false
	}
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.UTC); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}

func referrerProfitSharingDisplay(rawStatus, paidAt string, now time.Time) (display string, canShare bool) {
	status := strings.TrimSpace(rawStatus)
	if status == psStatusFinished {
		return referrerPSShared, false
	}
	if status == psStatusProcessing {
		return referrerPSProcessing, false
	}

	paid, ok := parseFlexibleTime(paidAt)
	inShareWindow := false
	expired := false
	if ok {
		age := now.UTC().Sub(paid)
		if age >= time.Duration(profitSharingExpireDays)*24*time.Hour {
			expired = true
		} else if age >= time.Duration(profitSharingFreezeDays)*24*time.Hour {
			inShareWindow = true
		}
	}

	if status == psStatusFailed {
		if expired {
			return referrerPSExpired, false
		}
		if inShareWindow {
			return referrerPSShareable, true
		}
		return referrerPSFailed, false
	}

	if !ok {
		return referrerPSFrozen, false
	}
	if expired {
		return referrerPSExpired, false
	}
	if inShareWindow && status == psStatusPending {
		return referrerPSShareable, true
	}
	return referrerPSFrozen, false
}
