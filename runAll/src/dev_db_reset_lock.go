package main

import "sync"

var devDBResetFlag struct {
	mu    sync.Mutex
	busy  bool
}

func tryAcquireDevDBResetFlag() bool {
	devDBResetFlag.mu.Lock()
	defer devDBResetFlag.mu.Unlock()
	if devDBResetFlag.busy {
		return false
	}
	devDBResetFlag.busy = true
	return true
}

func releaseDevDBResetFlag() {
	devDBResetFlag.mu.Lock()
	defer devDBResetFlag.mu.Unlock()
	devDBResetFlag.busy = false
}
