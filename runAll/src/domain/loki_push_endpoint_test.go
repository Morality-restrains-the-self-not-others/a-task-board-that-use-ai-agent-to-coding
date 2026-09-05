package domain

import "testing"

func TestLokiPushURL_FromBase(t *testing.T) {
	got, err := LokiPushURL("http://10.2.150.119:3100")
	if err != nil {
		t.Fatal(err)
	}
	want := "http://10.2.150.119:3100/loki/api/v1/push"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestLokiPushURL_Empty(t *testing.T) {
	if _, err := LokiPushURL(""); err != ErrInvalidLokiURL {
		t.Fatalf("err = %v", err)
	}
}
