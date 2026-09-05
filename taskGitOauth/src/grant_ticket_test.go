package main

import "testing"

func TestIssueAndConsumeGrantTicket(t *testing.T) {
	app := testApp(t)
	id, err := app.DB.IssueGrantTicket("u1", "GitHub.com", "99", 0)
	if err != nil || id == "" {
		t.Fatalf("issue: id=%q err=%v", id, err)
	}
	remote, ok, err := app.DB.ConsumeGrantTicket(id, "u1", "github.com")
	if err != nil || !ok || remote != "99" {
		t.Fatalf("consume: remote=%q ok=%v err=%v", remote, ok, err)
	}
	_, ok, err = app.DB.ConsumeGrantTicket(id, "u1", "github.com")
	if err != nil || ok {
		t.Fatalf("second consume must fail: ok=%v err=%v", ok, err)
	}
}
