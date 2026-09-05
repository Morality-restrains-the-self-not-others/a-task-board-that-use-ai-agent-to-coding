package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	internalAPIsSmokeStatusPending  = "pending"
	internalAPIsSmokeStatusOK       = "ok"
	internalAPIsSmokeStatusFailed   = "failed"
	internalAPIsSmokeStatusSkipped  = "skipped"
	internalAPIsSmokeDefaultTimeout = 90 * time.Second
)

// InternalAPIsSmokeReport is the last runAll-triggered live smoke result.
type InternalAPIsSmokeReport struct {
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	ExitCode  int       `json:"exit_code"`
	Output    string    `json:"output,omitempty"`
	CheckedAt time.Time `json:"checked_at,omitempty"`
	Trigger   string    `json:"trigger,omitempty"` // manual | start-all
}

func pendingInternalAPIsSmokeReport() InternalAPIsSmokeReport {
	return InternalAPIsSmokeReport{
		Status:  internalAPIsSmokeStatusPending,
		Message: "尚未执行 internal API live smoke，请点击「立即验证」或完成「全部启动」",
	}
}

func (r *Runner) GetInternalAPIsSmokeReport() InternalAPIsSmokeReport {
	if r == nil {
		return InternalAPIsSmokeReport{Status: internalAPIsSmokeStatusPending, Message: "runner unavailable"}
	}
	r.internalAPIsSmokeMu.RLock()
	defer r.internalAPIsSmokeMu.RUnlock()
	if r.internalAPIsSmokeLast == nil {
		return pendingInternalAPIsSmokeReport()
	}
	return *r.internalAPIsSmokeLast
}

func (r *Runner) setInternalAPIsSmokeReport(rep InternalAPIsSmokeReport) {
	if r == nil {
		return
	}
	r.internalAPIsSmokeMu.Lock()
	cp := rep
	r.internalAPIsSmokeLast = &cp
	r.internalAPIsSmokeMu.Unlock()
}

// runInternalAPIsSmokeScript executes scripts/smoke/internal-apis-live.sh.
// mode: required | optional
var runInternalAPIsSmokeScript = defaultRunInternalAPIsSmokeScript

func defaultRunInternalAPIsSmokeScript(ctx context.Context, root, mode string) (exitCode int, output string, err error) {
	script := filepath.Join(root, "scripts", "smoke", "internal-apis-live.sh")
	cmd := exec.CommandContext(ctx, "bash", script)
	cmd.Dir = root
	cmd.Env = append(cmd.Environ(),
		"INTERNAL_API_SMOKE_MODE="+strings.TrimSpace(mode),
	)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	runErr := cmd.Run()
	out := buf.String()
	if runErr == nil {
		return 0, out, nil
	}
	if ee, ok := runErr.(*exec.ExitError); ok {
		return ee.ExitCode(), out, nil
	}
	return -1, out, runErr
}

func (r *Runner) VerifyInternalAPIsSmoke(ctx context.Context, trigger string) InternalAPIsSmokeReport {
	if r == nil {
		return InternalAPIsSmokeReport{Status: internalAPIsSmokeStatusFailed, Message: "runner is required", Trigger: trigger}
	}
	root, err := r.monorepoRoot()
	if err != nil {
		rep := InternalAPIsSmokeReport{
			Status:    internalAPIsSmokeStatusFailed,
			Message:   fmt.Sprintf("monorepo root: %v", err),
			CheckedAt: time.Now(),
			Trigger:   trigger,
		}
		r.setInternalAPIsSmokeReport(rep)
		return rep
	}

	timeout := internalAPIsSmokeDefaultTimeout
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining > 0 && remaining < timeout {
			timeout = remaining
		}
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	exitCode, output, runErr := runInternalAPIsSmokeScript(runCtx, root, "required")
	rep := InternalAPIsSmokeReport{
		ExitCode:  exitCode,
		Output:    trimSmokeOutput(output),
		CheckedAt: time.Now(),
		Trigger:   trigger,
	}
	switch {
	case runErr != nil:
		rep.Status = internalAPIsSmokeStatusFailed
		rep.Message = runErr.Error()
	case exitCode == 0 && strings.Contains(output, "SKIPPED:"):
		rep.Status = internalAPIsSmokeStatusSkipped
		rep.Message = "stack not up (skipped)"
	case exitCode == 0:
		rep.Status = internalAPIsSmokeStatusOK
		rep.Message = summarizeSmokeOutput(output)
	default:
		rep.Status = internalAPIsSmokeStatusFailed
		rep.Message = summarizeSmokeOutput(output)
		if rep.Message == "" {
			rep.Message = fmt.Sprintf("smoke exit %d", exitCode)
		}
	}
	r.setInternalAPIsSmokeReport(rep)
	return rep
}

func (r *Runner) maybeRunInternalAPIsSmokeAfterStartAll(failed int) {
	if r == nil || failed > 0 {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), internalAPIsSmokeDefaultTimeout)
		defer cancel()
		rep := r.VerifyInternalAPIsSmoke(ctx, "start-all")
		log.Printf("[runAll] internal-apis smoke after start-all: status=%s exit=%d msg=%s",
			rep.Status, rep.ExitCode, rep.Message)
	}()
}

func trimSmokeOutput(s string) string {
	s = strings.TrimSpace(s)
	const max = 8000
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n…(truncated)"
}

func summarizeSmokeOutput(s string) string {
	lines := strings.Split(s, "\n")
	var summary string
	var fails []string
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "=== summary:") {
			summary = line
			continue
		}
		if strings.HasPrefix(line, "FAIL  ") || strings.HasPrefix(line, "FAIL\t") {
			fails = append(fails, line)
		}
	}
	if len(fails) > 0 {
		// Prefer concrete FAIL lines so UI can show why smoke failed.
		joined := strings.Join(fails, " | ")
		const maxFailMsg = 400
		if len(joined) > maxFailMsg {
			joined = joined[:maxFailMsg] + "…"
		}
		if summary != "" {
			return joined + " · " + summary
		}
		return joined
	}
	if summary != "" {
		return summary
	}
	return strings.TrimSpace(s)
}
