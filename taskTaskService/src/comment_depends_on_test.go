package main

import "testing"

func TestNormalizeDependsOnCommentIDs(t *testing.T) {
	got := normalizeDependsOnCommentIDs([]interface{}{" a ", "b", "a", ""})
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("got=%v", got)
	}
	got = normalizeDependsOnCommentIDs(`["c1","c2"]`)
	if len(got) != 2 || got[0] != "c1" || got[1] != "c2" {
		t.Fatalf("json got=%v", got)
	}
	got = normalizeDependsOnCommentIDs("c1, c2")
	if len(got) != 2 {
		t.Fatalf("csv got=%v", got)
	}
	if dependsOnCommentIDsJSON(nil) != "[]" {
		t.Fatalf("empty json")
	}
}
