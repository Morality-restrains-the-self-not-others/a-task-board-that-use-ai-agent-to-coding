package domain

import "testing"

func TestNewSessionTermination_EmptyBase(t *testing.T) {
	_, err := NewSessionTermination("", "/auth/login/")
	if err == nil {
		t.Error("expected error for empty GitServicePublicBase, got nil")
	}
}

func TestGitLabSignOutURL_WithRedirect(t *testing.T) {
	st, err := NewSessionTermination("http://127.0.0.1:8012", "http://127.0.0.1:4000/auth/login/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "http://127.0.0.1:8012/users/sign_out?redirect_uri=http%3A%2F%2F127.0.0.1%3A4000%2Fauth%2Flogin%2F"
	if got := st.GitLabSignOutURL(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestGitLabSignOutURL_WithoutRedirect(t *testing.T) {
	st, err := NewSessionTermination("http://127.0.0.1:8012", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "http://127.0.0.1:8012/users/sign_out"
	if got := st.GitLabSignOutURL(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestSessionTermination_Equals(t *testing.T) {
	a, _ := NewSessionTermination("http://127.0.0.1:8012", "/auth/login/")
	b, _ := NewSessionTermination("http://127.0.0.1:8012", "/auth/login/")
	c, _ := NewSessionTermination("http://other:8012", "/auth/login/")

	if !a.Equals(b) {
		t.Error("expected a.Equals(b) to be true")
	}
	if a.Equals(c) {
		t.Error("expected a.Equals(c) to be false")
	}
	if a.Equals(nil) {
		t.Error("expected a.Equals(nil) to be false")
	}
}
