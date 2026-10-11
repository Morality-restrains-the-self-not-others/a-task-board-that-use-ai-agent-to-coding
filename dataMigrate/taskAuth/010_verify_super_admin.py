#!/usr/bin/env python3
"""
确认/轮换超级管理员凭证 — 纯 Python，无 Django/SaaS 依赖。

管理员创建由 taskAuth bootstrap-admin 负责（db/task-auth/init.sh → taskAuth/run.sh bootstrap-admin），
种子数据在 dataMigrate/taskAuth/022_seed_bootstrap_admin.sql（INSERT IGNORE，幂等）。

安全策略（OPT-20260824-001）：管理员密码在初始化（bootstrap-admin）时按环境生成随机值，
明文不落盘。022 种子写入哨兵 __BOOTSTRAP_ADMIN_PASSWORD_PENDING__，由 CLI 替换为 bcrypt。
任何人（含开发/部署）均不知晓明文。管理员通过「忘记密码 → 邮箱重置链接」设置新密码。

本脚本职责（幂等安全，可重复执行）：
  1. 声明性确认：bootstrap admin 存在于 task_auth MySQL（或 HTTP fallback）。
  2. 已知弱密码 / 待填哨兵 / 历史共享哈希检测：命中则生成新的 crypto 随机密码哈希并写入。
  3. 一致性护栏：022 种子 SQL 必须含待填哨兵，不得再写死共享 bcrypt。
  4. 管理员邮箱 SSOT 为 conf/auth/task-auth/config.yaml 的 bootstrapAdmin.email（叠 conf-local）。
     022 SQL 使用占位符，由 migrate 渲染，本脚本只校验 conf 邮箱对应的登录行。

用法:
    python3 dataMigrate/taskAuth/010_verify_super_admin.py

历史: 原 `dataMigrate/saas/01_02_create_admin_ruandao.py` 读取 SQLite auth.db 校验。
MySQL 迁移后 auth.db 已废弃，管理员凭据存在于 task_auth MySQL 数据库中。
"""

import os
import secrets
import sys
from pathlib import Path

_HERE = Path(__file__).resolve().parent
if str(_HERE) not in sys.path:
    sys.path.insert(0, str(_HERE))
from bootstrap_admin_email import (  # noqa: E402
    find_repo_root as _find_repo_root,
    load_bootstrap_admin_email,
)

# 022 种子哨兵：bootstrap-admin / 本脚本命中后替换为真随机 bcrypt。
BOOTSTRAP_ADMIN_PASSWORD_PENDING = "__BOOTSTRAP_ADMIN_PASSWORD_PENDING__"

# 历史共享「随机」哈希（曾硬编码进 022）；命中亦须轮换为每环境独立随机。
LEGACY_SHARED_ADMIN_PASSWORD_HASH = "$2a$12$reqIRh/zFv.aez0dNtwWJ.kED6CpqFsppSm8pBOk6Vd9n8hKOK2A2"

# 已知弱明文密码（开发遗留，检测命中即轮换）
KNOWN_WEAK_PASSWORDS = ["admin123", "rgNodkdq8677!ci"]

# taskAuth MySQL 连接信息（与 taskAuth 服务配置对齐）
_TASKAUTH_HOST = os.environ.get("TASKAUTH_MYSQL_HOST", os.environ.get("INFRA_HOST", "10.2.150.68"))
_TASKAUTH_PORT = os.environ.get("TASKAUTH_MYSQL_PORT", "3306")
_TASKAUTH_USER = os.environ.get("TASKAUTH_MYSQL_USER", "root")
_TASKAUTH_PASS = os.environ.get("TASKAUTH_MYSQL_PASSWORD", "root123")
_TASKAUTH_DB = os.environ.get("TASKAUTH_MYSQL_DATABASE", "task_auth")


def _generate_random_password_hash() -> str:
    """生成新的随机密码 bcrypt 哈希；明文立即丢弃。"""
    import bcrypt

    plain = secrets.token_urlsafe(32)
    hashed = bcrypt.hashpw(plain.encode(), bcrypt.gensalt(rounds=12)).decode()
    del plain
    return hashed


def _is_known_weak_hash(password_hash: str) -> str | None:
    """判断哈希是否对应已知弱明文。返回命中的明文或 None。"""
    if not password_hash:
        return None
    try:
        import bcrypt

        for pw in KNOWN_WEAK_PASSWORDS:
            try:
                if bcrypt.checkpw(pw.encode(), password_hash.encode()):
                    return pw
            except ValueError:
                continue  # 非法 bcrypt 哈希，跳过
    except ImportError:
        # bcrypt 模块缺失 → 退化为精确字符串匹配已知哈希
        if password_hash in {
            "$2a$12$xq6hKRNnsn3lz4qI5lxtL.4TW1MGbtKAE6Ey/fuHqxEQ.EIC..zV6",  # admin123
            "$2a$12$LR3V2Zi5lb/s/nFMuOjA2eF5CyNiTRxDRs8mjhua7XKVAkDkvS0C.",  # rgNodkdq8677!ci
        }:
            return "（未知明文，命中已知哈希列表）"
    return None


def _needs_password_rotation(password_hash: str | None) -> str | None:
    """若需轮换返回原因文案，否则 None。"""
    if not password_hash:
        return "empty"
    if password_hash == BOOTSTRAP_ADMIN_PASSWORD_PENDING:
        return "pending-sentinel"
    if password_hash == LEGACY_SHARED_ADMIN_PASSWORD_HASH:
        return "legacy-shared-hash"
    weak = _is_known_weak_hash(password_hash)
    if weak is not None:
        return f"weak:{weak}"
    return None


def _mysql_conn():
    import mysql.connector

    return mysql.connector.connect(
        host=_TASKAUTH_HOST,
        port=int(_TASKAUTH_PORT),
        user=_TASKAUTH_USER,
        password=_TASKAUTH_PASS,
        database=_TASKAUTH_DB,
        connect_timeout=5,
    )


def _login_method_table() -> str:
    """返回 accounts_login_method / auth_login_method 中实际存在的表名（016 重命名前后兼容）。"""
    try:
        conn = _mysql_conn()
        cursor = conn.cursor()
        cursor.execute(
            """SELECT TABLE_NAME FROM information_schema.TABLES
               WHERE TABLE_SCHEMA = DATABASE()
                 AND TABLE_NAME IN ('accounts_login_method', 'auth_login_method')
               ORDER BY TABLE_NAME DESC LIMIT 1"""
        )
        row = cursor.fetchone()
        cursor.close()
        conn.close()
        return str(row[0]) if row else "auth_login_method"
    except Exception:
        return "auth_login_method"


def _check_admin_in_mysql(email: str) -> str | None:
    """通过直连 MySQL 检查管理员是否存在。返回 user_id 或 None。"""
    try:
        import mysql.connector  # noqa: F401
    except ImportError:
        print("mysql-connector-python 未安装，跳过 MySQL 直连校验")
        return "mysql-skipped"

    try:
        conn = _mysql_conn()
        table = _login_method_table()
        cursor = conn.cursor()
        cursor.execute(
            f"""SELECT object_id FROM `{table}`
               WHERE LOWER(identifier) = LOWER(%s)
                 AND method_type = 'email'
                 AND binding_voided_at IS NULL
               LIMIT 1""",
            (email,),
        )
        row = cursor.fetchone()
        cursor.close()
        conn.close()

        if row and row[0]:
            return str(row[0])
        return None
    except Exception as e:
        print(f"MySQL 校验失败（非致命）: {e}")
        return "mysql-unreachable"


def _get_password_hash_in_mysql(email: str) -> str | None:
    """读取管理员当前 password_hash。无法连接时返回 None。"""
    try:
        conn = _mysql_conn()
        table = _login_method_table()
        cursor = conn.cursor()
        cursor.execute(
            f"""SELECT password_hash FROM `{table}`
               WHERE LOWER(identifier) = LOWER(%s)
                 AND method_type = 'email'
                 AND binding_voided_at IS NULL
               LIMIT 1""",
            (email,),
        )
        row = cursor.fetchone()
        cursor.close()
        conn.close()
        return str(row[0]) if row and row[0] else None
    except Exception as e:
        print(f"MySQL 读取密码哈希失败（非致命）: {e}")
        return None


def _rotate_to_random_password(email: str, lm_user_id: str) -> bool:
    """将管理员 password_hash 轮换为新生成的随机哈希（明文丢弃）。"""
    try:
        new_hash = _generate_random_password_hash()
        conn = _mysql_conn()
        table = _login_method_table()
        cursor = conn.cursor()
        cursor.execute(
            f"""UPDATE `{table}`
               SET password_hash = %s, updated_at = NOW()
               WHERE object_id = %s AND method_type = 'email'""",
            (new_hash, lm_user_id),
        )
        conn.commit()
        cursor.close()
        conn.close()
        return True
    except Exception as e:
        print(f"密码轮换失败: {e}")
        return False


def _verify_hash_consistent_with_seed_sql() -> bool:
    """一致性护栏：022 必须含待填哨兵，且不得再写死共享 bcrypt。"""
    seed_sql = os.path.join(
        os.path.dirname(os.path.abspath(__file__)), "022_seed_bootstrap_admin.sql"
    )
    try:
        with open(seed_sql, encoding="utf-8") as f:
            content = f.read()
    except OSError as e:
        print(f"⚠ 无法读取种子 SQL 做一致性检查: {e}")
        return True  # 非致命
    ok = True
    if BOOTSTRAP_ADMIN_PASSWORD_PENDING not in content:
        print(
            f"✗ 一致性错误: 022_seed_bootstrap_admin.sql 缺少哨兵 "
            f"{BOOTSTRAP_ADMIN_PASSWORD_PENDING}"
        )
        ok = False
    if LEGACY_SHARED_ADMIN_PASSWORD_HASH in content:
        print(
            "✗ 一致性错误: 022 仍含历史共享 bcrypt 哈希，"
            "须改为待填哨兵并由 bootstrap-admin 生成随机密码"
        )
        ok = False
    return ok


def ensure_super_admin(email: str | None = None) -> str:
    """确认超级管理员存在并确保密码为随机值（幂等）。返回 user_id。"""
    if email is None:
        email = load_bootstrap_admin_email()
    consistent = _verify_hash_consistent_with_seed_sql()

    # 方式1: 直连 MySQL
    uid = _check_admin_in_mysql(email)
    if uid and uid != "mysql-skipped" and uid != "mysql-unreachable":
        current_hash = _get_password_hash_in_mysql(email)
        reason = _needs_password_rotation(current_hash)
        if reason is not None:
            print(f"⚠ 检测到管理员 {email} 密码需轮换（{reason}），生成新随机密码…")
            if _rotate_to_random_password(email, uid):
                print("✓ 已写入新随机密码哈希。管理员须通过邮箱重置设置新密码。")
            else:
                print("✗ 轮换失败，请手动更新 auth_login_method.password_hash")
        print(f"已确认 SuperAdmin {email}，user_id={uid}（task_auth MySQL）")
        return uid
    if uid == "mysql-unreachable":
        # 方式2: HTTP API fallback（无法读取/轮换哈希，仅确认存在）
        uid2 = _check_admin_via_http(email)
        if uid2:
            print(f"已确认 SuperAdmin {email}，user_id={uid2}（taskAuth HTTP API）")
            return uid2

    # 无法校验时信任 bootstrap-admin 的幂等性
    print(f"SuperAdmin {email}：taskAuth bootstrap-admin 负责创建（幂等安全）")
    print("  无法直连 MySQL 或 taskAuth HTTP API，假定管理员已存在")
    if not consistent:
        print("  ⚠ 请注意上方一致性错误未解决，密码轮换可能失效")
    return "bootstrap-admin-assumed"


def _check_admin_via_http(email: str) -> str | None:
    """通过 taskAuth HTTP API 检查管理员是否存在。返回 user_id 或 None。"""
    import json
    import urllib.error
    import urllib.request

    base = os.environ.get("TASKAUTH_BASE_URL", "http://127.0.0.1:8003")
    try:
        req = urllib.request.Request(
            f"{base}/api/internal/admin/exists?email={email}",
        )
        req.add_header("Accept", "application/json")
        with urllib.request.urlopen(req, timeout=5) as resp:
            data = json.loads(resp.read().decode())
            return str(data.get("user_id", "")) or None
    except (urllib.error.URLError, urllib.error.HTTPError, json.JSONDecodeError, OSError):
        return None


def _print_reset_guidance(email: str) -> None:
    print(f"\n管理员 {email} 登录指引（安全策略 OPT-20260824-001）:")
    print("  密码为初始化时随机生成且未在任何地方保存，任何人不知晓。")
    print("  首次使用请走「忘记密码」流程:")
    print("    1) 登录页点击「忘记密码」，输入管理员邮箱")
    print("    2) 查收邮件中的重置链接（taskAuth → Kafka/SMTP 投递）")
    print("    3) 打开链接设置新密码，之后用新密码登录")


if __name__ == "__main__":
    try:
        email = load_bootstrap_admin_email()
        uid = ensure_super_admin(email)
        print("\n管理员确认完成！")
        print(f"邮箱: {email}")
        print(f"user_id: {uid}")
        print("（密码不在脚本中保存 — 初始化随机生成，明文已销毁）")
        _print_reset_guidance(email)
    except Exception as e:
        print(f"管理员校验错误: {e}")
        import traceback

        traceback.print_exc()
        sys.exit(1)
