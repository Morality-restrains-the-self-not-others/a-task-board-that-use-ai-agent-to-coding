package main

import "testing"

func TestParseFieldName_Valid(t *testing.T) {
	p, tbl, col, err := ParseFieldName("saas-backend.auth_user.email")
	if err != nil {
		t.Fatal(err)
	}
	if p != "saas-backend" || tbl != "auth_user" || col != "email" {
		t.Fatalf("got %q %q %q", p, tbl, col)
	}
}

// taskFE is the runAll frontend service name (CamelCase); service segment
// must accept it so value-stream can start with conf/value-stream.yaml.
func TestParseFieldName_CamelCaseService(t *testing.T) {
	p, tbl, col, err := ParseFieldName("taskFE.runtime.currenttenant_resolution")
	if err != nil {
		t.Fatal(err)
	}
	if p != "taskFE" || tbl != "runtime" || col != "currenttenant_resolution" {
		t.Fatalf("got %q %q %q", p, tbl, col)
	}
}

func TestParseFieldName_InvalidSegments(t *testing.T) {
	_, _, _, err := ParseFieldName("saas-backend.email")
	if err == nil {
		t.Fatal("want error for two segments")
	}
}
