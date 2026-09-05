package domain

import (
	"testing"
)

func TestMergeStartupLogEntryIdempotentByID(t *testing.T) {
	id := StartupLogIDs{WorkspaceID: "ws", TaskID: "task", CommentID: "cmt"}
	b := EmptyStartupLogBundle(id)
	e := StartupLogEntry{ID: "L1", Message: "a", CreatedAt: "2026-08-27 01:00:00"}
	b = MergeStartupLogEntry(b, id, e)
	b = MergeStartupLogEntry(b, id, StartupLogEntry{ID: "L1", Message: "a", CreatedAt: "2026-08-27 01:00:00"})
	if len(b.Logs) != 1 {
		t.Fatalf("logs=%d want 1", len(b.Logs))
	}
	raw, err := MarshalStartupLogBundle(b)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseStartupLogBundle(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Logs) != 1 || parsed.Logs[0].ID != "L1" {
		t.Fatalf("parsed=%+v", parsed)
	}
}

func TestMergeStartupLogEntrySortsByCreatedAt(t *testing.T) {
	id := StartupLogIDs{WorkspaceID: "ws", TaskID: "task", CommentID: "cmt"}
	b := EmptyStartupLogBundle(id)
	b = MergeStartupLogEntry(b, id, StartupLogEntry{ID: "L2", CreatedAt: "2026-08-27 02:00:00"})
	b = MergeStartupLogEntry(b, id, StartupLogEntry{ID: "L1", CreatedAt: "2026-08-27 01:00:00"})
	if b.Logs[0].ID != "L1" || b.Logs[1].ID != "L2" {
		t.Fatalf("order=%+v", b.Logs)
	}
}
