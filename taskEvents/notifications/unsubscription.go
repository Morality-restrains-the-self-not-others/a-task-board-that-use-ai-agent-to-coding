package notifications

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"tracelog"
)

type unsubLookup struct {
	Unsubscribed   bool   `json:"unsubscribed"`
	UnsubscribeURL string `json:"unsubscribe_url"`
}

var lookupInviteUnsubscription = lookupInviteUnsubscriptionHTTP

func lookupInviteUnsubscriptionHTTP(email string) (unsubscribed bool, unsubscribeURL string) {
	email = strings.TrimSpace(email)
	if email == "" {
		return false, ""
	}
	u := taskAuthInternalURL() + "/api/internal/email-unsubscription/?email=" + url.QueryEscape(email)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		log.Printf("[notifications] unsub lookup build: %v", err)
		return false, ""
	}
	if sec := taskAuthInternalSecret(); sec != "" {
		req.Header.Set("X-TaskAuth-Internal-Secret", sec)
	}
	client := tracelog.DirectClient(5 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[notifications] unsub lookup failed: %v", err)
		return false, ""
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[notifications] unsub lookup status=%d %s", resp.StatusCode, strings.TrimSpace(string(body)))
		return false, ""
	}
	var out unsubLookup
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		log.Printf("[notifications] unsub lookup decode: %v", err)
		return false, ""
	}
	return out.Unsubscribed, out.UnsubscribeURL
}

func taskAuthInternalSecret() string {
	for _, k := range []string{"TASK_AUTH_INTERNAL_SECRET", "SHARED_INTERNAL_SECRET", "INTERNAL_SECRET"} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

func listUnsubscribeHeaders(unsubURL string) map[string]string {
	if strings.TrimSpace(unsubURL) == "" {
		return nil
	}
	return map[string]string{
		"List-Unsubscribe":      "<" + unsubURL + ">",
		"List-Unsubscribe-Post": "List-Unsubscribe=One-Click",
	}
}
