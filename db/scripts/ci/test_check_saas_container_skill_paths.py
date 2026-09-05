#!/usr/bin/env python3
"""Smoke tests for check_saas_container_skill_paths (no pytest required).

OPT-20260822-011 — 断言 git-oauth OpenAPI 写接口路径与评论回复路径必须出现在
saas-machine-container.md §5.2，防止新增写接口后文档漂移。
"""

from __future__ import annotations

import importlib.util
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
CHECKER = ROOT / "db" / "scripts" / "ci" / "check_saas_container_skill_paths.py"


def _load():
    spec = importlib.util.spec_from_file_location("check_saas_container_skill_paths", CHECKER)
    assert spec and spec.loader
    mod = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = mod
    spec.loader.exec_module(mod)
    return mod


def run_checker(*extra: str) -> tuple[int, str]:
    completed = subprocess.run(
        [sys.executable, str(CHECKER), *extra],
        cwd=ROOT,
        capture_output=True,
        text=True,
        check=False,
    )
    return completed.returncode, (completed.stdout or "") + (completed.stderr or "")


def test_live_repo_passes() -> None:
    """真实仓库当前状态应通过（所有路径均已登记在文档）。"""
    code, out = run_checker("--root", str(ROOT))
    assert code == 0, out
    assert "OK:" in out, out


def test_extract_openapi_paths() -> None:
    mod = _load()
    src = '''
func openAPIMergeRequestPaths() map[string]any {
	return map[string]any{
		"/api/git-oauth/merge-request-status/tenant_id/{tid}/": map[string]any{},
		"/api/git-oauth/merge-request-merge/tenant_id/{tid}/": map[string]any{},
	}
}
'''
    paths = mod.extract_openapi_paths(src)
    assert paths == [
        "/api/git-oauth/merge-request-status/tenant_id/{tid}/",
        "/api/git-oauth/merge-request-merge/tenant_id/{tid}/",
    ], paths


def test_extract_openapi_paths_ignores_non_path_keys() -> None:
    mod = _load()
    src = '''
	"summary": "一键合并",
	"/api/git-oauth/merge-request-merge/tenant_id/{tid}/": map[string]any{},
'''
    paths = mod.extract_openapi_paths(src)
    assert paths == ["/api/git-oauth/merge-request-merge/tenant_id/{tid}/"], paths


def test_extract_comment_reply_template() -> None:
    mod = _load()
    src = '''
// parseTenantIDCommentReplyPath 解析:
// api/tenant_id/{tid}/workspaceId/{wid}/tasks/{taskId}/comments/{parent}
func parseTenantIDCommentReplyPath(urlPath string) ... {
'''
    tpl = mod.extract_comment_reply_template(src)
    assert tpl == "/api/tenant_id/{tid}/workspaceId/{wid}/tasks/{taskId}/comments/{parent_comment_id}/", tpl


def test_missing_path_fails() -> None:
    """新增写接口未同步文档时检查必须失败。"""
    mod = _load()
    # 模拟：解析到的路径含文档缺失的新接口
    src = '''
	"/api/git-oauth/merge-request-merge/tenant_id/{tid}/": map[string]any{},
	"/api/git-oauth/new-endpoint/tenant_id/{tid}/": map[string]any{},
'''
    doc = "/api/git-oauth/merge-request-merge/tenant_id/{tid}/"
    missing = [p for p in mod.extract_openapi_paths(src) + [mod.extract_comment_reply_template("// api/tenant_id/{tid}/workspaceId/{wid}/tasks/{taskId}/comments/{parent}")] if mod.normalize_path(p) not in doc]
    assert "/api/git-oauth/new-endpoint/tenant_id/{tid}/" in missing, missing


def test_normalize_path_adds_slash() -> None:
    mod = _load()
    assert mod.normalize_path("api/foo") == "/api/foo"
    assert mod.normalize_path("/api/foo") == "/api/foo"
