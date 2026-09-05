package main

import (
	"context"
	"testing"
)

// OPT-20260901-028: grant_ticket 路径的 COMMENT_GIT_OAUTH_GRANTED 事件必须与 seed 路径
// 同形 —— 幂等键为 grant:comment:{commentID}:{userID}:{gitsite}，payload 携带 comment_id，
// 否则审计/幂等消费无法按评论粒度对账（ticket id 键无法与 seed 事件匹配）。
func TestApplyGrantTicketToIdentities_EmitsSeedStyleIdempotencyKey(t *testing.T) {
	prevConsume := consumeGitOAuthGrantTicketFn
	consumeGitOAuthGrantTicketFn = func(userID, gitsite, ticketID string) (string, bool) {
		if userID != "u1" || gitsite != "github.com" || ticketID != "tkt-1" {
			t.Fatalf("consume user=%s site=%s ticket=%s", userID, gitsite, ticketID)
		}
		return "gh-remote-1", true
	}
	t.Cleanup(func() { consumeGitOAuthGrantTicketFn = prevConsume })

	type captured struct {
		typ  string
		data map[string]interface{}
		key  string
	}
	var got []captured
	prevPub := publishDomainEventFn
	publishDomainEventFn = func(_ context.Context, eventType string, data map[string]interface{}, key string) error {
		got = append(got, captured{eventType, data, key})
		return nil
	}
	t.Cleanup(func() { publishDomainEventFn = prevPub })

	selections := []RepoIdentitySelection{{
		RepoURL:       "https://github.com/acme/demo.git",
		GitIdentityID: "gid-1",
	}}
	out := applyGrantTicketToIdentities("cmt-1", "u1", "tkt-1", selections)
	if len(out) != 1 || out[0].OauthRemoteUserID != "gh-remote-1" {
		t.Fatalf("stamped=%+v", out)
	}
	if len(got) != 1 {
		t.Fatalf("events=%+v", got)
	}
	if got[0].typ != "COMMENT_GIT_OAUTH_GRANTED" {
		t.Fatalf("event type=%q", got[0].typ)
	}
	wantKey := "grant:comment:cmt-1:u1:github.com"
	if got[0].key != wantKey {
		t.Fatalf("idempotency key=%q want %q", got[0].key, wantKey)
	}
	if got[0].data["comment_id"] != "cmt-1" {
		t.Fatalf("payload comment_id=%v", got[0].data["comment_id"])
	}
	if got[0].data["via"] != "grant_ticket" {
		t.Fatalf("payload via=%v", got[0].data["via"])
	}
	if got[0].data["remote_user_id"] != "gh-remote-1" {
		t.Fatalf("payload remote_user_id=%v", got[0].data["remote_user_id"])
	}
}

func TestApplyGrantTicketToIdentities_EmptyTicketReturnsUnchanged(t *testing.T) {
	selections := []RepoIdentitySelection{{
		RepoURL:       "https://github.com/acme/demo.git",
		GitIdentityID: "gid-1",
	}}
	out := applyGrantTicketToIdentities("cmt-1", "u1", "  ", selections)
	if len(out) != 1 || out[0].OauthRemoteUserID != "" {
		t.Fatalf("empty ticket must not stamp: %+v", out)
	}
}
