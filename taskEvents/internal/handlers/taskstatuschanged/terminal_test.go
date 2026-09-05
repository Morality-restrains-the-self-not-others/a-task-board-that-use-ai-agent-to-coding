package taskstatuschanged

import "testing"

func TestResolveTerminalKind(t *testing.T) {
	cases := []struct {
		name                string
		columnName          string
		completedBecameTrue bool
		want                string
	}{
		{"zh completed", "已完成", false, "completed"},
		{"en Completed", "Completed", false, "completed"},
		{"en completed lower", "completed", false, "completed"},
		{"zh cancelled", "已取消", false, "cancelled"},
		{"en Cancelled", "Cancelled", false, "cancelled"},
		{"en canceled", "canceled", false, "cancelled"},
		{"en cancelled", "cancelled", false, "cancelled"},
		{"in progress", "进行中", false, ""},
		{"empty column", "", false, ""},
		{"completed flag only", "", true, "completed"},
		{"cancelled column beats completed flag", "已取消", true, "cancelled"},
		{"Completed with flag", "Completed", true, "completed"},
		{"whitespace", "  completed  ", false, "completed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveTerminalKind(tc.columnName, tc.completedBecameTrue)
			if got != tc.want {
				t.Fatalf("ResolveTerminalKind(%q, %v) = %q, want %q",
					tc.columnName, tc.completedBecameTrue, got, tc.want)
			}
		})
	}
}
