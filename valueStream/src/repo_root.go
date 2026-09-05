package main

import (
	"os"
	"path/filepath"
	"runtime"
)

// repoRoot returns monorepo root (parent of valueStream/).
func repoRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", os.ErrInvalid
	}
	// .../valueStream/src/repo_root.go -> .../ram-mount
	return filepath.Abs(filepath.Join(filepath.Dir(file), "..", ".."))
}

func productionValueStreamConfigPath() (string, error) {
	root, err := repoRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "conf", "value-stream.yaml"), nil
}
