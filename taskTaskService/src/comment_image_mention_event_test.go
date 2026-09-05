package main

import "testing"

func TestImageMentionHasUnfinishedPredecessors(t *testing.T) {
	tests := []struct {
		mode  string
		nPrev int
		deps  []string
		want  bool
	}{
		{executionModeIndependent, 2, []string{"c1"}, false},
		{executionModeIndependent, 0, nil, false},
		{executionModeWaitPrevious, 0, nil, false},
		{executionModeWaitPrevious, 1, nil, true},
		{executionModeWaitPrevious, 0, []string{"cmt_prev"}, true},
	}
	for _, tc := range tests {
		got := imageMentionHasUnfinishedPredecessors(tc.mode, tc.nPrev, tc.deps)
		if got != tc.want {
			t.Fatalf("mode=%s n=%d deps=%v got=%v want=%v", tc.mode, tc.nPrev, tc.deps, got, tc.want)
		}
	}
}

func TestBuildTaskCommentImageMentionedDataIncludesMode(t *testing.T) {
	setupTestDB(t)
	startMockProjectServiceWithAtMode(t, false)
	taskID := createTestTaskForComments(t)
	first := postComment(t, taskID, `{"content":"first hello"}`)
	if first.Code != 201 {
		t.Fatalf("first comment: %d %s", first.Code, first.Body.String())
	}

	data := buildTaskCommentImageMentionedData(
		taskID, "t1", "ws1", "cmt_java", "img-1", "trae-agent",
		"@trae-agent java hello", "u1", executionModeWaitPrevious, nil,
	)
	if data["execution_mode"] != executionModeWaitPrevious {
		t.Fatalf("execution_mode=%v", data["execution_mode"])
	}
	if data["has_unfinished_predecessors"] != true {
		t.Fatalf("serial follow-up must report unfinished predecessors, got %#v", data["has_unfinished_predecessors"])
	}

	indep := buildTaskCommentImageMentionedData(
		taskID, "t1", "ws1", "cmt_lisp", "img-1", "trae-agent",
		"@trae-agent lisp hello", "u1", executionModeIndependent, nil,
	)
	if indep["execution_mode"] != executionModeIndependent {
		t.Fatalf("independent execution_mode=%v", indep["execution_mode"])
	}
	if indep["has_unfinished_predecessors"] != false {
		t.Fatalf("independent must not wait, got %#v", indep["has_unfinished_predecessors"])
	}
}
