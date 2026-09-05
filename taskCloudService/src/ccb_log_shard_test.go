package main

import (
	"regexp"
	"testing"
)

func TestCCBLogShardIndexFrozenVectors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		ws   string
		want uint32
	}{
		{"ws1", 8},
		{"ws2", 2},
		{"ws_-2309487803472456748", 6},
	}
	for _, tc := range cases {
		got, err := ccbLogShardIndex(tc.ws)
		if err != nil {
			t.Fatalf("workspace %q: %v", tc.ws, err)
		}
		if got != tc.want {
			t.Fatalf("workspace %q shard=%d want %d", tc.ws, got, tc.want)
		}
	}
}

func TestCCBLogShardIndexEmptyRejected(t *testing.T) {
	t.Parallel()
	if _, err := ccbLogShardIndex(""); err == nil {
		t.Fatal("empty workspace must be rejected")
	}
	if _, err := ccbLogShardIndex("  \t"); err == nil {
		t.Fatal("whitespace workspace must be rejected")
	}
}

func TestCCBLogTableNameFormatAndStability(t *testing.T) {
	t.Parallel()
	re := regexp.MustCompile(`^cloud_comment_container_binding_logs_[0-9]{2}$`)
	a, err := ccbLogTable("ws1")
	if err != nil {
		t.Fatal(err)
	}
	b, err := ccbLogTable("ws1")
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("unstable table name %q vs %q", a, b)
	}
	if !re.MatchString(a) {
		t.Fatalf("table name %q does not match shard pattern", a)
	}
	if a != "cloud_comment_container_binding_logs_08" {
		t.Fatalf("ws1 table=%s want ..._08", a)
	}
	other, err := ccbLogTable("ws2")
	if err != nil {
		t.Fatal(err)
	}
	if other == a {
		t.Fatalf("ws1 and ws2 unexpectedly share table %s", a)
	}
}

func TestCCBLogTableRejectsUnsanitizedInterpolation(t *testing.T) {
	t.Parallel()
	name, err := ccbLogTable(`ws1; DROP TABLE x`)
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^cloud_comment_container_binding_logs_[0-9]{2}$`).MatchString(name) {
		t.Fatalf("hostile workspace leaked into identifier: %q", name)
	}
}
