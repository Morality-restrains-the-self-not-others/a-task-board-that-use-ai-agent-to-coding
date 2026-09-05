package main

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path"
	"strings"

	"valueStream/domain/repositories"
)

var _ repositories.CommitChangeRepository = (*GitCommitChangeRepository)(nil)

// GitCommitChangeRepository 基于 git show 读取 HEAD 的变更文件列表。
type GitCommitChangeRepository struct {
	repoDir          string
	workingDirPrefix string
}

func NewGitCommitChangeRepository(repoDir, workingDir string) *GitCommitChangeRepository {
	return &GitCommitChangeRepository{
		repoDir:          strings.TrimSpace(repoDir),
		workingDirPrefix: normalizeGitPath(workingDir),
	}
}

func (r *GitCommitChangeRepository) ListHeadChangedFiles(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "show", "--name-only", "--pretty=format:", "HEAD")
	if r.repoDir != "" {
		cmd.Dir = r.repoDir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("git show HEAD changed files: %s", msg)
	}

	seen := make(map[string]struct{})
	paths := make([]string, 0)
	for _, line := range strings.Split(stdout.String(), "\n") {
		normalized := normalizeGitPath(line)
		if normalized == "" {
			continue
		}
		if r.workingDirPrefix != "" {
			if rest, ok := strings.CutPrefix(normalized, r.workingDirPrefix+"/"); ok {
				normalized = rest
			}
		}
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		paths = append(paths, normalized)
	}
	return paths, nil
}

func normalizeGitPath(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	slashPath := strings.ReplaceAll(trimmed, "\\", "/")
	cleaned := path.Clean(slashPath)
	if cleaned == "." {
		return ""
	}
	return strings.TrimPrefix(cleaned, "./")
}
