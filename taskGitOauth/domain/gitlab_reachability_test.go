package domain

import "testing"

func TestClassifyReachability(t *testing.T) {
	cases := []struct {
		name       string
		configured bool
		intranet   bool
		wantSkip   bool
		wantStatus string
	}{
		{name: "empty", configured: false, intranet: false, wantSkip: true, wantStatus: ReachabilityUnconfigured},
		{name: "empty still skip when intranet", configured: false, intranet: true, wantSkip: true, wantStatus: ReachabilityUnconfigured},
		{name: "intranet allows unreachable", configured: true, intranet: true, wantSkip: true, wantStatus: ReachabilitySkippedIntranet},
		{name: "public must probe", configured: true, intranet: false, wantSkip: false, wantStatus: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			skip, status := ClassifyReachability(tc.configured, tc.intranet)
			if skip != tc.wantSkip || status != tc.wantStatus {
				t.Fatalf("skip=%v status=%q want skip=%v status=%q", skip, status, tc.wantSkip, tc.wantStatus)
			}
		})
	}
}

func TestOAuthProbeNetworkStatus(t *testing.T) {
	cases := []struct {
		name            string
		isGitLab        bool
		intranet        bool
		liveUnreachable bool
		want            string
	}{
		{name: "github omits", isGitLab: false, want: ""},
		{name: "intranet skip even if down", isGitLab: true, intranet: true, liveUnreachable: true, want: ReachabilitySkippedIntranet},
		{name: "public gitlab down", isGitLab: true, liveUnreachable: true, want: ReachabilityUnreachable},
		{name: "public gitlab up", isGitLab: true, want: OAuthNetworkOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := OAuthProbeNetworkStatus(tc.isGitLab, tc.intranet, tc.liveUnreachable)
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestRepoHTTPOrigin(t *testing.T) {
	if got := RepoHTTPOrigin("http://115.29.110.74/ljy124/repo.git"); got != "http://115.29.110.74" {
		t.Fatalf("got %q", got)
	}
	if got := RepoHTTPOrigin("https://gitlab.daydaymoney.com:8443/g/r.git"); got != "https://gitlab.daydaymoney.com:8443" {
		t.Fatalf("got %q", got)
	}
	if got := RepoHTTPOrigin("git@host:group/repo.git"); got != "" {
		t.Fatalf("ssh got %q", got)
	}
}

func TestProbeTargetURL(t *testing.T) {
	if got := ProbeTargetURL("http://115.29.110.74"); got != "http://115.29.110.74/" {
		t.Fatalf("got %q", got)
	}
	if got := ProbeTargetURL(" https://gitlab.daydaymoney.com/ "); got != "https://gitlab.daydaymoney.com/" {
		t.Fatalf("got %q", got)
	}
	if got := ProbeTargetURL(""); got != "" {
		t.Fatalf("empty got %q", got)
	}
}
