package smtp

import (
	"errors"
	"testing"
)

func TestIsPermanent(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{
			name: "qq 535 login fail",
			err:  errors.New("535 Login fail. Account is abnormal, service is not open, password is incorrect, login frequency limited"),
			want: true,
		},
		{name: "550 user unknown", err: errors.New("550 User not found"), want: true},
		{name: "dial timeout", err: errors.New("dial tcp: i/o timeout"), want: false},
		{name: "dns misbehaving", err: errors.New("lookup smtp.qq.com on 127.0.0.53:53: server misbehaving"), want: false},
		{name: "connection refused", err: errors.New("dial tcp 127.0.0.1:1: connect: connection refused"), want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := IsPermanent(tc.err); got != tc.want {
				t.Fatalf("IsPermanent(%v)=%v want %v", tc.err, got, tc.want)
			}
		})
	}
}
