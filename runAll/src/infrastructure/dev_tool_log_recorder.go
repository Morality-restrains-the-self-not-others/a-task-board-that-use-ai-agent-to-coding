package infrastructure

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"runAll/src/domain"
)

// FileDevToolLogRecorder 将开发工具日志写入 <rootDir>/logs/<tool>.log 文件。
type FileDevToolLogRecorder struct {
	rootDir string
}

// NewFileDevToolLogRecorder 创建文件日志记录器。rootDir 为 monorepo 根路径。
func NewFileDevToolLogRecorder(rootDir string) *FileDevToolLogRecorder {
	return &FileDevToolLogRecorder{rootDir: rootDir}
}

func (r *FileDevToolLogRecorder) logDir() string {
	return filepath.Join(r.rootDir, "logs")
}

func (r *FileDevToolLogRecorder) logPath(tool string) string {
	return filepath.Join(r.logDir(), tool+".log")
}

func (r *FileDevToolLogRecorder) Append(tool string, message string) error {
	if err := domain.ValidateDevTool(tool); err != nil {
		return err
	}

	if err := os.MkdirAll(r.logDir(), 0755); err != nil {
		return fmt.Errorf("dev_tool_log: mkdir %s: %w", r.logDir(), err)
	}

	f, err := os.OpenFile(r.logPath(tool), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("dev_tool_log: open %s: %w", r.logPath(tool), err)
	}
	defer f.Close()

	ts := time.Now().Format("2006-01-02 15:04:05")
	line := fmt.Sprintf("[%s] %s\n", ts, message)
	if _, err := f.WriteString(line); err != nil {
		return fmt.Errorf("dev_tool_log: write %s: %w", r.logPath(tool), err)
	}
	return nil
}

func (r *FileDevToolLogRecorder) Tail(tool string, lines int) ([]string, error) {
	if err := domain.ValidateDevTool(tool); err != nil {
		return nil, err
	}
	if lines <= 0 {
		return []string{}, nil
	}

	f, err := os.Open(r.logPath(tool))
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("dev_tool_log: open %s: %w", r.logPath(tool), err)
	}
	defer f.Close()

	// 从文件尾部逐行回退读取，最多读 64KB buffer
	return tailLines(f, lines, 64*1024), nil
}

func (r *FileDevToolLogRecorder) Clear(tool string) error {
	if err := domain.ValidateDevTool(tool); err != nil {
		return err
	}

	if err := os.MkdirAll(r.logDir(), 0755); err != nil {
		return fmt.Errorf("dev_tool_log: mkdir %s: %w", r.logDir(), err)
	}

	f, err := os.OpenFile(r.logPath(tool), os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("dev_tool_log: truncate %s: %w", r.logPath(tool), err)
	}
	return f.Close()
}

func (r *FileDevToolLogRecorder) ClearAll() error {
	var errs []error
	for _, tool := range domain.AllDevTools {
		if err := r.Clear(tool); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// tailLines 从文件尾部读取最后 N 行，使用固定大小 buffer 循环回退读取。
// 返回按时间顺序排列的行（旧→新），如果文件不足 N 行则返回全部。
func tailLines(f *os.File, maxLines int, bufSize int) []string {
	stat, err := f.Stat()
	if err != nil || stat.Size() == 0 {
		return []string{}
	}

	fileSize := stat.Size()
	pos := fileSize
	var collected []string

	for pos > 0 && len(collected) < maxLines {
		readSize := int64(bufSize)
		if pos < readSize {
			readSize = pos
		}
		pos -= readSize

		chunk := make([]byte, readSize)
		if _, err := f.ReadAt(chunk, pos); err != nil {
			break
		}

		text := string(chunk)
		chunkLines := strings.Split(text, "\n")

		// 如果不是第一次读取，合并部分行：
		// 当前 chunk 的最后一行实际上是被截断的，需要与已收集的第一行拼接
		if len(collected) > 0 && len(chunkLines) > 0 {
			collected[0] = chunkLines[len(chunkLines)-1] + collected[0]
			chunkLines = chunkLines[:len(chunkLines)-1]
		}

		// 如果不在文件开头，丢弃 chunk 的第一行（它从行中间开始）
		if pos > 0 && len(chunkLines) > 0 {
			chunkLines = chunkLines[1:]
		}

		// 过滤空行并将新行前置到 collected
		var filtered []string
		for _, line := range chunkLines {
			line = strings.TrimSuffix(line, "\r")
			if line != "" || pos > 0 {
				filtered = append(filtered, line)
			}
		}
		collected = append(filtered, collected...)

		// 限制行数
		if len(collected) > maxLines {
			collected = collected[len(collected)-maxLines:]
		}
	}

	// 清理末尾空字符串
	for len(collected) > 0 && collected[len(collected)-1] == "" {
		collected = collected[:len(collected)-1]
	}

	return collected
}

// 确保实现了接口
var _ domain.DevToolLogRecorder = (*FileDevToolLogRecorder)(nil)
