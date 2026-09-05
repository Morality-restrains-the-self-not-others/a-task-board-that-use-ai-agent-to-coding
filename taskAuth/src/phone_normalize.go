package main

import (
	"regexp"
	"strings"
)

var (
	cnMobileRE = regexp.MustCompile(`^1\d{10}$`)
	e164FullRE = regexp.MustCompile(`^\+[1-9]\d{7,14}$`)
)

// canonicalPhoneForSMSAndLogin mirrors Django accounts.phone_normalize.canonical_phone_for_sms_and_login.
// CN → 11-digit national; other → E.164 with +.
func canonicalPhoneForSMSAndLogin(phone string) string {
	p := strings.ReplaceAll(strings.TrimSpace(phone), " ", "")
	if p == "" {
		return ""
	}
	if !strings.HasPrefix(p, "+") {
		if cnMobileRE.MatchString(p) {
			return p
		}
		if matched, _ := regexp.MatchString(`^861\d{10}$`, p); matched {
			return p[2:]
		}
		if strings.HasPrefix(p, "86") && len(p) == 13 && phoneIsAllDigits(p) {
			return p[2:]
		}
		p = "+" + p
	}
	if !e164FullRE.MatchString(p) {
		return ""
	}
	if strings.HasPrefix(p, "+86") {
		national := p[3:]
		if cnMobileRE.MatchString(national) {
			return national
		}
		return ""
	}
	return p
}

func splitCountryCallingCodeAndNational(canonical string) (cc, national string) {
	if canonical == "" {
		return "", ""
	}
	if !strings.HasPrefix(canonical, "+") {
		if cnMobileRE.MatchString(canonical) {
			return "+86", canonical
		}
		return "", ""
	}
	// Minimal split: +86 / +852 / +1 etc. Prefer longest known CC prefix for common cases.
	rest := canonical[1:]
	for _, digs := range []int{3, 2, 1} {
		if len(rest) <= digs {
			continue
		}
		cc = "+" + rest[:digs]
		national = rest[digs:]
		if national == "" {
			continue
		}
		if cc == "+86" && !cnMobileRE.MatchString(national) {
			continue
		}
		if phoneIsAllDigits(national) {
			return cc, national
		}
	}
	return "", ""
}

func composeE164(cc, national string) string {
	cc = strings.TrimSpace(cc)
	national = strings.TrimSpace(national)
	if cc == "" || national == "" {
		return ""
	}
	if !strings.HasPrefix(cc, "+") {
		cc = "+" + cc
	}
	return cc + national
}

func phoneIsAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
