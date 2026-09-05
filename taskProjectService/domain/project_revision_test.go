package domain

import "testing"

func TestDiffVersionedProjectNoChange(t *testing.T) {
	old := VersionedProjectContent{Name: " P ", Description: "body"}
	new := VersionedProjectContent{Name: "P", Description: "body"}
	changed, record := DiffVersionedProject(old, new)
	if record || len(changed) != 0 {
		t.Fatalf("whitespace-normalized equal must not record, got changed=%v record=%v", changed, record)
	}
}

func TestDiffVersionedProjectNameAndDescription(t *testing.T) {
	changed, record := DiffVersionedProject(
		VersionedProjectContent{Name: "A", Description: "x"},
		VersionedProjectContent{Name: "B", Description: "y"},
	)
	if !record || len(changed) != 2 || changed[0] != "name" || changed[1] != "description" {
		t.Fatalf("changed=%v record=%v", changed, record)
	}
}

func TestNextVersionAndCreate(t *testing.T) {
	if NextVersion(0) != 1 || !ShouldRecordCreate() {
		t.Fatal("create v1")
	}
	if CreateChangedFields() != "name,description" {
		t.Fatal(CreateChangedFields())
	}
	if TruncateRunes("abcdef", 3) != "abc" {
		t.Fatal("truncate")
	}
}
