package main

import (
	"strings"
	"testing"
)

func TestTailLines_TruncatesLongOutput(t *testing.T) {
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, "line")
	}
	input := strings.Join(lines, "\n")
	got := tailLines(input, 80)
	gotLines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(gotLines) != 80 {
		t.Fatalf("got %d lines, want 80", len(gotLines))
	}
	if !strings.HasSuffix(got, "line") {
		t.Fatalf("tail should end with last line")
	}
}
