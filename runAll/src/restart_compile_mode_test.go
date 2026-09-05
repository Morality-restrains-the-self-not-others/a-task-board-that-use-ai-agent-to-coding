package main

import (
	"context"
	"testing"
)

func TestCompileThenSwapFrom_UnsetIsFalse(t *testing.T) {
	if compileThenSwapFrom(context.Background()) {
		t.Fatal("plain context must not enable compile-then-swap")
	}
	if compileThenSwapFrom(nil) {
		t.Fatal("nil context must not enable compile-then-swap")
	}
}

func TestWithCompileThenSwap_SetsFlag(t *testing.T) {
	ctx := withCompileThenSwap(context.Background())
	if !compileThenSwapFrom(ctx) {
		t.Fatal("withCompileThenSwap must set compile-then-swap flag")
	}
}
