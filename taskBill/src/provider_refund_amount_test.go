package main

import "testing"

func TestProportionalAmountMinor(t *testing.T) {
	if got := proportionalAmountMinor(1000, 1000, 600); got != 600 {
		t.Fatalf("got %d", got)
	}
	if got := proportionalAmountMinor(1000, 1000, 1000); got != 1000 {
		t.Fatalf("full got %d", got)
	}
	if got := proportionalAmountMinor(1000, 1000, 0); got != 0 {
		t.Fatalf("zero got %d", got)
	}
}

func TestFormatMoneyMinor(t *testing.T) {
	if got := formatMoneyMinor(1050); got != "10.50" {
		t.Fatalf("got %q", got)
	}
	if got := formatMoneyMinor(5); got != "0.05" {
		t.Fatalf("got %q", got)
	}
}
