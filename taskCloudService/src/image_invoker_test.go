package main

import (
	"net/http"
	"testing"
)

func TestResolveImageInvokerUserIDPrefersBody(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Auth-User-Id", "auth-user")
	got := resolveImageInvokerUserID(req, map[string]interface{}{
		"image_invoker_user_id": "comment-author",
	})
	if got != "comment-author" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveImageInvokerUserIDFallsBackToAuth(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Auth-User-Id", "auth-user")
	got := resolveImageInvokerUserID(req, map[string]interface{}{})
	if got != "auth-user" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveImageInvokerUserIDCommentCreatedBy(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	got := resolveImageInvokerUserID(req, map[string]interface{}{
		"comment_created_by_id": "from-comment",
	})
	if got != "from-comment" {
		t.Fatalf("got %q", got)
	}
}
