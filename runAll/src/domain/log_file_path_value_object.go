package domain

import (
	"fmt"
	"path/filepath"
	"strings"
)

// LogFilePath is a validated log file path value object.
type LogFilePath struct {
	Value string
}

// NewLogFilePath creates a validated log file path.
func NewLogFilePath(path string) (LogFilePath, error) {
	p := strings.TrimSpace(path)
	if p == "" {
		return LogFilePath{}, fmt.Errorf("log file path must not be empty")
	}
	return LogFilePath{Value: p}, nil
}

// IsAbsolute reports whether the path is absolute.
func (p LogFilePath) IsAbsolute() bool {
	return filepath.IsAbs(p.Value)
}

// Dir returns the parent directory of the log file.
func (p LogFilePath) Dir() string {
	return filepath.Dir(p.Value)
}
