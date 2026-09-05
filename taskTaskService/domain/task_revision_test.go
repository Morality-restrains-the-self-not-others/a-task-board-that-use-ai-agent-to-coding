package domain

import "testing"

func TestDiffVersionedTaskNoChange(t *testing.T) {
	old := VersionedTaskContent{Title: " Hello ", Description: "body"}
	new := VersionedTaskContent{Title: "Hello", Description: "body"}
	changed, record := DiffVersionedTask(old, new)
	if record || len(changed) != 0 {
		t.Fatalf("whitespace-normalized equal must not record, got changed=%v record=%v", changed, record)
	}
}

func TestDiffVersionedTaskTitleAndDescription(t *testing.T) {
	old := VersionedTaskContent{Title: "A", Description: "x"}
	new := VersionedTaskContent{Title: "B", Description: "y"}
	changed, record := DiffVersionedTask(old, new)
	if !record {
		t.Fatal("expected record")
	}
	if len(changed) != 2 || changed[0] != "title" || changed[1] != "description" {
		t.Fatalf("changed=%v", changed)
	}
}

func TestDiffVersionedTaskTitleOnly(t *testing.T) {
	changed, record := DiffVersionedTask(
		VersionedTaskContent{Title: "A", Description: "same"},
		VersionedTaskContent{Title: "B", Description: "same"},
	)
	if !record || len(changed) != 1 || changed[0] != "title" {
		t.Fatalf("changed=%v record=%v", changed, record)
	}
}

func TestNextVersion(t *testing.T) {
	if NextVersion(0) != 1 {
		t.Fatalf("empty table want 1, got %d", NextVersion(0))
	}
	if NextVersion(3) != 4 {
		t.Fatalf("want 4, got %d", NextVersion(3))
	}
	if NextVersion(-1) != 1 {
		t.Fatalf("negative max treated as 0")
	}
}

func TestShouldRecordCreate(t *testing.T) {
	if !ShouldRecordCreate() {
		t.Fatal("create must always record v1")
	}
}

func TestCreateChangedFieldsAndTruncate(t *testing.T) {
	if CreateChangedFields() != "title,description" {
		t.Fatalf("create marker: %s", CreateChangedFields())
	}
	if JoinChangedFields([]string{"title"}) != "title" {
		t.Fatal("join")
	}
	long := stringsRepeat("你", 201)
	got := TruncateRunes(long, 200)
	if len([]rune(got)) != 200 {
		t.Fatalf("truncate runes=%d", len([]rune(got)))
	}
}

func stringsRepeat(s string, n int) string {
	out := make([]rune, 0, n)
	r := []rune(s)[0]
	for i := 0; i < n; i++ {
		out = append(out, r)
	}
	return string(out)
}
