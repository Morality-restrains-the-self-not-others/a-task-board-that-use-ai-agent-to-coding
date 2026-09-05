package infrastructure

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type FileServiceLogSink struct {
	mu           sync.Mutex
	rootDir      string
	servicePaths map[string]string // per-service custom log file paths
	files        map[string]*os.File
}

func NewFileServiceLogSink(rootDir string) (*FileServiceLogSink, error) {
	root := strings.TrimSpace(rootDir)
	if root == "" {
		return nil, fmt.Errorf("log root directory is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create log root: %w", err)
	}
	return &FileServiceLogSink{
		rootDir:      root,
		servicePaths: make(map[string]string),
		files:        make(map[string]*os.File),
	}, nil
}

// SetServicePath registers a custom log file path for a service.
// When set, the service's logs are written to this path instead of
// the default <rootDir>/<serviceName>.log.
func (s *FileServiceLogSink) SetServicePath(serviceName, path string) {
	if s == nil {
		return
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.servicePaths[serviceName] = path
}

func (s *FileServiceLogSink) AppendLine(serviceName, stream, message string) error {
	if s == nil {
		return fmt.Errorf("file log sink is nil")
	}
	serviceName = strings.TrimSpace(serviceName)
	if serviceName == "" {
		return fmt.Errorf("service name is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := s.fileForServiceLocked(serviceName)
	if err != nil {
		return err
	}

	ts := time.Now().UTC().Format(time.RFC3339Nano)
	line := fmt.Sprintf("%s (%s) %s\n", ts, stream, message)
	_, err = file.WriteString(line)
	return err
}

func (s *FileServiceLogSink) TruncateService(serviceName string) error {
	if s == nil {
		return fmt.Errorf("file log sink is nil")
	}
	serviceName = strings.TrimSpace(serviceName)
	if serviceName == "" {
		return fmt.Errorf("service name is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.closeServiceLocked(serviceName)

	path := s.pathForServiceLocked(serviceName)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create log dir for %q: %w", path, err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("truncate log file %q: %w", path, err)
	}
	// Keep the fresh handle so subsequent AppendLine writes the same inode Promtail tails.
	s.files[serviceName] = file
	return nil
}

func (s *FileServiceLogSink) TruncateAll() (int, error) {
	if s == nil {
		return 0, fmt.Errorf("file log sink is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for name := range s.files {
		s.closeServiceLocked(name)
	}

	paths := make(map[string]struct{})
	for _, path := range s.servicePaths {
		paths[path] = struct{}{}
	}
	entries, err := os.ReadDir(s.rootDir)
	if err != nil {
		return 0, fmt.Errorf("read log root: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}
		paths[filepath.Join(s.rootDir, entry.Name())] = struct{}{}
	}

	count := 0
	for path := range paths {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return count, fmt.Errorf("create log dir for %q: %w", path, err)
		}
		// OPT-20260905-005: keep a live tail of the orchestrator console so
		// :9999-empty-window forensics survive hourly truncate without digging archive.
		if filepath.Base(path) == runallConsoleLogBaseName {
			if err := truncateFileKeepTail(path, runallConsoleKeepTailBytes); err != nil {
				return count, fmt.Errorf("keep-tail truncate %q: %w", path, err)
			}
			count++
			continue
		}
		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			return count, fmt.Errorf("truncate log file %q: %w", path, err)
		}
		if err := file.Close(); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (s *FileServiceLogSink) Close() error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var firstErr error
	for name := range s.files {
		if err := s.closeServiceLocked(name); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (s *FileServiceLogSink) pathForServiceLocked(serviceName string) string {
	if cp, ok := s.servicePaths[serviceName]; ok {
		return cp
	}
	return filepath.Join(s.rootDir, serviceName+".log")
}

func (s *FileServiceLogSink) closeServiceLocked(serviceName string) error {
	file, ok := s.files[serviceName]
	if !ok {
		return nil
	}
	delete(s.files, serviceName)
	return file.Close()
}

func (s *FileServiceLogSink) fileForServiceLocked(serviceName string) (*os.File, error) {
	path := s.pathForServiceLocked(serviceName)
	if file, ok := s.files[serviceName]; ok {
		if !isStaleLogFileHandle(file, path) {
			return file, nil
		}
		_ = s.closeServiceLocked(serviceName)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create log dir for %q: %w", path, err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open log file %q: %w", path, err)
	}
	s.files[serviceName] = file
	return file, nil
}

// isStaleLogFileHandle reports whether the open FD no longer refers to the path on disk
// (e.g. the file was unlinked/recreated while runAll kept the old inode open).
func isStaleLogFileHandle(file *os.File, path string) bool {
	if file == nil {
		return true
	}
	fileInfo, err := file.Stat()
	if err != nil {
		return true
	}
	pathInfo, err := os.Stat(path)
	if err != nil {
		return true
	}
	return !os.SameFile(fileInfo, pathInfo)
}
