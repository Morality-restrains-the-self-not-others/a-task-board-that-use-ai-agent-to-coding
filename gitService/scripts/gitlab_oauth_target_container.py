#!/usr/bin/env python3
"""Map a GitLab OAuth provider website to the gitService docker container.

Multi-region GitLab (ADR-0014): each conf/infra/git-service*/ instance has its
own CE container. Doorkeeper Application rows must be created on the instance
that owns target.website — never on the default gitlab container by accident.
"""
from __future__ import annotations

import argparse
import os
import re
import sys
from pathlib import Path
from urllib.parse import urlparse

try:
    import yaml
except ImportError:
    yaml = None

_PLACEHOLDER_RE = re.compile(r"\$\{(scheme|baseDomain|subdomains\.[A-Za-z][A-Za-z0-9]*)\}")
_ENV_DEFAULT_RE = re.compile(r"\$\{(\w+):-([^}]*)\}")


def hostname_of(raw: str) -> str:
    s = (raw or "").strip()
    if not s:
        return ""
    if "://" not in s:
        s = "https://" + s
    host = (urlparse(s).hostname or "").strip().lower()
    return host


def resolve_env_default(value: str) -> str:
    def _repl(m: re.Match[str]) -> str:
        return os.environ.get(m.group(1), m.group(2) or "")

    return _ENV_DEFAULT_RE.sub(_repl, str(value))


def _runall_scripts(repo_root: Path) -> Path:
    direct = repo_root / "runAll" / "scripts"
    if (direct / "conf_local.py").is_file():
        return direct
    for parent in Path(__file__).resolve().parents:
        cand = parent / "runAll" / "scripts" / "conf_local.py"
        if cand.is_file():
            return cand.parent
    return direct


def _overlay_yaml(repo_root: Path, path: Path) -> dict:
    scripts = str(_runall_scripts(repo_root))
    if scripts not in sys.path:
        sys.path.insert(0, scripts)
    from conf_local import overlay_conf_file

    loaded = overlay_conf_file(path)
    return loaded if isinstance(loaded, dict) else {}


def load_domain_map(repo_root: Path) -> dict[str, str]:
    if yaml is None:
        return {}
    candidate = repo_root / "conf" / "base.yaml"
    if not candidate.is_file():
        return {}
    raw = _overlay_yaml(repo_root, candidate)
    scheme = os.environ.get("PUBLIC_SCHEME") or resolve_env_default(str(raw.get("scheme", "https")))
    scheme = scheme.strip().lower().rstrip(":/") or "https"
    base_domain = os.environ.get("BASE_DOMAIN") or resolve_env_default(str(raw.get("baseDomain", "")))
    if not base_domain:
        return {}
    domain_map = {"scheme": scheme, "baseDomain": base_domain}
    subdomains = raw.get("subdomains") or {}
    if isinstance(subdomains, dict):
        for key, template in subdomains.items():
            val = (
                str(template)
                .replace("${scheme}", scheme)
                .replace("${baseDomain}", base_domain)
            )
            domain_map[f"subdomains.{key}"] = val
    return domain_map


def resolve_template(val: str, domain_map: dict[str, str]) -> str:
    if not isinstance(val, str):
        return str(val or "")
    return _PLACEHOLDER_RE.sub(lambda m: domain_map.get(m.group(1), m.group(0)), val)


def git_service_instances(repo_root: Path, domain_map: dict[str, str] | None = None) -> list[tuple[str, str]]:
    """Return (hostname, container_name) for each conf/infra/git-service* instance."""
    if yaml is None:
        return []
    if domain_map is None:
        domain_map = load_domain_map(repo_root)
    infra = repo_root / "conf" / "infra"
    if not infra.is_dir():
        return []
    out: list[tuple[str, str]] = []
    for d in sorted(infra.glob("git-service*")):
        if not d.is_dir():
            continue
        cfg_path = d / "config.yaml"
        if not cfg_path.is_file():
            continue
        data = _overlay_yaml(repo_root, cfg_path)
        public = resolve_template(str(data.get("publicUrl") or data.get("allowedHost") or ""), domain_map)
        host = hostname_of(public)
        if not host:
            continue
        container = str(data.get("containerName") or "").strip() or "gitlab"
        out.append((host, container))
    return out


def resolve_container_for_website(website: str, instances: list[tuple[str, str]]) -> str | None:
    host = hostname_of(website)
    if not host:
        return None
    for h, container in instances:
        if h == host:
            return container
    return None


def resolve_container(website: str, repo_root: Path) -> str | None:
    website = resolve_template(website, load_domain_map(repo_root))
    return resolve_container_for_website(website, git_service_instances(repo_root))


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--website", required=True)
    parser.add_argument("--root", required=True)
    args = parser.parse_args(argv)
    container = resolve_container(args.website, Path(args.root))
    if not container:
        return 1
    print(container)
    return 0


if __name__ == "__main__":
    sys.exit(main())
