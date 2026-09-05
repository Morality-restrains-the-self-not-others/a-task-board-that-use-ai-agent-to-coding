package domain

import "strings"

// VersionedProjectContent is the subset of Project that participates in history.
type VersionedProjectContent struct {
	Name        string
	Description string
}

// NormalizeText trims versioned string fields so whitespace-only edits do not record.
func NormalizeText(s string) string {
	return strings.TrimSpace(s)
}

// DiffVersionedProject reports which versioned fields changed.
func DiffVersionedProject(old, new VersionedProjectContent) (changed []string, record bool) {
	if NormalizeText(old.Name) != NormalizeText(new.Name) {
		changed = append(changed, "name")
	}
	if NormalizeText(old.Description) != NormalizeText(new.Description) {
		changed = append(changed, "description")
	}
	return changed, len(changed) > 0
}

// NextVersion returns the next append-only version number (MAX+1).
func NextVersion(max int) int {
	if max < 0 {
		max = 0
	}
	return max + 1
}

// ShouldRecordCreate is always true: creating a project writes revision v1.
func ShouldRecordCreate() bool {
	return true
}

// JoinChangedFields serializes changed field names as a comma-separated list.
func JoinChangedFields(fields []string) string {
	return strings.Join(fields, ",")
}

// CreateChangedFields is the snapshot marker written on entity create.
func CreateChangedFields() string {
	return "name,description"
}

// TruncateRunes shortens s to at most n runes (list API description preview).
func TruncateRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
