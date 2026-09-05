package domain

import "strings"

// VersionedTaskContent is the subset of Task that participates in history.
type VersionedTaskContent struct {
	Title       string
	Description string
}

// NormalizeText trims versioned string fields so whitespace-only edits do not record.
func NormalizeText(s string) string {
	return strings.TrimSpace(s)
}

// DiffVersionedTask reports which versioned fields changed.
func DiffVersionedTask(old, new VersionedTaskContent) (changed []string, record bool) {
	if NormalizeText(old.Title) != NormalizeText(new.Title) {
		changed = append(changed, "title")
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

// ShouldRecordCreate is always true: creating a task writes revision v1.
func ShouldRecordCreate() bool {
	return true
}

// JoinChangedFields serializes changed field names as a comma-separated list.
func JoinChangedFields(fields []string) string {
	return strings.Join(fields, ",")
}

// CreateChangedFields is the snapshot marker written on entity create.
func CreateChangedFields() string {
	return "title,description"
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
