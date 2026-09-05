package infrastructure

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const oauthStateTTL = 30 * time.Minute

// OAuthBrowserState is the signed OAuth `state` echoed by GitHub/GitLab.
// Carrying context in state avoids relying on gitoauth_sessionid surviving
// cross-subdomain start→callback (e.g. api.* start vs gitoauth.api.* callback).
type OAuthBrowserState struct {
	V               int    `json:"v"`
	CSRF            string `json:"csrf"`
	UID             string `json:"uid"`
	Next            string `json:"next,omitempty"`
	RK              string `json:"rk,omitempty"`
	Feb             string `json:"feb,omitempty"`
	RedirectURI     string `json:"redirect_uri"`
	ServiceProvider string `json:"sp,omitempty"`
	AllowedHost     string `json:"ah,omitempty"`
	GrantKind       string `json:"gk,omitempty"`
	GrantID         string `json:"gid,omitempty"`
	RepoURL         string `json:"ru,omitempty"`
	Exp             int64  `json:"exp"`
}

// EncodeOAuthState returns a URL-safe signed state token.
func (s *SessionStore) EncodeOAuthState(st OAuthBrowserState) (string, error) {
	if s == nil {
		return "", fmt.Errorf("nil session store")
	}
	st.V = 1
	if st.CSRF == "" || st.UID == "" || st.RedirectURI == "" {
		return "", fmt.Errorf("oauth state missing csrf/uid/redirect_uri")
	}
	if st.Exp <= 0 {
		st.Exp = time.Now().Add(oauthStateTTL).Unix()
	}
	body, err := json.Marshal(st)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, s.secret)
	mac.Write(body)
	token := append(append([]byte{}, body...), mac.Sum(nil)...)
	return "v1." + base64.RawURLEncoding.EncodeToString(token), nil
}

// DecodeOAuthState verifies and parses a signed state token.
// Legacy plain CSRF values (no v1. prefix) return false without error.
func (s *SessionStore) DecodeOAuthState(raw string) (*OAuthBrowserState, bool, error) {
	if s == nil {
		return nil, false, fmt.Errorf("nil session store")
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, false, fmt.Errorf("empty state")
	}
	if !strings.HasPrefix(raw, "v1.") {
		return nil, false, nil
	}
	enc := strings.TrimPrefix(raw, "v1.")
	blob, err := base64.RawURLEncoding.DecodeString(enc)
	if err != nil {
		blob, err = base64.URLEncoding.DecodeString(enc)
		if err != nil {
			return nil, true, fmt.Errorf("bad state encoding")
		}
	}
	if len(blob) < 32 {
		return nil, true, fmt.Errorf("bad state length")
	}
	body, sig := blob[:len(blob)-32], blob[len(blob)-32:]
	mac := hmac.New(sha256.New, s.secret)
	mac.Write(body)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return nil, true, fmt.Errorf("bad state signature")
	}
	var st OAuthBrowserState
	if err := json.Unmarshal(body, &st); err != nil {
		return nil, true, fmt.Errorf("bad state json")
	}
	if st.V != 1 || st.CSRF == "" || st.UID == "" || st.RedirectURI == "" {
		return nil, true, fmt.Errorf("incomplete state")
	}
	if st.Exp <= 0 || time.Now().Unix() >= st.Exp {
		return nil, true, fmt.Errorf("state expired")
	}
	return &st, true, nil
}
