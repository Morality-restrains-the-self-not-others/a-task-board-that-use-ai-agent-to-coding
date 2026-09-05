package main

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestGenIDSnowflakeFormat(t *testing.T) {
	id := genID("csc")
	suffix, ok := strings.CutPrefix(id, "csc_")
	if !ok {
		t.Fatalf("genID(csc) = %q, want csc_<digits>", id)
	}
	if !regexp.MustCompile(`^\d+$`).MatchString(suffix) {
		t.Fatalf("genID(csc) = %q, suffix %q not all digits", id, suffix)
	}
	n, err := strconv.ParseInt(suffix, 10, 64)
	if err != nil {
		t.Fatalf("genID(csc) = %q not int64: %v", id, err)
	}
	if n <= 0 {
		t.Fatalf("genID(csc) = %q, suffix must be positive snowflake", id)
	}
}
