package main

import "testing"

func TestParseAutoCloneNestedReposFromProjectData(t *testing.T) {
	cases := []struct {
		name string
		data map[string]interface{}
		want bool
	}{
		{"missing", map[string]interface{}{}, true},
		{"true", map[string]interface{}{"auto_clone_nested_repos": true}, true},
		{"false", map[string]interface{}{"auto_clone_nested_repos": false}, false},
		{"zero", map[string]interface{}{"auto_clone_nested_repos": float64(0)}, false},
		{"str0", map[string]interface{}{"auto_clone_nested_repos": "0"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := autoCloneNestedReposFromMap(tc.data)
			if got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
