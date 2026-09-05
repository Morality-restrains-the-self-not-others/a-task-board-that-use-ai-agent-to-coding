package main

import "context"

// compileThenSwapContextKey marks a restart that must compile before swapping
// the running process (ADR-0027 precise-restart). Plain RestartService /
// RestartAll leave this unset and must not invoke build_command.
type compileThenSwapContextKey struct{}

func withCompileThenSwap(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, compileThenSwapContextKey{}, true)
}

func compileThenSwapFrom(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v, _ := ctx.Value(compileThenSwapContextKey{}).(bool)
	return v
}
