package domain

import "testing"

func TestGitIdentitySnapshotFromCommentZeroValue(t *testing.T) {
	var s GitIdentitySnapshot
	if s.FromComment {
		t.Fatal("FromComment must default false for task-table bindings")
	}
	s.FromComment = true
	s.GitIdentityID = "gid-comment"
	if !s.FromComment || s.GitIdentityID != "gid-comment" {
		t.Fatalf("comment-selected snapshot=%+v", s)
	}
}
