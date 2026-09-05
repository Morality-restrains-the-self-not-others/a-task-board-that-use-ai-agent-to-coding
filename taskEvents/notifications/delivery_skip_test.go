package notifications

import (
	"context"
	"encoding/json"
	"testing"

	"taskEvents/domain"
	"taskEvents/notifications/cfg"
)

func closedSMTPDelivery(t *testing.T) *LocalDelivery {
	t.Helper()
	d, err := NewLocalDelivery(cfg.Settings{
		Email: cfg.EmailSettings{
			Host:          "127.0.0.1",
			Port:          1,
			DefaultFrom:   "from@example.com",
			UseSSL:        false,
			LocalHostname: "localhost",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestInvitationCreatedSkipsWhenEmailSkipped(t *testing.T) {
	orig := lookupInviteUnsubscription
	t.Cleanup(func() { lookupInviteUnsubscription = orig })
	lookupInviteUnsubscription = func(string) (bool, string) { return false, "" }

	data, _ := json.Marshal(map[string]interface{}{
		"company_name":    "Acme",
		"email":           "optout@example.com",
		"invitation_url":  "https://example.test/join?token=x",
		"invite_method":   "email",
		"email_skipped":   true,
		"expiration_days": 7,
	})
	out, err := closedSMTPDelivery(t).Dispatch(context.Background(), domain.DomainCommand{
		EventType: "INVITATION_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "INVITATION_CREATED", Data: data, Key: "k-skip"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
}

func TestInvitationCreatedSkipsWhenLookupUnsubscribed(t *testing.T) {
	orig := lookupInviteUnsubscription
	t.Cleanup(func() { lookupInviteUnsubscription = orig })
	lookupInviteUnsubscription = func(email string) (bool, string) {
		if email != "blocked@example.com" {
			t.Fatalf("email=%s", email)
		}
		return true, "https://example.test/unsub"
	}

	data, _ := json.Marshal(map[string]interface{}{
		"company_name":   "Acme",
		"email":          "blocked@example.com",
		"invitation_url": "https://example.test/join?token=y",
		"invite_method":  "email",
	})
	out, err := closedSMTPDelivery(t).Dispatch(context.Background(), domain.DomainCommand{
		EventType: "INVITATION_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "INVITATION_CREATED", Data: data, Key: "k-lookup"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
}

func TestEmailSentSkipsWhenPreDelivered(t *testing.T) {
	data, _ := json.Marshal(map[string]interface{}{
		"subject":        "您的验证码",
		"message":        "code 123456",
		"html_message":   "<p>123456</p>",
		"from_email":     "from@example.com",
		"recipient_list": []string{"otp@example.com"},
		"template_name":  "verification_code",
		"pre_delivered":  true,
	})
	out, err := closedSMTPDelivery(t).Dispatch(context.Background(), domain.DomainCommand{
		EventType: "EMAIL_SENT",
		Envelope:  domain.EventEnvelope{EventType: "EMAIL_SENT", Data: data, Key: "k-pre-delivered"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
}

func TestEmailSentSkipsWhenUnsubscribed(t *testing.T) {
	orig := lookupInviteUnsubscription
	t.Cleanup(func() { lookupInviteUnsubscription = orig })
	lookupInviteUnsubscription = func(string) (bool, string) { return true, "https://example.test/unsub" }

	data, _ := json.Marshal(map[string]interface{}{
		"subject":        "账号邀请",
		"message":        "plain",
		"html_message":   "<p>html</p>",
		"from_email":     "from@example.com",
		"recipient_list": []string{"skip@example.com"},
		"template_name":  "email_registration_invite",
	})
	out, err := closedSMTPDelivery(t).Dispatch(context.Background(), domain.DomainCommand{
		EventType: "EMAIL_SENT",
		Envelope:  domain.EventEnvelope{EventType: "EMAIL_SENT", Data: data, Key: "k-email-sent"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
}
