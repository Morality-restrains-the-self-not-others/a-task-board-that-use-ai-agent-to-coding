package main

import "fmt"

const (
	defaultInviteExpirationDays = 90
	maxInviteExpirationDays     = 365
)

func normalizeInviteExpirationDays(days int, missingDefaults bool) (int, error) {
	if days <= 0 {
		if missingDefaults {
			return defaultInviteExpirationDays, nil
		}
		return 0, fmt.Errorf("expiration_days 必须大于0")
	}
	if days > maxInviteExpirationDays {
		return 0, fmt.Errorf("expiration_days 不能超过%d天", maxInviteExpirationDays)
	}
	return days, nil
}
