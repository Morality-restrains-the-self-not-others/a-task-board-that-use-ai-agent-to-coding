#!/usr/bin/env python3
"""One-shot migration: legacy task2app/conf/port_config.json -> <monorepo>/conf/<app>/.

If port_config.json is already removed, conf/ is authoritative — script exits 0 with a note.

DEPRECATED (OPT-20260806-057): conf/core/django 目录已退役，本脚本的 django 分支为历史遗留，
仅当存在旧 port_config.json 时才可能触发；新环境不会使用。
"""
from __future__ import annotations

import json
import re
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from conf_lib import dump_yaml, repo_root

JSON_KEY_TO_APP = {
    "django": "core/django",
    "vue": "frontend/vue",
    "aiProvider": "ai/ai-provider",
    "gitService": "infra/git-service",
    "dockerInfra": "infra/docker-infra",
    "mock_trae_worker": "mock/mock-trae-worker",
        "taskSSE": "gateway/task-sse",
    "taskAgentSupport": "ai/task-agent-support",
    "taskAIEndPoint": "ai/task-ai-endpoint",
    "taskContainerGateway": "gateway/task-container-gateway",
    "relayToTrae": "infra/relay-to-trae",
    "taskAuth": "auth/task-auth",
    "taskBill": "billing/task-bill",
    "stripe": "billing/stripe",
}


def _slug_provider_key(url_key: str) -> str:
    s = url_key.lower().replace("://", "-").replace("/", "-").replace(":", "-")
    s = re.sub(r"[^a-z0-9-]+", "-", s)
    return re.sub(r"-+", "-", s).strip("-")


def migrate() -> None:
    root = repo_root()
    src = root / "task2app" / "conf" / "port_config.json"
    if not src.is_file():
        marker = root / "conf" / "core" / "django" / "config.yaml"
        if marker.is_file():
            print(f"skip: {src} gone; using existing {marker.parent.parent.parent}/")
            return
        raise SystemExit(f"missing {src} and conf/core/django/config.yaml")
    data = json.loads(src.read_text(encoding="utf-8"))
    conf = root / "conf"

    # django
    django_body = dict(data.get("django") or {})
    if data.get("task2appSsoJwtSecret"):
        django_body["ssoJwtSecret"] = data["task2appSsoJwtSecret"]
    if data.get("ssh_login_allowed_addresses"):
        django_body["ssh_login_allowed_addresses"] = data["ssh_login_allowed_addresses"]
    dump_yaml(conf / "core" / "django" / "config.yaml", django_body)

    # test phones
    test_src = root / "conf" / "port_config.test.json"
    if not test_src.is_file():
        test_src = root / "task2app" / "conf" / "port_config.test.json"
    if test_src.is_file():
        tdata = json.loads(test_src.read_text(encoding="utf-8"))
        phones = (tdata.get("django") or {}).get("test_phone_numbers")
        if phones:
            dump_yaml(conf / "core" / "django" / "config.test.yaml", {"test_phone_numbers": phones})

    for key, app in JSON_KEY_TO_APP.items():
        if key == "django":
            continue
        block = data.get(key)
        if block is None:
            continue
        dump_yaml(conf / app / "config.yaml", block)

    # git-oauth providers
    git_oauth = data.get("gitOauth")
    if isinstance(git_oauth, dict):
        prov_dir = conf / "git-oauth" / "providers"
        prov_dir.mkdir(parents=True, exist_ok=True)
        for website_key, prov in git_oauth.items():
            if not isinstance(prov, dict):
                continue
            dump_yaml(prov_dir / f"{_slug_provider_key(website_key)}.yaml", prov)

    # domain-events
    de = data.get("domainEvents")
    if isinstance(de, dict):
        global_de = {k: v for k, v in de.items() if k != "consumers"}
        dump_yaml(conf / "domain-events" / "config.yaml", global_de)
        consumers = de.get("consumers") or {}
        if isinstance(consumers, dict):
            for event_slug, event_block in consumers.items():
                if not isinstance(event_block, dict):
                    continue
                dump_yaml(conf / "domain-events" / event_slug / "config.yaml", event_block)

    print(f"migrated -> {conf}")


if __name__ == "__main__":
    migrate()
