#!/usr/bin/env python3
"""gitService 配置解析（元规则 47 SSOT 消费方，OPT-20260812-033）。

读取顺序：runAll/scripts/conf-read.py JSON（已解析 ${...} 模板）优先；
conf-read 不可用时手动 YAML（config.yaml + conf-local 同相对路径）兜底，
此时剥掉未解析模板变量。config.local.yaml 已废弃（ADR-0054），加载器只读
config.yaml 与 conf-local overlay。

用法: python3 load_gitservice_config.py <config_json> <main_yaml>
输出: KEY=VALUE 行（供 run.sh `load_gitservice_config()` 逐一 export）。

单测直接 import 本模块：见 scripts/test_load_gitservice_config.py。
"""
from __future__ import annotations

import json
import os
import sys
from pathlib import Path
from urllib.parse import urlparse

# 资源键默认值（与 conf/infra/git-service/config.yaml 一致；conf 为 SSOT）
DEFAULT_MEM_LIMIT = "6g"
DEFAULT_SHM_SIZE = "512m"
DEFAULT_PUMA_WORKERS = 2
DEFAULT_SIDEKIQ_CONCURRENCY = 5
DEFAULT_MIN_DOCKER_MEMORY_MIB = 6144
DEFAULT_HTTP_PORT = 8012
DEFAULT_SSH_PORT = 2222
DEFAULT_IMAGE_TAG = "19.2.4-ce.0"

# 登录/注册开关默认值（OPT-20260807-034）：默认关闭注册与账密登录，仅 OIDC
DEFAULT_SIGNUP_ENABLED = False
DEFAULT_PASSWORD_AUTH_WEB = False
DEFAULT_PASSWORD_AUTH_GIT = False
# 本机 GitLab 须显式打开后才能被 runAll 面板 / run.sh start 拉起（ADR-0047）
DEFAULT_RUNALL_START_ENABLED = False


def _safe_int(value, default):
    try:
        return int(value)
    except (TypeError, ValueError):
        return default


def _safe_bool(value, default):
    """解析 YAML/JSON 布尔；输出统一为 'true'/'false' 供 compose 插值。"""
    if isinstance(value, bool):
        return value
    if isinstance(value, str):
        v = value.strip().lower()
        if v in {"true", "1", "yes", "on"}:
            return True
        if v in {"false", "0", "no", "off"}:
            return False
    return default


def _bool_str(value):
    return "true" if value else "false"


def _gitlab_image(image_tag) -> str:
    """conf imageTag → compose GITLAB_IMAGE（允许误填完整镜像名）。"""
    tag = str(image_tag or DEFAULT_IMAGE_TAG).strip() or DEFAULT_IMAGE_TAG
    if "/" in tag:
        return tag
    return f"gitlab/gitlab-ce:{tag}"


def _read_yaml(path: Path, yaml_mod):
    if yaml_mod is None or not path.is_file():
        return {}
    try:
        data = yaml_mod.safe_load(path.read_text(encoding="utf-8"))
    except (OSError, yaml_mod.YAMLError):
        return {}
    return data if isinstance(data, dict) else {}


def _expand_infra_host(value: str) -> str:
    """Replace ${INFRA_HOST} from env; leave host.docker.internal intact."""
    raw = (value or "").strip()
    if not raw:
        return ""
    host = (os.environ.get("INFRA_HOST") or os.environ.get("RUNALL_INFRA_HOST") or "").strip()
    if host.startswith("http://"):
        host = host[len("http://"):]
    elif host.startswith("https://"):
        host = host[len("https://"):]
    if host and "${INFRA_HOST}" in raw:
        raw = raw.replace("${INFRA_HOST}", host.split("/")[0].split(":")[0])
    return raw


def _finalize(
    host,
    http_port,
    ssh_port,
    advertise_ssh_port,
    allowed_host_url,
    public_url,
    gitlab_home_conf,
    mem_limit,
    shm_size,
    puma_workers,
    sidekiq_concurrency,
    min_docker_memory_mib,
    signup_enabled,
    password_auth_web,
    password_auth_git,
    strip_template,
    oidc_client_id="",
    container_name="",
    image_tag="",
    region_slug="",
    traffic_gate_base="",
):
    """两分支共用的 hostname/display_host/external_url 计算与 KEY=VALUE 输出。"""
    allowed_host = ""
    if allowed_host_url:
        try:
            allowed_host = (urlparse(allowed_host_url).hostname or "").strip()
        except ValueError:
            allowed_host = ""
    if strip_template and allowed_host and "${" in allowed_host:
        allowed_host = ""
    hostname = allowed_host or host
    display_host = hostname if host in {"localhost", "::1", "127.0.0.1", "0.0.0.0"} else host
    external_url = (public_url or allowed_host_url or "").rstrip("/")
    if (not external_url) or (strip_template and "${" in external_url):
        external_url = f"http://{hostname}:{http_port}"
    return {
        "GITLAB_EXTERNAL_HOST": hostname,
        "GITLAB_HTTP_PORT": http_port,
        "GITLAB_SSH_PORT": ssh_port,
        "GITLAB_SSH_ADVERTISE_PORT": advertise_ssh_port,
        "GITLAB_ALLOWED_HOST_URL": allowed_host_url,
        "GITLAB_HOSTNAME": hostname,
        "GITLAB_DISPLAY_HOST": display_host,
        "GITLAB_EXTERNAL_URL": external_url,
        "GITLAB_HOME_FROM_CONF": gitlab_home_conf,
        "GITLAB_MEM_LIMIT": mem_limit,
        "GITLAB_SHM_SIZE": shm_size,
        "GITLAB_PUMA_WORKERS": puma_workers,
        "GITLAB_SIDEKIQ_CONCURRENCY": sidekiq_concurrency,
        "GITLAB_MIN_DOCKER_MEMORY_MIB": min_docker_memory_mib,
        "GITLAB_SIGNUP_ENABLED": _bool_str(signup_enabled),
        "GITLAB_PASSWORD_AUTH_WEB": _bool_str(password_auth_web),
        "GITLAB_PASSWORD_AUTH_GIT": _bool_str(password_auth_git),
        "GITLAB_OIDC_CLIENT_ID": (oidc_client_id or "gitlab-git-service").strip() or "gitlab-git-service",
        "GITLAB_CONTAINER_NAME": (container_name or "gitlab").strip() or "gitlab",
        "GITLAB_IMAGE": _gitlab_image(image_tag),
        "TRAE_GITLAB_REGION": (region_slug or "").strip(),
        "TRAE_TASKBILL_BASE": _expand_infra_host(traffic_gate_base) or "http://host.docker.internal:8004",
        "TRAE_GITLAB_PUBLIC_HOST": hostname,
        "TRAE_GITLAB_INTRANET_HOSTS": "",
    }


def _conf_local_overlay(main_path: Path) -> Path:
    """conf/<rel> → <repo>/conf-local/<rel> (ADR-0054)."""
    resolved = main_path.resolve()
    parts = resolved.parts
    try:
        idx = parts.index("conf")
    except ValueError:
        return Path()
    root = Path(*parts[:idx]) if idx > 0 else Path("/")
    rel = Path(*parts[idx + 1 :])
    return root / "conf-local" / rel


def _deep_merge(base: dict, overlay: dict) -> dict:
    """递归深合并（与 runAll/scripts/conf_lib.py deep_merge 语义一致，ADR-0054）。

    dict.update 是浅合并：conf-local 里 nested dict 会把 skeleton 的兄弟键整体
    顶掉。这里对 dict 递归合并，标量仍以 overlay 为准。
    """
    out = dict(base)
    for k, v in overlay.items():
        if k in out and isinstance(out[k], dict) and isinstance(v, dict):
            out[k] = _deep_merge(out[k], v)
        else:
            out[k] = v
    return out


def _yaml_merged(main_path: Path) -> dict:
    try:
        import yaml as yaml_mod
    except ImportError:
        return {}
    merged = _read_yaml(main_path, yaml_mod)
    overlay = _conf_local_overlay(main_path)
    if overlay != Path():
        merged = _deep_merge(merged, _read_yaml(overlay, yaml_mod))
    return merged


def _first_secret(*vals) -> str:
    for val in vals:
        text = str(val or "").strip()
        if text:
            return text
    return ""


def _attach_secrets(out: dict, resolved: dict, main_path: Path) -> None:
    yaml_cfg = _yaml_merged(main_path)
    out["GITLAB_OIDC_CLIENT_SECRET"] = _first_secret(
        (resolved or {}).get("oidcClientSecret"),
        yaml_cfg.get("oidcClientSecret"),
    )
    out["TRAE_TASKBILL_INTERNAL_SECRET"] = _first_secret(
        (resolved or {}).get("trafficGateInternalSecret"),
        yaml_cfg.get("trafficGateInternalSecret"),
    )


def resolve(config_json: str, main_path: Path) -> dict:
    """解析 gitService 配置并返回 KEY=VALUE 字典（run.sh export / 单测直接使用）。"""
    resolved = {}
    if config_json:
        try:
            resolved = json.loads(config_json)
        except (json.JSONDecodeError, TypeError):
            resolved = {}

    if resolved:
        # conf-read.py 已解析模板：不做 ${...} 剥离。
        ssh_port = _safe_int(resolved.get("sshPort"), DEFAULT_SSH_PORT)
        out = _finalize(
            host=str(resolved.get("host") or "localhost").strip() or "localhost",
            http_port=_safe_int(resolved.get("port"), DEFAULT_HTTP_PORT),
            ssh_port=ssh_port,
            advertise_ssh_port=_safe_int(resolved.get("advertiseSshPort"), 0) or ssh_port,
            allowed_host_url=str(resolved.get("allowedHost") or "").strip(),
            public_url=str(resolved.get("publicUrl") or "").strip(),
            gitlab_home_conf=str(resolved.get("gitlabHome") or "").strip(),
            mem_limit=str(resolved.get("memLimit") or DEFAULT_MEM_LIMIT).strip() or DEFAULT_MEM_LIMIT,
            shm_size=str(resolved.get("shmSize") or DEFAULT_SHM_SIZE).strip() or DEFAULT_SHM_SIZE,
            puma_workers=_safe_int(resolved.get("pumaWorkers"), DEFAULT_PUMA_WORKERS),
            sidekiq_concurrency=_safe_int(resolved.get("sidekiqConcurrency"), DEFAULT_SIDEKIQ_CONCURRENCY),
            min_docker_memory_mib=_safe_int(resolved.get("minDockerMemoryMib"), DEFAULT_MIN_DOCKER_MEMORY_MIB),
            signup_enabled=_safe_bool(resolved.get("signupEnabled"), DEFAULT_SIGNUP_ENABLED),
            password_auth_web=_safe_bool(resolved.get("passwordAuthWeb"), DEFAULT_PASSWORD_AUTH_WEB),
            password_auth_git=_safe_bool(resolved.get("passwordAuthGit"), DEFAULT_PASSWORD_AUTH_GIT),
            strip_template=False,
            oidc_client_id=str(resolved.get("oidcClientId") or "").strip(),
            container_name=str(resolved.get("containerName") or "").strip(),
            image_tag=str(resolved.get("imageTag") or "").strip(),
            region_slug=str(resolved.get("regionSlug") or "").strip(),
            traffic_gate_base=str(resolved.get("trafficGateTaskBillBase") or "").strip(),
        )
        out["GITLAB_RUNALL_START_ENABLED"] = _bool_str(
            _safe_bool(resolved.get("runAllStartEnabled"), DEFAULT_RUNALL_START_ENABLED)
        )
        _attach_secrets(out, resolved, main_path)
        return out

    # Fallback：手动 YAML 合并（config.yaml + conf-local），剥未解析模板。
    merged = _yaml_merged(main_path)
    gs = merged
    ssh_port = _safe_int(gs.get("sshPort"), DEFAULT_SSH_PORT)
    out = _finalize(
        host=str(gs.get("host") or "localhost").strip() or "localhost",
        http_port=_safe_int(gs.get("port"), DEFAULT_HTTP_PORT),
        ssh_port=ssh_port,
        advertise_ssh_port=_safe_int(gs.get("advertiseSshPort"), 0) or ssh_port,
        allowed_host_url=str(gs.get("allowedHost") or "").strip(),
        public_url=str(gs.get("publicUrl") or "").strip(),
        gitlab_home_conf=str(gs.get("gitlabHome") or "").strip(),
        mem_limit=str(gs.get("memLimit") or DEFAULT_MEM_LIMIT).strip() or DEFAULT_MEM_LIMIT,
        shm_size=str(gs.get("shmSize") or DEFAULT_SHM_SIZE).strip() or DEFAULT_SHM_SIZE,
        puma_workers=_safe_int(gs.get("pumaWorkers"), DEFAULT_PUMA_WORKERS),
        sidekiq_concurrency=_safe_int(gs.get("sidekiqConcurrency"), DEFAULT_SIDEKIQ_CONCURRENCY),
        min_docker_memory_mib=_safe_int(gs.get("minDockerMemoryMib"), DEFAULT_MIN_DOCKER_MEMORY_MIB),
        signup_enabled=_safe_bool(gs.get("signupEnabled"), DEFAULT_SIGNUP_ENABLED),
        password_auth_web=_safe_bool(gs.get("passwordAuthWeb"), DEFAULT_PASSWORD_AUTH_WEB),
        password_auth_git=_safe_bool(gs.get("passwordAuthGit"), DEFAULT_PASSWORD_AUTH_GIT),
        strip_template=True,
        oidc_client_id=str(gs.get("oidcClientId") or "").strip(),
        container_name=str(gs.get("containerName") or "").strip(),
        image_tag=str(gs.get("imageTag") or "").strip(),
        region_slug=str(gs.get("regionSlug") or "").strip(),
        traffic_gate_base=str(gs.get("trafficGateTaskBillBase") or "").strip(),
    )
    out["GITLAB_RUNALL_START_ENABLED"] = _bool_str(
        _safe_bool(gs.get("runAllStartEnabled"), DEFAULT_RUNALL_START_ENABLED)
    )
    _attach_secrets(out, gs, main_path)
    return out


def main() -> int:
    if len(sys.argv) < 3:
        print(
            "usage: load_gitservice_config.py <config_json> <main_yaml>",
            file=sys.stderr,
        )
        return 2
    config_json = sys.argv[1] if len(sys.argv) > 1 else ""
    out = resolve(config_json, Path(sys.argv[2]))
    for key, value in out.items():
        print(f"{key}={value}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
