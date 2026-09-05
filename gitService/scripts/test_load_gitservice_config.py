#!/usr/bin/env python3
"""Unit tests: gitService 配置解析模块（conf-read JSON 优先 / YAML 兜底 / 模板剥离）。

OPT-20260812-033：解析逻辑抽为独立模块后，直接 import 覆盖两分支与资源键默认值，
避免 heredoc 双路径漂移复发。
"""
from __future__ import annotations

import json
import sys
import tempfile
from pathlib import Path

import yaml

SCRIPT = Path(__file__).resolve().parent / "load_gitservice_config.py"
sys.path.insert(0, str(SCRIPT.parent))
import load_gitservice_config as lgc  # noqa: E402


def _conf(mem_limit="6g", shm_size="512m", puma_workers=2,
          sidekiq_concurrency=5, min_docker_memory_mib=6144,
          signup_enabled=False, password_auth_web=False, password_auth_git=False):
    return {
        "allowedHost": "${scheme}://${subdomains.gitlab}",
        "host": "0.0.0.0",
        "port": 8012,
        "sshPort": 22,
        "advertiseSshPort": 22,
        "publicUrl": "${scheme}://${subdomains.gitlab}",
        "gitlabHome": "/tmp/gitlab-home-test",
        "memLimit": mem_limit,
        "shmSize": shm_size,
        "pumaWorkers": puma_workers,
        "sidekiqConcurrency": sidekiq_concurrency,
        "minDockerMemoryMib": min_docker_memory_mib,
        "signupEnabled": signup_enabled,
        "passwordAuthWeb": password_auth_web,
        "passwordAuthGit": password_auth_git,
    }


def _write_conf(tmp: Path, data: dict) -> Path:
    p = tmp / "config.yaml"
    p.write_text(yaml.safe_dump(data, allow_unicode=True), encoding="utf-8")
    return p


def test_conf_read_success_path_defines_resource_keys():
    """conf-read JSON 非空时，资源键必须仍被解析（回归 NameError）。"""
    resolved = _conf()
    out = lgc.resolve(json.dumps(resolved), Path("/nonexistent-main"))
    assert out["GITLAB_MEM_LIMIT"] == "6g"
    assert out["GITLAB_SHM_SIZE"] == "512m"
    assert out["GITLAB_PUMA_WORKERS"] == 2
    assert out["GITLAB_SIDEKIQ_CONCURRENCY"] == 5
    assert out["GITLAB_MIN_DOCKER_MEMORY_MIB"] == 6144
    # OPT-20260807-034：登录/注册开关随 conf 透出（默认关闭注册与账密）
    assert out["GITLAB_SIGNUP_ENABLED"] == "false"
    assert out["GITLAB_PASSWORD_AUTH_WEB"] == "false"
    assert out["GITLAB_PASSWORD_AUTH_GIT"] == "false"
    assert out["GITLAB_OIDC_CLIENT_ID"] == "gitlab-git-service"
    assert out["GITLAB_CONTAINER_NAME"] == "gitlab"
    assert out["GITLAB_IMAGE"] == "gitlab/gitlab-ce:19.2.4-ce.0"
    assert out["TRAE_TASKBILL_BASE"] == "http://host.docker.internal:8004"
    assert out["TRAE_GITLAB_REGION"] == ""


def test_signup_password_switches_reflect_conf():
    """conf 打开 signup/账密时，compose 插值变量同步为 true。"""
    resolved = _conf(signup_enabled=True, password_auth_web=True, password_auth_git=True)
    out = lgc.resolve(json.dumps(resolved), Path("/nope"))
    assert out["GITLAB_SIGNUP_ENABLED"] == "true"
    assert out["GITLAB_PASSWORD_AUTH_WEB"] == "true"
    assert out["GITLAB_PASSWORD_AUTH_GIT"] == "true"


def test_signup_switches_default_false_when_missing():
    """conf-read JSON 缺登录开关键时落到安全默认 false（关闭注册/账密）。"""
    resolved = {"host": "localhost", "port": 8012, "sshPort": 22}
    out = lgc.resolve(json.dumps(resolved), Path("/nope"))
    assert out["GITLAB_SIGNUP_ENABLED"] == "false"
    assert out["GITLAB_PASSWORD_AUTH_WEB"] == "false"
    assert out["GITLAB_PASSWORD_AUTH_GIT"] == "false"


def test_yaml_fallback_uses_conf_paths():
    """conf-read 不可用时 YAML 兜底：只合 config.yaml，忽略同目录 config.local.yaml。"""
    with tempfile.TemporaryDirectory() as td:
        tmp = Path(td)
        main = _write_conf(tmp, _conf())
        local = tmp / "config.local.yaml"
        local.write_text("port: 8013\n", encoding="utf-8")
        out = lgc.resolve("", main)
    assert out["GITLAB_HTTP_PORT"] == 8012
    assert out["GITLAB_MEM_LIMIT"] == "6g"


def test_yaml_fallback_conf_local_overrides_main():
    with tempfile.TemporaryDirectory() as td:
        tmp = Path(td)
        main = tmp / "conf" / "infra" / "git-service" / "config.yaml"
        main.parent.mkdir(parents=True)
        main.write_text(yaml.safe_dump(_conf() | {"port": 8012, "runAllStartEnabled": False}), encoding="utf-8")
        overlay = tmp / "conf-local" / "infra" / "git-service" / "config.yaml"
        overlay.parent.mkdir(parents=True)
        overlay.write_text("port: 8019\nrunAllStartEnabled: true\n", encoding="utf-8")
        stale = main.parent / "config.local.yaml"
        stale.write_text("port: 1\nrunAllStartEnabled: false\n", encoding="utf-8")
        out = lgc.resolve("", main)
    assert out["GITLAB_HTTP_PORT"] == 8019
    assert out["GITLAB_RUNALL_START_ENABLED"] == "true"


def test_yaml_fallback_strips_unresolved_templates():
    """手动 YAML 未解析模板：${...} 应从 hostname/external_url 剥离。"""
    with tempfile.TemporaryDirectory() as td:
        tmp = Path(td)
        main = _write_conf(tmp, _conf())
        out = lgc.resolve("", main)
    assert out["GITLAB_EXTERNAL_HOST"] == "0.0.0.0"
    assert out["GITLAB_EXTERNAL_URL"] == "http://0.0.0.0:8012"


def test_conf_read_keeps_resolved_hostname():
    """conf-read 已解析模板：external_url 直接用 resolved publicUrl。"""
    resolved = _conf()
    resolved["allowedHost"] = "https://gitlab.daydaymoney.com"
    resolved["publicUrl"] = "https://gitlab.daydaymoney.com"
    out = lgc.resolve(json.dumps(resolved), Path("/nope"))
    assert out["GITLAB_EXTERNAL_HOST"] == "gitlab.daydaymoney.com"
    assert out["GITLAB_EXTERNAL_URL"] == "https://gitlab.daydaymoney.com"


def test_resource_defaults_on_missing_keys():
    """conf-read JSON 缺资源键时落到默认值。"""
    resolved = {"host": "localhost", "port": 8012, "sshPort": 22}
    out = lgc.resolve(json.dumps(resolved), Path("/nope"))
    assert out["GITLAB_MEM_LIMIT"] == "6g"
    assert out["GITLAB_SHM_SIZE"] == "512m"
    assert out["GITLAB_PUMA_WORKERS"] == 2
    assert out["GITLAB_IMAGE"] == "gitlab/gitlab-ce:19.2.4-ce.0"


def test_traffic_gate_keys_from_conf():
    resolved = _conf()
    resolved["regionSlug"] = "tencent-sh-1"
    resolved["trafficGateTaskBillBase"] = "http://host.docker.internal:8004"
    out = lgc.resolve(json.dumps(resolved), Path("/nope"))
    assert out["TRAE_GITLAB_REGION"] == "tencent-sh-1"
    assert out["TRAE_TASKBILL_BASE"] == "http://host.docker.internal:8004"
    assert out["TRAE_GITLAB_PUBLIC_HOST"] == out["GITLAB_EXTERNAL_HOST"]


def test_image_tag_from_conf():
    resolved = _conf()
    resolved["imageTag"] = "19.0.8-ce.0"
    out = lgc.resolve(json.dumps(resolved), Path("/nope"))
    assert out["GITLAB_IMAGE"] == "gitlab/gitlab-ce:19.0.8-ce.0"


def test_runall_start_enabled_defaults_false():
    """本机 GitLab 未写 runAllStartEnabled 时禁止被 runAll / run.sh start 拉起。"""
    out = lgc.resolve(json.dumps(_conf()), Path("/nope"))
    assert out["GITLAB_RUNALL_START_ENABLED"] == "false"
    missing = {"host": "localhost", "port": 8012, "sshPort": 22}
    out = lgc.resolve(json.dumps(missing), Path("/nope"))
    assert out["GITLAB_RUNALL_START_ENABLED"] == "false"


def test_runall_start_enabled_true_from_conf_read_json():
    resolved = _conf()
    resolved["runAllStartEnabled"] = True
    out = lgc.resolve(json.dumps(resolved), Path("/nope"))
    assert out["GITLAB_RUNALL_START_ENABLED"] == "true"


def test_runall_start_enabled_conf_local_overrides_main():
    """本机闸门写在 conf-local，不读 config.local.yaml。"""
    with tempfile.TemporaryDirectory() as td:
        tmp = Path(td)
        main = tmp / "conf" / "infra" / "git-service" / "config.yaml"
        main.parent.mkdir(parents=True)
        main.write_text(yaml.safe_dump(_conf() | {"runAllStartEnabled": False}), encoding="utf-8")
        overlay = tmp / "conf-local" / "infra" / "git-service" / "config.yaml"
        overlay.parent.mkdir(parents=True)
        overlay.write_text("runAllStartEnabled: true\n", encoding="utf-8")
        local = main.parent / "config.local.yaml"
        local.write_text("runAllStartEnabled: false\n", encoding="utf-8")
        out = lgc.resolve("", main)
    assert out["GITLAB_RUNALL_START_ENABLED"] == "true"


def test_committed_git_service_yaml_start_disabled_when_present():
    """monorepo 提交库默认必须是 false；独立 clone gitService 时跳过。"""
    conf = Path(__file__).resolve().parents[2] / "conf/infra/git-service/config.yaml"
    if not conf.is_file():
        return
    data = yaml.safe_load(conf.read_text(encoding="utf-8"))
    assert data.get("runAllStartEnabled") is False, (
        "conf/infra/git-service/config.yaml runAllStartEnabled must be false "
        "(opt-in via conf-local/infra/git-service/config.yaml); see ADR-0047"
    )


def test_yaml_exports_oidc_and_taskbill_secrets_from_conf_local():
    with tempfile.TemporaryDirectory() as td:
        tmp = Path(td)
        main = tmp / "conf" / "infra" / "git-service" / "config.yaml"
        main.parent.mkdir(parents=True)
        main.write_text(
            yaml.safe_dump(_conf() | {"oidcClientSecret": "", "trafficGateInternalSecret": ""}),
            encoding="utf-8",
        )
        overlay = tmp / "conf-local" / "infra" / "git-service" / "config.yaml"
        overlay.parent.mkdir(parents=True)
        overlay.write_text(
            "oidcClientSecret: overlay-oidc\ntrafficGateInternalSecret: overlay-bill\n",
            encoding="utf-8",
        )
        stale = main.parent / "config.local.yaml"
        stale.write_text("oidcClientSecret: stale\n", encoding="utf-8")
        out = lgc.resolve("", main)
    assert out["GITLAB_OIDC_CLIENT_SECRET"] == "overlay-oidc"
    assert out["TRAE_TASKBILL_INTERNAL_SECRET"] == "overlay-bill"


def test_yaml_merged_deep_merges_nested_keys():
    """OPT-20260901-012: conf-local nested dict 深合并，不得用 dict.update 顶掉兄弟键。"""
    with tempfile.TemporaryDirectory() as td:
        tmp = Path(td)
        main = tmp / "conf" / "infra" / "git-service" / "config.yaml"
        main.parent.mkdir(parents=True)
        main.write_text(
            "oidc:\n  client_id: gitlab-git-service\n  enabled: false\n",
            encoding="utf-8",
        )
        overlay = tmp / "conf-local" / "infra" / "git-service" / "config.yaml"
        overlay.parent.mkdir(parents=True)
        overlay.write_text("oidc:\n  client_secret: overlay-oidc\n", encoding="utf-8")
        merged = lgc._yaml_merged(main)
    assert merged["oidc"]["client_id"] == "gitlab-git-service"
    assert merged["oidc"]["enabled"] is False
    assert merged["oidc"]["client_secret"] == "overlay-oidc"


def test_conf_read_json_secret_wins_over_empty_yaml():
    resolved = _conf() | {"oidcClientSecret": "from-json", "trafficGateInternalSecret": "bill-json"}
    out = lgc.resolve(json.dumps(resolved), Path("/nope"))
    assert out["GITLAB_OIDC_CLIENT_SECRET"] == "from-json"
    assert out["TRAE_TASKBILL_INTERNAL_SECRET"] == "bill-json"


def test_tracked_gitservice_and_conf_have_no_local_secret_literals():
    needle = "do-not-use-in-prod"
    git_root = Path(__file__).resolve().parents[1]
    hits = []
    for folder in (git_root, git_root.parent / "conf"):
        if not folder.is_dir():
            continue
        for path in folder.rglob("*"):
            if not path.is_file() or ".git" in path.parts:
                continue
            if path.name.startswith("test_") or path.name.endswith("_test.py"):
                continue
            try:
                text = path.read_text(encoding="utf-8")
            except (OSError, UnicodeDecodeError):
                continue
            if needle in text:
                hits.append(str(path))
    assert hits == [], hits


if __name__ == "__main__":
    tests = [v for k, v in sorted(globals().items()) if k.startswith("test_")]
    for t in tests:
        t()
        print(f"PASS {t.__name__}")
    print(f"ok ({len(tests)} tests)")
