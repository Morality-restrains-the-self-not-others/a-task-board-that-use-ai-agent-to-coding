package domain

import (
	"errors"
	"net/url"
	"strings"
)

// ErrInvalidLokiURL indicates the Loki base URL could not be parsed.
var ErrInvalidLokiURL = errors.New("invalid loki url")

// LokiPushURL returns the Loki push API URL for Promtail clients from a base Loki HTTP URL.
func LokiPushURL(lokiBase string) (string, error) {
	base := strings.TrimSpace(lokiBase)
	if base == "" {
		return "", ErrInvalidLokiURL
	}
	u, err := url.Parse(base)
	if err != nil {
		return "", ErrInvalidLokiURL
	}
	if u.Scheme == "" {
		u, err = url.Parse("http://" + base)
		if err != nil {
			return "", ErrInvalidLokiURL
		}
	}
	u.Path = "/loki/api/v1/push"
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}
