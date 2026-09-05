package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// GitLab Branches API (lib/api/branches.rb) accepts only
// sort=name_asc|updated_asc|updated_desc. order_by=updated_at&sort=desc is 400.
func TestFetchGitLabBranchesUsesUpdatedDescSort(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		sort := r.URL.Query().Get("sort")
		switch sort {
		case "", "name_asc", "updated_asc", "updated_desc":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"name":"main","commit":{"committed_date":"2026-08-11T00:00:00Z"}}]`))
		default:
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"sort does not have a valid value"}`))
		}
	}))
	defer srv.Close()

	branches, errMsg := fetchGitLabBranches(srv.URL+"/group/proj.git", "tok", "", "")
	if errMsg != "" {
		t.Fatalf("err=%q query=%q (GitLab rejects sort=desc / order_by=updated_at)", errMsg, gotQuery)
	}
	if len(branches) != 1 || branches[0] != "main" {
		t.Fatalf("branches=%v", branches)
	}
	if got := queryValue(gotQuery, "sort"); got != "updated_desc" {
		t.Fatalf("sort=%q want updated_desc; query=%q", got, gotQuery)
	}
	if got := queryValue(gotQuery, "order_by"); got != "" {
		t.Fatalf("order_by=%q must not be sent; branches API has no order_by", got)
	}
}

func queryValue(raw, key string) string {
	for _, part := range strings.Split(raw, "&") {
		k, v, ok := strings.Cut(part, "=")
		if ok && k == key {
			return v
		}
	}
	return ""
}

func TestSortBranchNamesByUpdatedAtDesc(t *testing.T) {
	t1 := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 8, 11, 9, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	got := sortBranchNamesByUpdatedAtDesc([]branchNamedTime{
		{Name: "old", UpdatedAt: t3},
		{Name: "mid", UpdatedAt: t1},
		{Name: "new", UpdatedAt: t2},
	})
	want := []string{"new", "mid", "old"}
	if len(got) != len(want) {
		t.Fatalf("len=%d want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestSortBranchNamesByUpdatedAtDescUndatedKeepsOrderAfterDated(t *testing.T) {
	t1 := time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)
	got := sortBranchNamesByUpdatedAtDesc([]branchNamedTime{
		{Name: "undated-a"},
		{Name: "dated", UpdatedAt: t1},
		{Name: "undated-b"},
	})
	want := []string{"dated", "undated-a", "undated-b"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestSortBranchNamesByUpdatedAtDescAllUndatedPreservesInput(t *testing.T) {
	got := sortBranchNamesByUpdatedAtDesc([]branchNamedTime{
		{Name: "main"},
		{Name: "dev"},
		{Name: "feature/x"},
	})
	want := []string{"main", "dev", "feature/x"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestParseFlexibleTime(t *testing.T) {
	if parseFlexibleTime("").IsZero() != true {
		t.Fatal("empty should be zero")
	}
	ts := parseFlexibleTime("2026-08-11T09:30:00.000Z")
	if ts.IsZero() {
		t.Fatal("expected parsed time")
	}
	if ts.Year() != 2026 || ts.Month() != time.August || ts.Day() != 11 {
		t.Fatalf("unexpected %v", ts)
	}
}
