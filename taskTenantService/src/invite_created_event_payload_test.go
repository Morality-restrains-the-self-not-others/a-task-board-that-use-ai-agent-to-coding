package main

import (
	"testing"
)

// OPT-20260829-011: INVITATION_CREATED must not carry the plaintext
// invitation_token. The open-link token is a join credential shared by many
// users; Kafka/UI logs widen its leak surface. invitation_url (used by the
// email template) still carries it for delivery.
func TestInviteCreatedEventOmitsInvitationToken(t *testing.T) {
	mux := setupTestService(t)
	var records []memberJoinedRecord
	publishEventHook = func(eventType string, data map[string]interface{}) {
		records = append(records, memberJoinedRecord{eventType: eventType, data: data})
	}
	t.Cleanup(func() { publishEventHook = nil })

	createInviteAndGetToken(t, mux, `{"invite_method":"link","company_member_name":"One","workspace_id":"ws1"}`)

	var created map[string]interface{}
	found := false
	for _, r := range records {
		if r.eventType == "INVITATION_CREATED" {
			created = r.data
			found = true
		}
	}
	if !found {
		t.Fatal("no INVITATION_CREATED event recorded")
	}
	if _, has := created["invitation_token"]; has {
		t.Fatalf("INVITATION_CREATED payload must not carry invitation_token: %v", created)
	}
	if url, _ := created["invitation_url"].(string); url == "" {
		t.Fatalf("invitation_url missing from payload: %v", created)
	}
	if id, _ := created["invitation_id"].(string); id == "" {
		t.Fatalf("invitation_id missing from payload: %v", created)
	}
}
