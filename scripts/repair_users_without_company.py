#!/usr/bin/env python3
"""
事件重放脚本 — 纯 Python，无 Django/SaaS 依赖。

往 Kafka 投递 USER_CREATED 事件，修复缺公司的用户。
Go 消费者 (0_create_company → company_created chain) 异步创建公司。

用法:
    python3 scripts/repair_users_without_company.py
    python3 scripts/repair_users_without_company.py --user-id 869798850078994432
    python3 scripts/repair_users_without_company.py --dry-run

依赖: taskAuth (8003) + taskTenantService (8020) + Kafka 必须可达

历史: 原 dataMigrate/saas/03_03_repair_users_without_company.py（依赖 Django）
"""

import argparse
import json
import logging
import os
import sys
import urllib.error
import urllib.request
from typing import Optional, List

logging.basicConfig(level=logging.INFO, format="%(levelname)s: %(message)s")
logger = logging.getLogger(__name__)

# ── 服务地址配置 ──────────────────────────────────────────────────────────
_INFRA_HOST = os.environ.get("INFRA_HOST", "10.2.150.68")
DEFAULT_TASKAUTH_URL = "http://127.0.0.1:8003"
DEFAULT_TENANT_URL = "http://127.0.0.1:8020"
DEFAULT_KAFKA_BOOTSTRAP = f"{_INFRA_HOST}:9093"
USER_CREATED_TOPIC = "user-created"


# ── HTTP helpers ──────────────────────────────────────────────────────────

def _http_get(url: str, timeout: int = 10) -> tuple[int, dict]:
    req = urllib.request.Request(url)
    req.add_header("Accept", "application/json")
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            return resp.status, json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as e:
        body_raw = e.read().decode("utf-8", errors="replace")
        try:
            return e.code, json.loads(body_raw)
        except json.JSONDecodeError:
            return e.code, {"detail": body_raw}
    except (urllib.error.URLError, OSError) as e:
        return 0, {"error": str(e)}


def _http_post(url: str, body: dict, timeout: int = 10) -> tuple[int, dict]:
    data = json.dumps(body).encode("utf-8")
    req = urllib.request.Request(url, data=data, method="POST")
    req.add_header("Content-Type", "application/json")
    req.add_header("Accept", "application/json")
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            return resp.status, json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as e:
        body_raw = e.read().decode("utf-8", errors="replace")
        try:
            return e.code, json.loads(body_raw)
        except json.JSONDecodeError:
            return e.code, {"detail": body_raw}
    except (urllib.error.URLError, OSError) as e:
        return 0, {"error": str(e)}


# ── 服务 API ──────────────────────────────────────────────────────────────

def _taskauth_base() -> str:
    return os.environ.get("TASKAUTH_BASE_URL", DEFAULT_TASKAUTH_URL).rstrip("/")


def _tenant_base() -> str:
    return os.environ.get("TENANT_SERVICE_URL", DEFAULT_TENANT_URL).rstrip("/")


def get_all_user_ids() -> List[str]:
    """从 taskAuth 获取所有活跃用户 ID 列表。"""
    ids: set[str] = set()
    base = _taskauth_base()

    # 途径1: taskAuth 内部用户列表
    status, data = _http_get(f"{base}/api/internal/users/")
    if status == 200:
        users = data if isinstance(data, list) else data.get("users", []) if isinstance(data, dict) else []
        for u in users:
            uid = str(u.get("id", "")).strip() if isinstance(u, dict) else str(getattr(u, "id", "")).strip()
            if uid:
                ids.add(uid)

    # 途径2: 批量解析 (备选)
    if not ids:
        logger.warning("taskAuth 用户列表为空，尝试备选方案...")
        status2, data2 = _http_get(f"{base}/api/internal/user-count/")
        if status2 == 200:
            logger.info(f"taskAuth 用户总数: {data2.get('count', 0)}")

    if not ids:
        logger.warning("无用户可扫描（数据库为空或 taskAuth 中无用户），无需修复")
        return []

    return sorted(ids)


def user_has_own_company(user_id: str) -> bool:
    """检查用户是否已有作为创建者的公司。

    受邀加入他人公司不算：USER_CREATED 建的是「自己的公司」
    （CompanyByCreator），仅看 members/exists 会误跳过受邀用户。
    """
    base = _tenant_base()
    status, data = _http_get(
        f"{base}/api/internal/tenant/companies/by-creator?creator_id={user_id}"
    )
    if status == 200:
        if isinstance(data, list):
            return len(data) > 0
        if isinstance(data, dict):
            return bool(data.get("id") or data.get("companies"))
    # API 不可达时假定无自有公司（之后 Kafka 重放会修复）
    return False


# 兼容旧名：历史调用方/文档可能仍写 user_has_membership
def user_has_membership(user_id: str) -> bool:
    return user_has_own_company(user_id)


def resolve_user_info(user_id: str) -> tuple[str, str]:
    """从 taskAuth 解析用户名和邮箱。返回 (username, email)。"""
    uid = str(user_id).strip()
    base = _taskauth_base()
    username = uid[:8]
    email = ""

    # 通过用户详情 API 获取
    status, data = _http_get(f"{base}/api/accounts/users/{uid}/")
    if status == 200:
        email = str(data.get("email", "")).strip()
        username = str(data.get("username", "")).strip() or (email.split("@")[0] if email else uid[:8])

    return username, email


# ── Kafka ─────────────────────────────────────────────────────────────────

def _get_kafka_producer():
    """延迟导入 confluent_kafka。"""
    from confluent_kafka import Producer
    bootstrap = os.environ.get("KAFKA_BOOTSTRAP_SERVERS", DEFAULT_KAFKA_BOOTSTRAP)
    return Producer({"bootstrap.servers": bootstrap})


def send_user_created_event(user_id: str, username: str, email: str) -> bool:
    """向 Kafka 投递 USER_CREATED 事件（wrapped in envelope per broker wire format）。
    返回是否成功。"""
    producer = _get_kafka_producer()
    event_data = {
        "user_id": user_id,
        "username": username,
        "email": email,
        "phone": None,
        "is_active": True,
    }
    # Wire format: {"event_type": "USER_CREATED", "data": {...}}
    # See taskEvents/broker/envelope.go wireEnvelope
    envelope = {
        "event_type": "USER_CREATED",
        "data": event_data,
    }
    try:
        producer.produce(
            USER_CREATED_TOPIC,
            key=user_id.encode("utf-8"),
            value=json.dumps(envelope).encode("utf-8"),
            callback=lambda err, msg: None,
        )
        producer.flush(timeout=10)
        return True
    except Exception as exc:
        logger.error("Kafka produce 失败: %s", exc)
        return False


# ── 核心逻辑 ──────────────────────────────────────────────────────────────

def fire_user_created(user_id: str, dry_run: bool = False) -> dict:
    """检查并投递 USER_CREATED 事件。返回结果字典。"""
    uid = str(user_id).strip()
    result = {"user_id": uid, "action": "none"}

    # 跳过已有「自有公司」的用户（受邀成员不算）
    if user_has_own_company(uid):
        result["action"] = "skip"
        result["detail"] = "own company already exists (by-creator)"
        return result

    username, email = resolve_user_info(uid)
    # resolve_user_info 无昵称时回退 uid[:8]（雪花前缀），不适合作公司名；
    # 置空交给消费者按个人昵称生成「{user}的公司」，再空则「我的公司」。
    if not username or username == uid[:8]:
        username = ""
    display_name = username or "我的公司"
    result["username"] = display_name

    if dry_run:
        result["action"] = "would_fire"
        result["detail"] = f"would fire USER_CREATED for {display_name} ({email or 'no email'})"
        return result

    if send_user_created_event(uid, username, email):
        result["action"] = "fired"
        result["detail"] = f"USER_CREATED event sent to Kafka for {display_name}"
        logger.info("[REPLAY] USER_CREATED → %s (%s)", display_name, uid)
    else:
        result["action"] = "error"
        result["detail"] = "Failed to send Kafka event"

    return result


def main():
    parser = argparse.ArgumentParser(
        description="往 Kafka 重放 USER_CREATED 事件，修复缺公司的用户"
    )
    parser.add_argument("--user-id", help="仅修复指定用户（默认扫描所有用户）")
    parser.add_argument("--dry-run", action="store_true", help="仅检测，不实际投递事件")
    parser.add_argument("--taskauth-url", default=None)
    parser.add_argument("--tenant-url", default=None)
    parser.add_argument("--kafka-bootstrap", default=None)
    args = parser.parse_args()

    if args.taskauth_url:
        os.environ["TASKAUTH_BASE_URL"] = args.taskauth_url
    if args.tenant_url:
        os.environ["TENANT_SERVICE_URL"] = args.tenant_url
    if args.kafka_bootstrap:
        os.environ["KAFKA_BOOTSTRAP_SERVERS"] = args.kafka_bootstrap

    if args.user_id:
        user_ids = [args.user_id]
    else:
        user_ids = get_all_user_ids()

    if not user_ids:
        logger.info("无用户需要修复，正常退出。")
        print("无用户需要修复（数据库为空），正常退出。")
        return

    logger.info("扫描 %d 个用户...", len(user_ids))
    if args.dry_run:
        logger.info("DRY RUN MODE — 不投递事件")

    results = []
    for uid in user_ids:
        result = fire_user_created(uid, dry_run=args.dry_run)
        results.append(result)

    # 汇总
    fired = [r for r in results if r["action"] == "fired"]
    skipped = [r for r in results if r["action"] == "skip"]
    errors = [r for r in results if r["action"] == "error"]
    would = [r for r in results if r["action"] == "would_fire"]

    print(f"\n=== 事件重放结果 ===")
    print(f"总用户数: {len(results)}")
    print(f"已投递事件: {len(fired)}")
    for r in fired:
        print(f'  user={r["user_id"]} username={r.get("username")}')
    print(f"已跳过(有公司): {len(skipped)}")
    print(f"待投递(dry-run): {len(would)}")
    print(f"错误: {len(errors)}")
    for r in errors:
        print(f'  user={r["user_id"]}: {r.get("detail")}')

    if fired:
        print(f"\n✓ 已往 Kafka 投递 {len(fired)} 个 USER_CREATED 事件")
        print("  Go 消费者将异步创建公司 → 交付物/进度体系 → 默认工作空间")

    if errors:
        sys.exit(1)


if __name__ == "__main__":
    main()
