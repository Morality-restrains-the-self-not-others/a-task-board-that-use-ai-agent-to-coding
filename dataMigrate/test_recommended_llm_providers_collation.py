#!/usr/bin/env python3
"""cloud_recommended_llm_providers 字符集排序规则一致性（OPT-20260816-015）。

约定：所有建表/迁移统一 `COLLATE=utf8mb4_unicode_ci`，避免存量库与新建库
collation 漂移导致 JOIN/比较冲突。
"""
from __future__ import annotations

import re
from pathlib import Path

MIG_DIR = Path(__file__).resolve().parent / "taskCloudService"

CONVERT_RE = re.compile(
    r"ALTER\s+TABLE\s+cloud_recommended_llm_providers\s+CONVERT\s+TO\s+"
    r"CHARACTER\s+SET\s+utf8mb4\s+COLLATE\s+utf8mb4_unicode_ci",
    re.IGNORECASE,
)


def _sql(name: str) -> str:
    return (MIG_DIR / name).read_text(encoding="utf-8")


def test_009_declares_unicode_collation():
    """建表即声明 COLLATE=utf8mb4_unicode_ci，新库不再漂移。"""
    sql = _sql("009_recommended_llm_providers.sql")
    assert "COLLATE=utf8mb4_unicode_ci" in sql, "009 CREATE TABLE 应声明 COLLATE=utf8mb4_unicode_ci"


def test_022_converts_table_to_unicode_collation():
    """存在 022 迁移，把存量库统一到 utf8mb4_unicode_ci。"""
    mig = next(MIG_DIR.glob("022_*.sql"))
    sql = mig.read_text(encoding="utf-8")
    assert CONVERT_RE.search(sql), "022 迁移应包含 CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"
