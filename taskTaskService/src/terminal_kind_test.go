package main

import "testing"

func TestResolveTerminalKind(t *testing.T) {
	cases := []struct {
		name      string
		col       string
		completed bool
		want      string
	}{
		{"zh completed", "已完成", false, "completed"},
		{"zh cancelled", "已取消", false, "cancelled"},
		{"en completed", "completed", false, "completed"},
		{"en cancelled", "cancelled", false, "cancelled"},
		{"flag only", "", true, "completed"},
		{"open", "进行中", false, ""},
		{"cancel beats flag", "已取消", true, "cancelled"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveTerminalKind(tc.col, tc.completed)
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestUpdateEntersTerminal(t *testing.T) {
	cases := []struct {
		name          string
		prevCol, col  string
		colName       string
		prevCompleted bool
		completed     bool
		want          bool
	}{
		{"to completed column", "col-wip", "col-done", "已完成", false, false, true},
		{"to cancelled column", "col-wip", "col-cancel", "已取消", false, false, true},
		{"to in progress", "col-todo", "col-wip", "进行中", false, false, false},
		{"completed flag", "col-wip", "col-wip", "", false, true, true},
		{"same completed column", "col-done", "col-done", "已完成", true, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := updateEntersTerminal(tc.prevCol, tc.col, tc.colName, tc.prevCompleted, tc.completed)
			if got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
