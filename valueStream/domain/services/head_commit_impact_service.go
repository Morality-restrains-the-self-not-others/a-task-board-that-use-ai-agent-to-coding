package services

import (
	"context"
	"fmt"
	"path"
	"strings"

	"valueStream/domain/repositories"
)

type ImpactStatus string

const (
	ImpactStatusKnown   ImpactStatus = "known"
	ImpactStatusUnknown ImpactStatus = "unknown"
)

// ImpactStep 描述用于影响判定的环节输入。
type ImpactStep struct {
	Name     string
	TestFile string
}

// ImpactStream 描述用于影响判定的价值流输入。
type ImpactStream struct {
	Name  string
	Steps []ImpactStep
}

// StepImpactResult 描述单个环节的影响结果。
type StepImpactResult struct {
	Name         string
	TestFile     string
	Impacted     bool
	ImpactStatus ImpactStatus
	Reason       string
}

// StreamImpactResult 描述单个价值流的影响结果。
type StreamImpactResult struct {
	Name          string
	Impacted      bool
	ImpactStatus  ImpactStatus
	ImpactedSteps []string
	Reason        string
	Steps         []StepImpactResult
}

// HeadCommitImpactService 编排 HEAD 提交影响判定用例。
type HeadCommitImpactService struct {
	repository repositories.CommitChangeRepository
}

func NewHeadCommitImpactService(repository repositories.CommitChangeRepository) *HeadCommitImpactService {
	return &HeadCommitImpactService{repository: repository}
}

func (s *HeadCommitImpactService) Evaluate(ctx context.Context, streams []ImpactStream) ([]StreamImpactResult, error) {
	if s.repository == nil {
		return nil, fmt.Errorf("commit change repository is required")
	}
	changedFiles, err := s.repository.ListHeadChangedFiles(ctx)
	if err != nil {
		return buildUnknownImpactResults(streams, err), nil
	}
	return buildKnownImpactResults(streams, changedFiles), nil
}

func buildKnownImpactResults(streams []ImpactStream, changedFiles []string) []StreamImpactResult {
	changedSet := make(map[string]struct{}, len(changedFiles))
	changedPaths := make([]string, 0, len(changedFiles))
	for _, changed := range changedFiles {
		normalized := normalizeImpactPath(changed)
		if normalized == "" {
			continue
		}
		if _, exists := changedSet[normalized]; exists {
			continue
		}
		changedSet[normalized] = struct{}{}
		changedPaths = append(changedPaths, normalized)
	}

	results := make([]StreamImpactResult, 0, len(streams))
	for _, stream := range streams {
		result := StreamImpactResult{
			Name:         stream.Name,
			ImpactStatus: ImpactStatusKnown,
		}
		for _, step := range stream.Steps {
			normalizedTestFile := normalizeImpactPath(step.TestFile)
			_, impacted := changedSet[normalizedTestFile]
			if !impacted {
				impacted = hasEquivalentChangedPath(normalizedTestFile, changedPaths)
			}
			stepResult := StepImpactResult{
				Name:         step.Name,
				TestFile:     step.TestFile,
				Impacted:     impacted,
				ImpactStatus: ImpactStatusKnown,
			}
			if impacted {
				stepResult.Reason = "step test_file changed in HEAD"
				result.ImpactedSteps = append(result.ImpactedSteps, step.Name)
				result.Impacted = true
			} else {
				stepResult.Reason = "step test_file not changed in HEAD"
			}
			result.Steps = append(result.Steps, stepResult)
		}
		if result.Impacted {
			result.Reason = "at least one step test_file changed in HEAD"
		} else {
			result.Reason = "no step test_file changed in HEAD"
		}
		results = append(results, result)
	}
	return results
}

func hasEquivalentChangedPath(testFile string, changedPaths []string) bool {
	if testFile == "" {
		return false
	}
	testAbs := isAbsoluteLikePath(testFile)
	for _, changed := range changedPaths {
		changedAbs := isAbsoluteLikePath(changed)
		if testAbs == changedAbs {
			continue
		}
		if strings.HasSuffix(testFile, "/"+changed) || strings.HasSuffix(changed, "/"+testFile) {
			return true
		}
	}
	return false
}

func isAbsoluteLikePath(p string) bool {
	if strings.HasPrefix(p, "/") {
		return true
	}
	return len(p) > 1 && p[1] == ':'
}

func buildUnknownImpactResults(streams []ImpactStream, cause error) []StreamImpactResult {
	results := make([]StreamImpactResult, 0, len(streams))
	for _, stream := range streams {
		result := StreamImpactResult{
			Name:         stream.Name,
			ImpactStatus: ImpactStatusUnknown,
			Reason:       fmt.Sprintf("head changed files unavailable: %v", cause),
		}
		for _, step := range stream.Steps {
			result.Steps = append(result.Steps, StepImpactResult{
				Name:         step.Name,
				TestFile:     step.TestFile,
				Impacted:     false,
				ImpactStatus: ImpactStatusUnknown,
				Reason:       "head changed files unavailable",
			})
		}
		results = append(results, result)
	}
	return results
}

func normalizeImpactPath(raw string) string {
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
