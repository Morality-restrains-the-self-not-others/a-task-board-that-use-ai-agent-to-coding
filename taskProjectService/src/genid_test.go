package main

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestGenIDSnowflakeFormat(t *testing.T) {
	for _, prefix := range []string{"ws", "proj", "ds"} {
		id := genID(prefix)
		suffix, ok := strings.CutPrefix(id, prefix+"_")
		if !ok {
			t.Fatalf("genID(%q) = %q, want %q_<digits>", prefix, id, prefix)
		}
		if !regexp.MustCompile(`^\d+$`).MatchString(suffix) {
			t.Fatalf("genID(%q) = %q, suffix %q not all digits (overflow IDs look like ws_-123)", prefix, id, suffix)
		}
		n, err := strconv.ParseInt(suffix, 10, 64)
		if err != nil {
			t.Fatalf("genID(%q) = %q, suffix %q not int64: %v", prefix, id, suffix, err)
		}
		if n <= 0 {
			t.Fatalf("genID(%q) = %q, suffix must be positive snowflake", prefix, id)
		}
	}
}
