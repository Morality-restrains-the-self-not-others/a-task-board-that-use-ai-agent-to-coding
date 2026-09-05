package interfaces

import "testing"

func TestParseContainerAPIPathRejectsWithoutComment(t *testing.T) {
	cases := []string{
		"/api/tenant/t1/workspace/w1/task/task_1/cloud/server-container-token/task-detail/",
		"/api/tenant/t1/workspace/w1/task/task_1/comment/-/cloud/server-container-token/task-detail/",
		"/api/tenant/t1/workspace/w1/task/task_1/comment//cloud/server-container-token/task-detail/",
	}
	for _, path := range cases {
		if _, _, _, _, _, ok := parseContainerAPIPath(path); ok {
			t.Fatalf("expected reject for %q", path)
		}
	}
}

func TestParseContainerAPIPathWithCommentSegment(t *testing.T) {
	tid, wid, tk, cid, action, ok := parseContainerAPIPath(
		"/api/tenant/t1/workspace/w1/task/task_1/comment/cmt_1/cloud/server-container-token/exchange-refresh/",
	)
	if !ok {
		t.Fatal("expected comment-scoped path to match")
	}
	if tid != "t1" || wid != "w1" || tk != "task_1" || cid != "cmt_1" || action != "exchange-refresh" {
		t.Fatalf("got tenant=%s workspace=%s task=%s comment=%s action=%s", tid, wid, tk, cid, action)
	}
}

func TestParseContainerAPIPathRejectsMissingCloud(t *testing.T) {
	_, _, _, _, _, ok := parseContainerAPIPath(
		"/api/tenant/t1/workspace/w1/task/task_1/comment/cmt_1/server-container-token/task-detail/",
	)
	if ok {
		t.Fatal("path without cloud segment must not match")
	}
}

func TestCommentScopeMismatch(t *testing.T) {
	if commentScopeMismatch("cmt_1", "cmt_1") {
		t.Fatal("same comment must match")
	}
	if !commentScopeMismatch("cmt_1", "cmt_2") {
		t.Fatal("different comments must mismatch")
	}
	if commentScopeMismatch("", "cmt_1") || commentScopeMismatch("-", "cmt_1") {
		t.Fatal("empty path comment is advisory")
	}
	if commentScopeMismatch("cmt_1", "") {
		t.Fatal("empty token comment is advisory")
	}
}
