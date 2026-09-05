package main

import "testing"

func TestParseIDField_rejectsFloat64(t *testing.T) {
	_, err := parseIDField(float64(827923618468040704))
	if err == nil {
		t.Fatal("expected error for float64 id")
	}
}

func TestParseIDField_acceptsString(t *testing.T) {
	id, err := parseIDField("827923618468040704")
	if err != nil {
		t.Fatal(err)
	}
	if id != 827923618468040704 {
		t.Fatalf("got %d", id)
	}
}

func TestFormatID(t *testing.T) {
	if formatID(828887302689878016) != "828887302689878016" {
		t.Fatal("formatID mismatch")
	}
}
