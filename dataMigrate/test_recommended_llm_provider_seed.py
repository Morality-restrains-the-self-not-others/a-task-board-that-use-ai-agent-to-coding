#!/usr/bin/env python3
"""Default recommended LLM catalog after all taskCloudService migrations.

Business rule: the seeded list must not recommend OpenAI or Anthropic,
and must include Xiaomi MiMo (小米 MiMo).
"""
from __future__ import annotations

import re
from pathlib import Path

MIG_DIR = Path(__file__).resolve().parent / "taskCloudService"

INSERT_BLOCK = re.compile(
    r"INSERT(?:\s+IGNORE)?\s+INTO\s+cloud_recommended_llm_providers\s*\([^)]+\)\s*VALUES\s*(.+?);",
    re.IGNORECASE | re.DOTALL,
)
ROW_TUPLE = re.compile(r"\('([^']+)',\s*'((?:\\'|[^'])*)'")
DELETE_WHERE = re.compile(
    r"DELETE\s+FROM\s+cloud_recommended_llm_providers\s+WHERE\s+(.+?);",
    re.IGNORECASE | re.DOTALL,
)


def effective_provider_names() -> dict[str, str]:
    """Apply INSERT/DELETE against cloud_recommended_llm_providers in file order."""
    rows: dict[str, str] = {}
    for path in sorted(MIG_DIR.glob("*.sql")):
        sql = path.read_text(encoding="utf-8")
        events: list[tuple[int, str, str]] = []
        for match in DELETE_WHERE.finditer(sql):
            events.append((match.start(), "delete", match.group(1)))
        for match in INSERT_BLOCK.finditer(sql):
            events.append((match.start(), "insert", match.group(1)))
        events.sort(key=lambda item: item[0])
        for _, kind, payload in events:
            if kind == "delete":
                drop_ids: set[str] = set()
                drop_names: set[str] = set()
                for group in re.findall(r"id\s+IN\s*\(([^)]+)\)", payload, re.IGNORECASE):
                    drop_ids.update(re.findall(r"'([^']+)'", group))
                for group in re.findall(r"name\s+IN\s*\(([^)]+)\)", payload, re.IGNORECASE):
                    drop_names.update(re.findall(r"'([^']+)'", group))
                rows = {
                    rid: name
                    for rid, name in rows.items()
                    if rid not in drop_ids and name not in drop_names
                }
                continue
            for row in ROW_TUPLE.finditer(payload):
                rows[row.group(1)] = row.group(2)
    return rows


def test_default_catalog_drops_openai_and_anthropic():
    names = set(effective_provider_names().values())
    assert "OpenAI" not in names, names
    assert "Anthropic" not in names, names


def test_default_catalog_includes_xiaomi_mimo():
    names = list(effective_provider_names().values())
    assert any(
        "mimo" in name.lower() or "小米" in name for name in names
    ), names


def test_default_catalog_keeps_core_domestic_providers():
    names = set(effective_provider_names().values())
    assert "DeepSeek" in names, names
    assert any("Qwen" in name or "Tongyi" in name for name in names), names


if __name__ == "__main__":
    test_default_catalog_drops_openai_and_anthropic()
    test_default_catalog_includes_xiaomi_mimo()
    test_default_catalog_keeps_core_domestic_providers()
    print("ok")
