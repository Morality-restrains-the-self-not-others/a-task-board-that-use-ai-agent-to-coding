package daydaymoneymeta

import "testing"

func TestParseYAML_OK(t *testing.T) {
	raw := `
version: 1
service_id: taskProjectService
display_name: Task Project Service
tags:
  - svc:taskProjectService
  - domain:project
`
	m, err := ParseYAML(raw)
	if err != nil {
		t.Fatalf("ParseYAML: %v", err)
	}
	if m.ServiceID != "taskProjectService" {
		t.Fatalf("service_id=%q", m.ServiceID)
	}
	if !TagsContain(m.Tags, "svc:taskProjectService") {
		t.Fatalf("tags=%v", m.Tags)
	}
}

func TestParseYAML_ForbiddenWorkspaceID(t *testing.T) {
	raw := `
version: 1
service_id: foo
workspace_id: ws-1
tags:
  - svc:foo
`
	if _, err := ParseYAML(raw); err == nil {
		t.Fatal("expected forbidden key error")
	}
}

func TestParseYAML_MissingSvcTag_AutoAddedThenOK(t *testing.T) {
	// NormalizeTags adds svc: during Validate — but Validate requires svc after normalize.
	// If user omits svc tag, NormalizeTags prepends it → should succeed.
	raw := `
version: 1
service_id: fooBar
tags:
  - domain:x
`
	m, err := ParseYAML(raw)
	if err != nil {
		t.Fatalf("expected auto svc tag, got %v", err)
	}
	if !TagsContain(m.Tags, "svc:fooBar") {
		t.Fatalf("tags=%v", m.Tags)
	}
}

func TestMergeTags_Union(t *testing.T) {
	got := MergeTags([]string{"team:a", "svc:foo"}, []string{"svc:foo", "domain:x"}, "foo")
	if !TagsContain(got, "team:a") || !TagsContain(got, "domain:x") || !TagsContain(got, "svc:foo") {
		t.Fatalf("got=%v", got)
	}
}
