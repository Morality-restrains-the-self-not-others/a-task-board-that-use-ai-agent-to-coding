package interfaces

import "testing"

func TestParseTokenInitPath(t *testing.T) {
	tid, wid, tk, cid, ok := parseTokenInitPath(
		"/v1/token/init/tenant/850256677331562496/workspace/861623708318031872/task/task_12939023414154091865/comment/cmt_1",
	)
	if !ok {
		t.Fatal("expected ok")
	}
	if tid != "850256677331562496" || wid != "861623708318031872" || tk != "task_12939023414154091865" || cid != "cmt_1" {
		t.Fatalf("got tenant=%q workspace=%q task=%q comment=%q", tid, wid, tk, cid)
	}
}

func TestParseTokenInitPathRejectsLegacyAndIncomplete(t *testing.T) {
	cases := []string{
		"/v1/token/init",
		"/v1/token/init/",
		"/v1/token/init/tenant/t1/workspace/w1",
		"/v1/token/init/tenant/850256677331562496/workspace/861623708318031872/task/task_12939023414154091865",
		"/v1/token/init/tenant//workspace/w1/task/tk",
		"/v1/token/init/tenant/t1/workspace/w1/task/tk/comment/",
		"/v1/token/init/tenant/t1/workspace/w1/task/tk/comment/-",
		"/api/tenant/t1/workspace/w1/task/tk/v1/token/init",
	}
	for _, p := range cases {
		if _, _, _, _, ok := parseTokenInitPath(p); ok {
			t.Fatalf("expected reject for %q", p)
		}
	}
}
