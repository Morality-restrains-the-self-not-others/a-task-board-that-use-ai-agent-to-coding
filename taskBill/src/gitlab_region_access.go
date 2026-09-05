package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

const (
	gitlabRegionAccessRelease     = "release"
	gitlabRegionAccessDevelopment = "development"
)

// errGitlabRegionDevModeForbidden 开发模式区域拒绝非测试账号。
var errGitlabRegionDevModeForbidden = errors.New("该区域处于开发模式，仅测试角色账号可使用")

type testerCtxKey struct{}

func contextWithTester(ctx context.Context, isTester bool) context.Context {
	return context.WithValue(ctx, testerCtxKey{}, isTester)
}

func testerFromContext(ctx context.Context) bool {
	v, _ := ctx.Value(testerCtxKey{}).(bool)
	return v
}

func requestIsTester(r *http.Request) bool {
	if r == nil {
		return false
	}
	return strings.TrimSpace(r.Header.Get("X-User-Is-Tester")) == "1"
}

// NormalizeGitlabRegionAccessMode maps empty/unknown values to release.
func NormalizeGitlabRegionAccessMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case gitlabRegionAccessDevelopment:
		return gitlabRegionAccessDevelopment
	default:
		return gitlabRegionAccessRelease
	}
}

// ParseGitlabRegionAccessMode accepts only release|development.
func ParseGitlabRegionAccessMode(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return gitlabRegionAccessRelease, nil
	}
	if s == gitlabRegionAccessRelease || s == gitlabRegionAccessDevelopment {
		return s, nil
	}
	return "", fmt.Errorf("access_mode 须为 release 或 development")
}

// CanUseGitlabRegion 发布模式任意账号可用；开发模式仅测试账号。
func CanUseGitlabRegion(accessMode string, isTester bool) bool {
	if NormalizeGitlabRegionAccessMode(accessMode) == gitlabRegionAccessDevelopment {
		return isTester
	}
	return true
}

func filterUsableGitlabRegions(regions []GitlabRegion, isTester bool) []GitlabRegion {
	out := make([]GitlabRegion, 0, len(regions))
	for _, r := range regions {
		if CanUseGitlabRegion(r.AccessMode, isTester) {
			out = append(out, r)
		}
	}
	return out
}
