package main

import (
	"errors"
	"syscall"
	"testing"
	"time"
)

// OPT-20260820-006: after a successful /api/shutdown-self (2xx) the old instance
// is already exiting with managed services kept alive. Sending a SIGTERM on top
// would flip its skipShutdownServices=false and stop every healthy service, so
// the retirement path must only signal when the POST failed or the exit never
// happens.

func newTestRetireHooks(shutdownStatus int, shutdownErr error, alive func(int) bool) (*previousRunAllHooks, *[]syscall.Signal) {
	var kills []syscall.Signal
	h := &previousRunAllHooks{
		postShutdownSelf: func() (int, error) { return shutdownStatus, shutdownErr },
		alive:            alive,
		kill: func(pid int, sig syscall.Signal) error {
			kills = append(kills, sig)
			return nil
		},
		gracefulWait: 30 * time.Millisecond,
	}
	return h, &kills
}

func TestKillPreviousRunAllProcess_SkipsOccupied9999DuringGoTest(t *testing.T) {
	posted := 0
	oldLookup := lookupUIPortListeners
	oldPost := postUIShutdownSelf
	t.Cleanup(func() {
		lookupUIPortListeners = oldLookup
		postUIShutdownSelf = oldPost
	})
	lookupUIPortListeners = func(port string) ([]int, error) {
		if port == "9999" {
			return []int{424242}, nil
		}
		return nil, nil
	}
	postUIShutdownSelf = func(url string) (int, error) {
		posted++
		t.Fatalf("must not POST shutdown-self to production 9999 during go test, url=%s", url)
		return 200, nil
	}
	result := killPreviousRunAllProcess(":9999")
	if posted != 0 {
		t.Fatalf("posted=%d want 0", posted)
	}
	if result.HadPrevious || result.GracefulShutdownSelf {
		t.Fatalf("result=%+v want no retirement of :9999 during go test", result)
	}
}

func TestKillPreviousRunAllProcess_RetiresNonProductionPortDuringGoTest(t *testing.T) {
	posted := 0
	oldLookup := lookupUIPortListeners
	oldPost := postUIShutdownSelf
	t.Cleanup(func() {
		lookupUIPortListeners = oldLookup
		postUIShutdownSelf = oldPost
	})
	lookupUIPortListeners = func(port string) ([]int, error) {
		if port == "18001" {
			return []int{424243}, nil
		}
		return nil, nil
	}
	postUIShutdownSelf = func(url string) (int, error) {
		posted++
		if url != "http://localhost:18001/api/shutdown-self" {
			t.Fatalf("url=%s", url)
		}
		return 200, nil
	}
	result := killPreviousRunAllProcess(":18001")
	if posted != 1 {
		t.Fatalf("posted=%d want 1", posted)
	}
	if !result.HadPrevious || !result.GracefulShutdownSelf {
		t.Fatalf("result=%+v want retire temp UI port", result)
	}
}

func TestRetirePreviousRunAll_ShutdownSelf200AlreadyExitedSkipsSignal(t *testing.T) {
	h, kills := newTestRetireHooks(200, nil, func(int) bool { return false })
	result := retirePreviousRunAll(999999, h)
	if !result.GracefulShutdownSelf {
		t.Fatalf("GracefulShutdownSelf = false, want true")
	}
	if len(*kills) != 0 {
		t.Fatalf("kill() signals = %v, want none (shutdown-self exit must not be SIGTERM'd)", *kills)
	}
}

func TestRetirePreviousRunAll_ShutdownSelf200ExitTimeoutFallsBackToSigterm(t *testing.T) {
	alive := true
	h, kills := newTestRetireHooks(200, nil, func(int) bool { return alive })
	h.kill = func(pid int, sig syscall.Signal) error {
		*kills = append(*kills, sig)
		alive = false // SIGTERM makes it exit
		return nil
	}
	result := retirePreviousRunAll(999999, h)
	if !result.GracefulShutdownSelf {
		t.Fatalf("GracefulShutdownSelf = false, want true")
	}
	if len(*kills) != 1 || (*kills)[0] != syscall.SIGTERM {
		t.Fatalf("kill() signals = %v, want exactly [SIGTERM]", *kills)
	}
}

func TestRetirePreviousRunAll_PostFailureSendsSigterm(t *testing.T) {
	h, kills := newTestRetireHooks(0, errors.New("connection refused"), func(int) bool { return false })
	result := retirePreviousRunAll(999999, h)
	if result.GracefulShutdownSelf {
		t.Fatalf("GracefulShutdownSelf = true, want false")
	}
	if len(*kills) != 1 || (*kills)[0] != syscall.SIGTERM {
		t.Fatalf("kill() signals = %v, want exactly [SIGTERM]", *kills)
	}
}

func TestRetirePreviousRunAll_Non2xxSendsSigterm(t *testing.T) {
	h, kills := newTestRetireHooks(500, nil, func(int) bool { return false })
	result := retirePreviousRunAll(999999, h)
	if result.GracefulShutdownSelf {
		t.Fatalf("GracefulShutdownSelf = true, want false")
	}
	if len(*kills) != 1 || (*kills)[0] != syscall.SIGTERM {
		t.Fatalf("kill() signals = %v, want exactly [SIGTERM]", *kills)
	}
}

func TestRetirePreviousRunAll_SigtermIgnoredEscalatesToSigkill(t *testing.T) {
	h, kills := newTestRetireHooks(0, errors.New("refused"), func(int) bool { return true }) // never dies
	_ = retirePreviousRunAll(999999, h)
	if len(*kills) != 2 || (*kills)[0] != syscall.SIGTERM || (*kills)[1] != syscall.SIGKILL {
		t.Fatalf("kill() signals = %v, want [SIGTERM SIGKILL]", *kills)
	}
}
