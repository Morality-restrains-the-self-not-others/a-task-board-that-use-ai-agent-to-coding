#!/usr/bin/env python3
"""Characterization: git-service 资源键 SSOT 在 conf（元规则 47）+ 解析模块回归。

OPT-20260812-033 后解析逻辑在 gitService/scripts/load_gitservice_config.py，
run.sh 调用该脚本；本测直接 import 覆盖 conf-read JSON / YAML 兜底两分支。
"""
from __future__ import annotations

import json
import os
import sys
from pathlib import Path

import yaml

REPO = Path(__file__).resolve().parents[3]
CONF = REPO / "conf" / "infra" / "git-service" / "config.yaml"
LOADER = REPO / "gitService" / "scripts" / "load_gitservice_config.py"

sys.path.insert(0, str(LOADER.parent))
import load_gitservice_config as lgc  # noqa: E402


def test_config_yaml_has_resource_ssot_keys() -> None:
    data = yaml.safe_load(CONF.read_text(encoding="utf-8"))
    assert isinstance(data, dict)
    assert data.get("memLimit"), "memLimit must be set in conf (human SSOT)"
    assert data.get("shmSize"), "shmSize must be set in conf"
    assert int(data.get("pumaWorkers") or 0) >= 1
    assert int(data.get("sidekiqConcurrency") or 0) >= 1
    assert int(data.get("minDockerMemoryMib") or 0) >= 1024
    # OPT-20260807-034：登录/注册开关 SSOT 在 conf，默认关闭注册与账密
    assert data.get("signupEnabled") is False, "signupEnabled must default to false in conf"
    assert data.get("passwordAuthWeb") is False, "passwordAuthWeb must default to false in conf"
    assert data.get("passwordAuthGit") is False, "passwordAuthGit must default to false in conf"
    assert data.get("imageTag") == "19.2.4-ce.0", "imageTag SSOT must pin Omnibus CE tag"
    assert data.get("runAllStartEnabled") is False, (
        "runAllStartEnabled must default to false (ADR-0047); opt-in via conf-local"
    )


def test_signup_switches_flow_to_compose_vars() -> None:
    """登录/注册开关经 loader 透出为 compose 插值变量（与 conf 一致）。"""
    data = yaml.safe_load(CONF.read_text(encoding="utf-8"))
    out = lgc.resolve("", CONF)
    want = "true" if data.get("signupEnabled") else "false"
    assert out["GITLAB_SIGNUP_ENABLED"] == want, out
    assert out["GITLAB_PASSWORD_AUTH_WEB"] == ("true" if data.get("passwordAuthWeb") else "false")
    assert out["GITLAB_PASSWORD_AUTH_GIT"] == ("true" if data.get("passwordAuthGit") else "false")
    assert out["GITLAB_IMAGE"] == f"gitlab/gitlab-ce:{data.get('imageTag')}"


def test_compose_consumes_gitlab_image_env() -> None:
    compose = (REPO / "gitService" / "docker-compose.yml").read_text(encoding="utf-8")
    assert "image: ${GITLAB_IMAGE:-gitlab/gitlab-ce:19.2.4-ce.0}" in compose
    assert "gitlab/gitlab-ce:19.0.0-ce.0" not in compose


def test_region_instances_share_image_tag() -> None:
    data = yaml.safe_load(CONF.read_text(encoding="utf-8"))
    sh = yaml.safe_load(
        (REPO / "conf" / "infra" / "git-service-tencent-sh-1" / "config.yaml").read_text(
            encoding="utf-8"
        )
    )
    assert sh.get("imageTag") == data.get("imageTag")


def test_run_sh_calls_extracted_loader() -> None:
    """run.sh 应消费独立脚本，不再内嵌 heredoc（防双路径漂移）。"""
    script = (REPO / "gitService" / "run.sh").read_text(encoding="utf-8")
    assert "load_gitservice_config.py" in script, (
        "run.sh 应调用 scripts/load_gitservice_config.py"
    )
    assert "<<'PY'" not in script, (
        "load_gitservice_config 不应再以内嵌 heredoc 形式存在于 run.sh"
    )
    assert "ensure_gitlab_container.sh" in script, "run.sh 须 source 容器生命周期脚本"
    assert "local_gitlab_start_gate.sh" in script, "run.sh 须 source 本机 GitLab 启动闸门（ADR-0047）"
    assert "GITLAB_IMAGE_OVERRIDE" in script
    helper = (REPO / "gitService" / "scripts" / "ensure_gitlab_container.sh").read_text(
        encoding="utf-8"
    )
    assert "image 变更" in helper, "须检测 Omnibus 镜像漂移并重建"
    for name in (
        "apply_auth_policy.sh",
        "fix_oidc_traceid.sh",
        "fix_oidc_ssl.sh",
        "sync_omniauth_oidc.sh",
        "sync_local_oauth_app_scopes.sh",
    ):
        path = REPO / "gitService" / "scripts" / name
        assert path.is_file() and os.access(path, os.X_OK), f"{name} must be executable"


def test_run_sh_exports_mem_limit_from_conf() -> None:
    """YAML 兜底路径：conf memLimit 被导出。"""
    data = yaml.safe_load(CONF.read_text(encoding="utf-8"))
    want = str(data["memLimit"]).strip()
    out = lgc.resolve("", CONF)
    assert out["GITLAB_MEM_LIMIT"] == want, out


def test_run_sh_conf_read_success_path_defines_mem_limit() -> None:
    """回归：conf-read JSON 非空时 mem_limit 仍被解析（原 NameError）。"""
    data = yaml.safe_load(CONF.read_text(encoding="utf-8"))
    want = str(data["memLimit"]).strip()
    resolved = {
        "host": "0.0.0.0",
        "port": 8012,
        "sshPort": 22,
        "advertiseSshPort": 22,
        "allowedHost": "",
        "publicUrl": "",
        "gitlabHome": "/tmp/gitlab-home-test",
        "memLimit": want,
        "shmSize": str(data.get("shmSize") or "512m"),
        "pumaWorkers": int(data.get("pumaWorkers") or 2),
        "sidekiqConcurrency": int(data.get("sidekiqConcurrency") or 5),
        "minDockerMemoryMib": int(data.get("minDockerMemoryMib") or 6144),
    }
    out = lgc.resolve(json.dumps(resolved), Path("/nonexistent"))
    assert out["GITLAB_MEM_LIMIT"] == want, out
    assert "GITLAB_HTTP_PORT" in out


if __name__ == "__main__":
    test_config_yaml_has_resource_ssot_keys()
    test_signup_switches_flow_to_compose_vars()
    test_compose_consumes_gitlab_image_env()
    test_region_instances_share_image_tag()
    test_run_sh_calls_extracted_loader()
    test_run_sh_exports_mem_limit_from_conf()
    test_run_sh_conf_read_success_path_defines_mem_limit()
    print("ok")
