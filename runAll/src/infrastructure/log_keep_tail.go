package infrastructure

import (
	"fmt"
	"io"
	"os"
)

// OPT-20260905-005: orchestrator console (nohup tee of runAll stdout) must retain a
// live tail after TruncateAll / shell truncate — .runall/last_orchestrator_exit.json
// is primary; this is a secondary breadcrumb in the live log path.
const (
	runallConsoleLogBaseName   = "runall-console.log"
	runallConsoleKeepTailBytes = 256 * 1024 // 256 KiB
)

// truncateFileKeepTail rewrites path to its last keepBytes (same path; best-effort
// same-inode when possible). Empty/missing files become empty create.
func truncateFileKeepTail(path string, keepBytes int64) error {
	if keepBytes <= 0 {
		return fmt.Errorf("keepBytes must be positive")
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			f, createErr := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
			if createErr != nil {
				return createErr
			}
			return f.Close()
		}
		return err
	}
	size := info.Size()
	if size == 0 {
		return nil
	}
	var tail []byte
	if size <= keepBytes {
		tail, err = os.ReadFile(path)
		if err != nil {
			return err
		}
	} else {
		f, openErr := os.Open(path)
		if openErr != nil {
			return openErr
		}
		if _, err := f.Seek(size-keepBytes, io.SeekStart); err != nil {
			_ = f.Close()
			return err
		}
		tail = make([]byte, keepBytes)
		n, readErr := io.ReadFull(f, tail)
		_ = f.Close()
		if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
			return readErr
		}
		tail = tail[:n]
	}
	// Prefer truncating the existing inode so an open nohup FD keeps writing here.
	f, err := os.OpenFile(path, os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	if err := f.Truncate(0); err != nil {
		_ = f.Close()
		return err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		_ = f.Close()
		return err
	}
	if _, err := f.Write(tail); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
