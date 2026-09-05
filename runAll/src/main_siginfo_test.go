package main

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestFormatSignalOriginFields_IncludesChildPid(t *testing.T) {
	origin := signalOrigin{Pid: 4242, Uid: 1000, Code: 0}
	got := formatSignalOriginFields(origin)
	for _, want := range []string{"si_pid=4242", "si_uid=1000", "si_code=0"} {
		if !strings.Contains(got, want) {
			t.Fatalf("formatSignalOriginFields = %q, want substring %q", got, want)
		}
	}
	logged := formatTerminationSignalLog(syscall.SIGTERM, origin)
	if !strings.Contains(logged, "terminated") || !strings.Contains(logged, "si_pid=4242") {
		t.Fatalf("formatTerminationSignalLog = %q", logged)
	}
}

// TestFormatTerminationSignalLog_OmitsOriginWhenUnknown pins the signal.Notify
// contract: the sender identity is no longer captured (si_pid=0 would falsely
// imply a sender), so the log must carry only the signal name.
func TestFormatTerminationSignalLog_OmitsOriginWhenUnknown(t *testing.T) {
	got := formatTerminationSignalLog(syscall.SIGTERM, signalOrigin{})
	if got != syscall.SIGTERM.String() {
		t.Fatalf("formatTerminationSignalLog(unknown) = %q, want %q", got, syscall.SIGTERM.String())
	}
	if strings.Contains(got, "si_pid=") {
		t.Fatalf("formatTerminationSignalLog(unknown) must not print si_pid fields, got %q", got)
	}
}

// TestTerminationSignal_SurvivesProcessDirectedSIGTERM is the regression test for
// nightly flake 2026-09-01: a process-directed SIGTERM used to race the Go
// runtime's unblocked-thread delivery and kill the orchestrator ("signal:
// terminated") before applyOrchestratorExitKeepPolicy could run. The production
// wait path (waitTerminationSignal) is signal.Notify-based now, so the helper
// subprocess must survive its own SIGTERM and observe the signal.
func TestTerminationSignal_SurvivesProcessDirectedSIGTERM(t *testing.T) {
	if os.Getenv("RUNALL_SIGINFO_HELPER") == "1" {
		os.Exit(runSiginfoHelper())
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestTerminationSignal_SurvivesProcessDirectedSIGTERM$", "-test.count=1")
	cmd.Env = append(os.Environ(), "RUNALL_SIGINFO_HELPER=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("helper died on process-directed SIGTERM (regression): %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "SIGTERM_RECEIVED") {
		t.Fatalf("helper output missing SIGTERM_RECEIVED:\n%s", out)
	}
}

func runSiginfoHelper() int {
	// Exercise the exact production wait path. No os/signal.Notify is registered
	// here directly: waitTerminationSignal owns it. A process-directed SIGTERM
	// must be forwarded to the channel and never abort the process.
	type waitResult struct {
		sig    os.Signal
		origin signalOrigin
		err    error
	}
	results := make(chan waitResult, 1)
	go func() {
		s, origin, err := waitTerminationSignal(nil)
		results <- waitResult{sig: s, origin: origin, err: err}
	}()
	time.Sleep(50 * time.Millisecond)

	killPath, err := exec.LookPath("kill")
	if err != nil {
		os.Stderr.WriteString("lookPath kill: " + err.Error() + "\n")
		return 2
	}
	child, err := os.StartProcess(killPath, []string{"kill", "-TERM", strconv.Itoa(os.Getpid())}, &os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	})
	if err != nil {
		os.Stderr.WriteString("start kill: " + err.Error() + "\n")
		return 2
	}
	_, _ = child.Wait()

	select {
	case got := <-results:
		if got.err != nil {
			os.Stderr.WriteString("waitTerminationSignal: " + got.err.Error() + "\n")
			return 1
		}
		if got.sig != syscall.SIGTERM {
			os.Stderr.WriteString("want SIGTERM, got " + got.sig.String() + "\n")
			return 1
		}
		os.Stdout.WriteString("SIGTERM_RECEIVED\n")
		return 0
	case <-time.After(5 * time.Second):
		os.Stderr.WriteString("timed out waiting for SIGTERM from child\n")
		return 1
	}
}
