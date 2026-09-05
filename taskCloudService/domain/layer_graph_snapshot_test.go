package domain

import "testing"

func TestLayerGraphSnapshotIDValidate(t *testing.T) {
	if err := (LayerGraphSnapshotID{WorkspaceID: "w", TaskID: "t", CommentID: "c"}).Validate(); err != nil {
		t.Fatalf("want valid: %v", err)
	}
	if err := (LayerGraphSnapshotID{WorkspaceID: " ", TaskID: "t", CommentID: "c"}).Validate(); err == nil {
		t.Fatal("empty workspace should fail")
	}
	if err := (LayerGraphSnapshotID{WorkspaceID: "w", TaskID: "", CommentID: "c"}).Validate(); err == nil {
		t.Fatal("empty task should fail")
	}
	if err := (LayerGraphSnapshotID{WorkspaceID: "w", TaskID: "t", CommentID: ""}).Validate(); err == nil {
		t.Fatal("empty comment should fail")
	}
}

func TestValidateGraphDocument(t *testing.T) {
	if err := ValidateGraphDocument([]byte(`{"layers":[],"jobs":[]}`)); err != nil {
		t.Fatal(err)
	}
	if err := ValidateGraphDocument([]byte(`{"layers":{},"jobs":[]}`)); err == nil {
		t.Fatal("layers object should fail")
	}
	if err := ValidateGraphDocument([]byte(`{"jobs":[]}`)); err == nil {
		t.Fatal("missing layers should fail")
	}
}

func TestEmptyGraphDocument(t *testing.T) {
	doc := EmptyGraphDocument()
	layers, _ := doc["layers"].([]any)
	jobs, _ := doc["jobs"].([]any)
	if layers == nil || jobs == nil || len(layers) != 0 || len(jobs) != 0 {
		t.Fatalf("empty graph=%v", doc)
	}
}
