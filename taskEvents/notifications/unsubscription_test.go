package notifications

import "testing"

func TestListUnsubscribeHeaders(t *testing.T) {
	if listUnsubscribeHeaders("") != nil {
		t.Fatal("empty url should yield nil headers")
	}
	h := listUnsubscribeHeaders("https://example.test/api/public/email-unsubscribe/?token=abc")
	if h["List-Unsubscribe"] != "<https://example.test/api/public/email-unsubscribe/?token=abc>" {
		t.Fatalf("List-Unsubscribe=%q", h["List-Unsubscribe"])
	}
	if h["List-Unsubscribe-Post"] != "List-Unsubscribe=One-Click" {
		t.Fatalf("List-Unsubscribe-Post=%q", h["List-Unsubscribe-Post"])
	}
}
