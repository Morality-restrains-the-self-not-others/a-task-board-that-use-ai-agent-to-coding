#!/usr/bin/env python
"""Seed default privacy policy and license agreements via taskBill Go API.

2026-07-26: Django license_agreement + privacy_policy apps deleted.
Legal documents now managed by taskBill (Go). This script calls taskBill API directly.
2026-07-31: Moved from dataMigrate/saas/ to dataMigrate/taskBill/ (SaaS Django decommissioned).
2026-08-03: Updated to read agreement content from .md files with version 0.9.

幂等：taskBill Go API 的 license_agreements 表有 document_kind + is_active 唯一约束，
重复执行不会创建重复记录。

用法:
    python3 dataMigrate/taskBill/026_seed_default_legal_documents.py

环境变量:
    TASKBILL_BASE_URL — taskBill 服务地址（默认 http://127.0.0.1:8004）
"""

import json
import logging
import os
import sys
import urllib.error
import urllib.request

logger = logging.getLogger(__name__)

TASKBILL_BASE = os.environ.get("TASKBILL_BASE_URL", "http://127.0.0.1:8004").rstrip("/")

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))


def _read_md(filename: str) -> str:
    """Read a markdown file from the script directory."""
    path = os.path.join(SCRIPT_DIR, filename)
    with open(path, "r", encoding="utf-8") as f:
        return f.read()


def _build_privacy_policy():
    return {
        "title": "隐私政策条款",
        "content": _read_md("privacy_policy_v0.9.md"),
        "version": "0.9",
        "is_active": True,
        "is_material_change": False,
    }


def _build_service_agreement():
    return {
        "title": "用户服务协议",
        "content": _read_md("service_agreement_v0.9.md"),
        "version": "0.9",
        "document_kind": "service",
        "is_active": True,
        "is_material_change": False,
    }


def _build_payment_agreement():
    return {
        "title": "支付服务条款协议",
        "content": _read_md("payment_agreement_v0.9.md"),
        "version": "0.9",
        "document_kind": "recharge_cents",
        "is_active": True,
        "is_material_change": False,
    }


def _post_json(path: str, body: dict) -> bool:
    url = f"{TASKBILL_BASE}{path}"
    data = json.dumps(body).encode("utf-8")
    try:
        req = urllib.request.Request(url, data=data, method="POST")
        req.add_header("Content-Type", "application/json")
        with urllib.request.urlopen(req, timeout=15) as resp:
            if resp.status in (200, 201):
                logger.info("taskBill %s → %s", path, resp.status)
                return True
            logger.warning("taskBill %s → %s: %s", path, resp.status, resp.read().decode("utf-8", errors="replace"))
            return False
    except urllib.error.HTTPError as e:
        body = e.read().decode("utf-8", errors="replace")
        logger.warning("taskBill %s → %s: %s", path, e.code, body)
        return False
    except Exception as exc:
        logger.warning("taskBill %s failed: %s", path, exc)
        return False


def _get_json(path: str) -> dict | None:
    url = f"{TASKBILL_BASE}{path}"
    try:
        req = urllib.request.Request(url)
        req.add_header("Accept", "application/json")
        with urllib.request.urlopen(req, timeout=10) as resp:
            if resp.status == 200:
                return json.loads(resp.read().decode("utf-8"))
            return None
    except urllib.error.HTTPError as e:
        if e.code == 404:
            return None
        logger.warning("taskBill GET %s → %s", path, e.code)
        return None
    except Exception as exc:
        logger.warning("taskBill GET %s failed: %s", path, exc)
        return None


def create_default_privacy_policy():
    """检查是否已有 v0.9 隐私条款，若无则创建。"""
    current = _get_json("/api/privacy-policy/public/current/")
    if current and current.get("version") == "0.9":
        print(f"  隐私条款 v0.9 已存在：{current.get('title')}")
        return True

    print("  创建隐私条款 v0.9...")
    if _post_json("/api/system_admin/privacy-policy/", _build_privacy_policy()):
        print("  隐私条款 v0.9 创建成功")
        return True
    print("  隐私条款创建失败（taskBill 可能未启动）")
    return False


def create_default_license_agreement(kind: str, doc: dict):
    """检查指定类型的服务协议是否已存在 v0.9，若无则创建。"""
    current = _get_json(f"/api/license-agreement/public/current/?kind={kind}")
    if current and current.get("version") == "0.9":
        print(f"  服务协议 v0.9 已存在（{kind}）：{current.get('title')}")
        return True

    print(f"  创建服务协议 v0.9（{kind}）...")
    if _post_json("/api/system_admin/license-agreement/", doc):
        print(f"  服务协议 v0.9 创建成功（{kind}）")
        return True
    print(f"  服务协议创建失败（{kind}，taskBill 可能未启动）")
    return False


def create_default_legal_documents():
    """幂等创建默认法律文档。"""
    print("创建默认隐私条款与服务协议 (v0.9)...")
    ok = True
    if not create_default_privacy_policy():
        ok = False
    if not create_default_license_agreement("service", _build_service_agreement()):
        ok = False
    if not create_default_license_agreement("recharge_cents", _build_payment_agreement()):
        ok = False
    if ok:
        print("默认法律文档 v0.9 创建完成")
    else:
        print("部分默认法律文档创建失败（taskBill 可能未启动，重试即可）")
    return ok


if __name__ == "__main__":
    create_default_legal_documents()
