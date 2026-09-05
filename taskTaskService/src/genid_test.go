package main

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// OPT-20260814-018: genID 对齐 35_snowflake_id_generation.md。
// 格式锁定为 <prefix>_<snowflake digits>；雪花 ID 落在 int64 范围内，
// 旧实现 unixNano*1000+seq（~1.7e21）会溢出 int64 —— 本断言可区分新旧形态。

func TestGenIDSnowflakeFormat(t *testing.T) {
	for _, prefix := range []string{"task", "cmt", "gi", "tp", "tfp", "snap", "tri"} {
		id := genID(prefix)
		suffix, ok := strings.CutPrefix(id, prefix+"_")
		if !ok {
			t.Fatalf("genID(%q) = %q, want %q_<digits>", prefix, id, prefix)
		}
		if !regexp.MustCompile(`^\d+$`).MatchString(suffix) {
			t.Fatalf("genID(%q) = %q, suffix %q not all digits", prefix, id, suffix)
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

func TestGenIDUnique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		id := genID("task")
		if seen[id] {
			t.Fatalf("genID collision: %s", id)
		}
		seen[id] = true
	}
}

func TestGenIDKeepsPrefix(t *testing.T) {
	id := genID("task")
	if !strings.HasPrefix(id, "task_") {
		t.Fatalf("genID(task) = %q, want task_ prefix", id)
	}
}
