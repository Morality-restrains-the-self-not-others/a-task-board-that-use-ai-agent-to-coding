package main

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"runAll/src/domain"
)

const stdioFileTailScanMultiplier = 4

func (r *Runner) prepareManagedStdioLogPath(svc Service) (string, error) {
	if r == nil {
		return "", os.ErrInvalid
	}
	if r.cfg != nil && strings.TrimSpace(r.cfg.Logging.FileRoot) != "" {
		path := resolveServiceLogFile(r.cfg, svc.Name, svc.LogFile)
		r.setStdioLogPath(svc.Name, path)
		return path, nil
	}
	safe := safeStdioFileToken(svc.Name)
	f, err := os.CreateTemp("", "runall-stdio-"+safe+"-*.log")
	if err != nil {
		return "", err
	}
	path := f.Name()
	if err := f.Close(); err != nil {
		return "", err
	}
	r.setStdioLogPath(svc.Name, path)
	return path, nil
}

func (r *Runner) setStdioLogPath(name, path string) {
	if r == nil {
		return
	}
	r.stdioLogMu.Lock()
	defer r.stdioLogMu.Unlock()
	if r.stdioLogPaths == nil {
		r.stdioLogPaths = make(map[string]string)
	}
	r.stdioLogPaths[name] = path
}

func (r *Runner) stdioLogPath(name string) string {
	if r == nil {
		return ""
	}
	r.stdioLogMu.Lock()
	path := ""
	if r.stdioLogPaths != nil {
		path = r.stdioLogPaths[name]
	}
	r.stdioLogMu.Unlock()
	if path != "" {
		return path
	}
	if r.cfg == nil || strings.TrimSpace(r.cfg.Logging.FileRoot) == "" {
		return ""
	}
	confApp := ""
	if svc := r.findService(name); svc != nil {
		confApp = svc.LogFile
	}
	return resolveServiceLogFile(r.cfg, name, confApp)
}

func openManagedServiceStdioFile(path string) (*os.File, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, os.ErrInvalid
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
}

func attachManagedServiceStdio(cmd *exec.Cmd, file *os.File) {
	if cmd == nil || file == nil {
		return
	}
	cmd.Stdout = file
	cmd.Stderr = file
}

func closeParentStdioFile(file *os.File) error {
	if file == nil {
		return nil
	}
	return file.Close()
}

func safeStdioFileToken(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "svc"
	}
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, name)
}

func tailFileLines(path string, n int) ([]string, error) {
	if strings.TrimSpace(path) == "" || n <= 0 {
		return nil, nil
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines = append(lines, line)
		if len(lines) > n {
			lines = lines[len(lines)-n:]
		}
	}
	if err := scanner.Err(); err != nil && err != io.EOF {
		return lines, err
	}
	return lines, nil
}

func classifyStdioFileLine(line string) (stream, message string) {
	line = strings.TrimRight(line, "\r")
	const stderrMark = " (stderr) "
	const stdoutMark = " (stdout) "
	if i := strings.Index(line, stderrMark); i >= 0 {
		return domain.StreamStderr, strings.TrimSpace(line[i+len(stderrMark):])
	}
	if i := strings.Index(line, stdoutMark); i >= 0 {
		return domain.StreamStdout, strings.TrimSpace(line[i+len(stdoutMark):])
	}
	// Child writes inherit a single file FD; treat unprefixed lines as stderr
	// so launch-failure enrichment still surfaces Python/Go tracebacks.
	return domain.StreamStderr, strings.TrimSpace(line)
}

func (r *Runner) recentStdioFileMessages(serviceName, stream string, limit int) []string {
	if r == nil || limit <= 0 {
		return nil
	}
	lines, err := tailFileLines(r.stdioLogPath(serviceName), limit*stdioFileTailScanMultiplier)
	if err != nil || len(lines) == 0 {
		return nil
	}
	out := make([]string, 0, limit)
	for i := len(lines) - 1; i >= 0; i-- {
		gotStream, msg := classifyStdioFileLine(lines[i])
		if msg == "" {
			continue
		}
		if stream != "" && gotStream != stream {
			continue
		}
		out = append(out, msg)
		if len(out) >= limit {
			break
		}
	}
	if len(out) == 0 {
		return nil
	}
	for left, right := 0, len(out)-1; left < right; left, right = left+1, right-1 {
		out[left], out[right] = out[right], out[left]
	}
	return out
}

func (r *Runner) stdioFileLogEntries(serviceName string, limit int) []domain.LogEntry {
	if r == nil || limit <= 0 {
		return nil
	}
	lines, err := tailFileLines(r.stdioLogPath(serviceName), limit)
	if err != nil || len(lines) == 0 {
		return nil
	}
	now := time.Now()
	out := make([]domain.LogEntry, 0, len(lines))
	for _, line := range lines {
		stream, msg := classifyStdioFileLine(line)
		if msg == "" {
			continue
		}
		entry, err := domain.NewLogEntry(now, serviceName, stream, msg)
		if err != nil {
			continue
		}
		out = append(out, entry)
	}
	return out
}
