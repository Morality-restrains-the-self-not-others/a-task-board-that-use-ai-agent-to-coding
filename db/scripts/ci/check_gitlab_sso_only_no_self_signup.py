#!/usr/bin/env python3
"""GitLab 禁止自行注册、仅允许 SSO（元规则 50 / ADR-0016）。

平台 GitLab（含全部 conf/infra/git-service* 区域实例）默认关闭公开注册与账密登录，
仅保留 taskAuth OIDC OmniAuth。本门禁检查已提交 SSOT 与 compose/loader 默认值，
防止 Agent 以「登录坏了」为由打开 signup / password auth。

用法:
  python3 db/scripts/ci/check_gitlab_sso_only_no_self_signup.py
"""
from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]

AUTH_FALSE_KEYS = ("signupEnabled", "passwordAuthWeb", "passwordAuthGit")

COMPOSE_FALSE_ENV = (
    "GITLAB_SIGNUP_ENABLED",
    "GITLAB_PASSWORD_AUTH_WEB",
    "GITLAB_PASSWORD_AUTH_GIT",
)

LOADER_DEFAULTS = (
    "DEFAULT_SIGNUP_ENABLED = False",
    "DEFAULT_PASSWORD_AUTH_WEB = False",
    "DEFAULT_PASSWORD_AUTH_GIT = False",
)

META_FILES = (
    ".ai/01_project_constraints/55_gitlab_sso_only_no_self_signup.md",
    ".cursor/rules/gitlab-sso-only-no-self-signup.mdc",
    "docs/adr/0016-gitlab-sso-only-no-self-signup.md",
)

OMNIAUTH_ENABLED_RE = re.compile(
    r"gitlab_rails\s*\[\s*['\"]omniauth_enabled['\"]\s*\]\s*=\s*true\b"
)
OMNIAUTH_SSO_RE = re.compile(
    r"gitlab_rails\s*\[\s*['\"]omniauth_allow_single_sign_on['\"]\s*\]"
)
OIDC_RE = re.compile(r"openid_connect")
BLOCK_AUTO_FALSE_RE = re.compile(
    r"gitlab_rails\s*\[\s*['\"]omniauth_block_auto_created_users['\"]\s*\]\s*=\s*false\b"
)


def is_explicit_false(value) -> bool:
    if isinstance(value, str):
        return value.strip().lower() in {"false", "0", "no", "off"}
    return value is False


def check_git_service_config(path: Path, data: dict) -> list[str]:
    rel = str(path)
    if not isinstance(data, dict):
        return [f"{rel}: config.yaml 必须是 mapping"]
    hits = []
    for key in AUTH_FALSE_KEYS:
        if key not in data:
            hits.append(f"{rel}: missing {key} (must be explicit false)")
            continue
        if not is_explicit_false(data[key]):
            hits.append(f"{rel}: {key}={data[key]!r} must be false (SSO-only, no self-signup)")
    return hits


def check_compose_text(rel: str, text: str) -> list[str]:
    hits = []
    for env in COMPOSE_FALSE_ENV:
        false_token = f"${{{env}:-false}}"
        true_token = f"${{{env}:-true}}"
        if true_token in text:
            hits.append(f"{rel}: {env} fallback must be false, found :-true")
        if false_token not in text:
            hits.append(f"{rel}: missing {false_token} (compose must default auth switches off)")
    if not OMNIAUTH_ENABLED_RE.search(text):
        hits.append(f"{rel}: gitlab_rails['omniauth_enabled'] must be true")
    if not OMNIAUTH_SSO_RE.search(text):
        hits.append(f"{rel}: missing omniauth_allow_single_sign_on")
    if not OIDC_RE.search(text):
        hits.append(f"{rel}: OmniAuth must include openid_connect (taskAuth SSO)")
    if not BLOCK_AUTO_FALSE_RE.search(text):
        hits.append(
            f"{rel}: omniauth_block_auto_created_users must be false "
            "(first SSO login may provision GitLab user)"
        )
    if re.search(r"gitlab_rails\s*\[\s*['\"]signup_enabled['\"]\s*\]\s*=\s*true\b", text):
        hits.append(f"{rel}: hardcoded signup_enabled = true is forbidden")
    return hits


def check_loader_text(rel: str, text: str) -> list[str]:
    hits = []
    for needle in LOADER_DEFAULTS:
        if needle not in text:
            hits.append(f"{rel}: missing {needle}")
    return hits


def check_apply_auth_policy_text(rel: str, text: str) -> list[str]:
    if "signup_enabled" not in text:
        return [f"{rel}: apply_auth_policy.sh must write signup_enabled"]
    return []


def check_run_sh_text(rel: str, text: str, helper_text: str = "") -> list[str]:
    blob = f"{text}\n{helper_text}"
    if "apply_auth_policy.sh" not in blob:
        return [f"{rel}: run.sh must invoke apply_auth_policy.sh (DB-level auth policy)"]
    return []


def _load_yaml(path: Path):
    try:
        import yaml
    except ImportError:
        return None, f"{path}: PyYAML required"
    try:
        data = yaml.safe_load(path.read_text(encoding="utf-8"))
    except (OSError, yaml.YAMLError) as exc:
        return None, f"{path}: YAML parse error: {exc}"
    return data, None


def iter_git_service_configs(root: Path):
    infra = root / "conf" / "infra"
    if not infra.is_dir():
        return
    yield from sorted(infra.glob("git-service*/config.yaml"))


def collect_violations(root: Path, *, check_meta: bool = True) -> list[str]:
    hits: list[str] = []
    configs = list(iter_git_service_configs(root))
    if not configs:
        hits.append("conf/infra/git-service*/config.yaml: no GitLab region SSOT found")
    for path in configs:
        data, err = _load_yaml(path)
        if err:
            hits.append(err)
            continue
        try:
            rel = path.relative_to(root)
        except ValueError:
            rel = path
        hits.extend(check_git_service_config(rel, data or {}))

    compose = root / "gitService" / "docker-compose.yml"
    if not compose.is_file():
        hits.append("gitService/docker-compose.yml: missing")
    else:
        hits.extend(
            check_compose_text(
                "gitService/docker-compose.yml", compose.read_text(encoding="utf-8")
            )
        )
    regional = root / "gitService" / "docker-compose.tencent-sh-1.yml"
    if regional.is_file():
        hits.extend(
            check_compose_text(
                "gitService/docker-compose.tencent-sh-1.yml",
                regional.read_text(encoding="utf-8"),
            )
        )

    loader = root / "gitService" / "scripts" / "load_gitservice_config.py"
    if not loader.is_file():
        hits.append("gitService/scripts/load_gitservice_config.py: missing")
    else:
        hits.extend(
            check_loader_text(
                "gitService/scripts/load_gitservice_config.py",
                loader.read_text(encoding="utf-8"),
            )
        )

    apply_sh = root / "gitService" / "scripts" / "apply_auth_policy.sh"
    if not apply_sh.is_file():
        hits.append("gitService/scripts/apply_auth_policy.sh: missing")
    else:
        hits.extend(
            check_apply_auth_policy_text(
                "gitService/scripts/apply_auth_policy.sh",
                apply_sh.read_text(encoding="utf-8"),
            )
        )

    run_sh = root / "gitService" / "run.sh"
    if not run_sh.is_file():
        hits.append("gitService/run.sh: missing")
    else:
        helper = root / "gitService" / "scripts" / "ensure_gitlab_container.sh"
        helper_text = helper.read_text(encoding="utf-8") if helper.is_file() else ""
        hits.extend(
            check_run_sh_text(
                "gitService/run.sh",
                run_sh.read_text(encoding="utf-8"),
                helper_text,
            )
        )

    if check_meta:
        for rel in META_FILES:
            if not (root / rel).is_file():
                hits.append(f"{rel}: missing meta file")
    return hits


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=ROOT)
    args = parser.parse_args(argv)
    root = args.root.resolve()
    hits = collect_violations(root, check_meta=True)
    if hits:
        print("VIOLATION (rule 55_gitlab_sso_only_no_self_signup.md / ADR-0016):")
        for h in hits:
            print(f"  - {h}")
        return 1
    print("ok: GitLab SSO-only / no self-signup")
    return 0


if __name__ == "__main__":
    sys.exit(main())
