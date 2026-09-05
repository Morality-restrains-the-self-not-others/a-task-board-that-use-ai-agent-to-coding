package main

import "testing"

// parseBootstrapEmailInviteArgs 是 bootstrap-email-invite CLI 的参数解析纯函数，
// 接收完整 os.Args（argv[0]=程序名，argv[1]=子命令）。
// 回归: CLI 曾把 "--email" flag 本身当作邮箱值（OPT-20260805 修复）。
func TestParseBootstrapEmailInviteArgs(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"flag form", []string{"taskAuth", "bootstrap-email-invite", "--email", "a@b.com"}, "a@b.com"},
		{"positional form", []string{"taskAuth", "bootstrap-email-invite", "a@b.com"}, "a@b.com"},
		{"no email", []string{"taskAuth", "bootstrap-email-invite"}, ""},
		{"flag without value", []string{"taskAuth", "bootstrap-email-invite", "--email"}, ""},
		{"email flag repeated", []string{"taskAuth", "bootstrap-email-invite", "--email", "a@b.com", "--email", "c@d.com"}, "c@d.com"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseBootstrapEmailInviteArgs(tc.args); got != tc.want {
				t.Fatalf("parseBootstrapEmailInviteArgs(%v) = %q, want %q", tc.args, got, tc.want)
			}
		})
	}
}
