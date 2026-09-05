package main

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	legalNameMinRunes = 2
	legalNameMaxRunes = 32
)

func normalizeLegalName(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	n := utf8.RuneCountInString(s)
	if n == 0 {
		return "", errReferralLegalNameRequired
	}
	if n < legalNameMinRunes {
		return "", errReferralLegalNameTooShort
	}
	if n > legalNameMaxRunes {
		return "", errReferralLegalNameTooLong
	}
	for _, r := range s {
		if unicode.Is(unicode.Han, r) || unicode.IsLetter(r) || r == '·' || r == '•' || r == ' ' {
			continue
		}
		return "", errReferralLegalNameInvalid
	}
	return s, nil
}

func lookupLegalName(userID string) string {
	if db == nil || strings.TrimSpace(userID) == "" {
		return ""
	}
	var name string
	err := db.QueryRow(
		`SELECT legal_name FROM referral_code WHERE user_id = ? ORDER BY id DESC LIMIT 1`,
		userID,
	).Scan(&name)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(name)
}
