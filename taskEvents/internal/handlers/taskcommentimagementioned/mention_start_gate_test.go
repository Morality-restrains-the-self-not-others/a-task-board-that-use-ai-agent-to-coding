package taskcommentimagementioned

import "testing"

func TestShouldColdStartOnImageMention(t *testing.T) {
	tests := []struct {
		name string
		data map[string]interface{}
		want bool
	}{
		{
			name: "independent starts even with predecessors",
			data: map[string]interface{}{
				"execution_mode":              "independent",
				"has_unfinished_predecessors": true,
				"depends_on_comment_ids":      []interface{}{"cmt_prev"},
			},
			want: true,
		},
		{
			name: "wait_previous with unfinished predecessors does not start",
			data: map[string]interface{}{
				"execution_mode":              "wait_previous",
				"has_unfinished_predecessors": true,
			},
			want: false,
		},
		{
			name: "wait_previous with explicit depends_on does not start",
			data: map[string]interface{}{
				"execution_mode":         "wait_previous",
				"depends_on_comment_ids": []interface{}{"cmt_prev"},
			},
			want: false,
		},
		{
			name: "wait_previous first comment without predecessors starts",
			data: map[string]interface{}{
				"execution_mode":              "wait_previous",
				"has_unfinished_predecessors": false,
			},
			want: true,
		},
		{
			name: "missing execution_mode defaults to wait_previous and starts when no predecessors",
			data: map[string]interface{}{"comment_id": "cmt_1"},
			want: true,
		},
		{
			name: "legacy event without mode but has_unfinished_predecessors skips start",
			data: map[string]interface{}{"has_unfinished_predecessors": true},
			want: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldColdStartOnImageMention(tc.data)
			if got != tc.want {
				t.Fatalf("got=%v want=%v data=%v", got, tc.want, tc.data)
			}
		})
	}
}
