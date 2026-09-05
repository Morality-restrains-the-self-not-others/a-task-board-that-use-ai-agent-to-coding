"""回归测试：010_verify_super_admin.py 随机密码安全策略（OPT-20260824-001）。

覆盖：
  - 种子管理员哈希必须为随机值（不得命中已知弱明文 admin123 / rgNodkdq8677!ci）
  - 已知弱哈希检测 _is_known_weak_hash 命中/不命中
  - 010 与 022_seed_bootstrap_admin.sql 的哈希一致性护栏
  - 管理员邮箱从 conf bootstrapAdmin.email 读取（非脚本硬编码 SSOT）
"""
import importlib.util
import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).parent))


def _load_010():
    spec = importlib.util.spec_from_file_location(
        "verify_super_admin_mod", str(Path(__file__).parent / "010_verify_super_admin.py")
    )
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


@pytest.fixture(scope="module")
def v10():
    return _load_010()


# ── 1. 种子哈希必须为随机值（不得命中已知弱明文） ─────────────────────────

def test_seed_hash_not_weak(v10):
    """022 种子中的管理员密码哈希不得再对应已知弱明文。"""
    import bcrypt
    assert bcrypt.checkpw(b"admin123", v10.RANDOM_ADMIN_PASSWORD_HASH.encode()) is False
    assert bcrypt.checkpw(b"rgNodkdq8677!ci", v10.RANDOM_ADMIN_PASSWORD_HASH.encode()) is False


# ── 2. 弱哈希检测 ─────────────────────────────────────────────────────────

def test_weak_hash_detection_hits(v10):
    assert v10._is_known_weak_hash(
        "$2a$12$xq6hKRNnsn3lz4qI5lxtL.4TW1MGbtKAE6Ey/fuHqxEQ.EIC..zV6"
    ) is not None
    assert v10._is_known_weak_hash(
        "$2a$12$LR3V2Zi5lb/s/nFMuOjA2eF5CyNiTRxDRs8mjhua7XKVAkDkvS0C."
    ) is not None


def test_weak_hash_detection_misses_random(v10):
    assert v10._is_known_weak_hash(v10.RANDOM_ADMIN_PASSWORD_HASH) is None
    assert v10._is_known_weak_hash("") is None
    assert v10._is_known_weak_hash("not-a-bcrypt-hash") is None
    assert v10._is_known_weak_hash(None) is None


# ── 3. 010 与 022 种子 SQL 哈希一致性护栏 ─────────────────────────────────

def test_hash_consistent_with_seed_sql(v10):
    """010 的 RANDOM_ADMIN_PASSWORD_HASH 必须与 022_seed_bootstrap_admin.sql 中一致。"""
    assert v10._verify_hash_consistent_with_seed_sql() is True


def test_seed_sql_contains_random_hash(v10):
    sql = (Path(__file__).parent / "022_seed_bootstrap_admin.sql").read_text(encoding="utf-8")
    assert v10.RANDOM_ADMIN_PASSWORD_HASH in sql
    # 022 中不得再出现已知弱明文哈希（admin123 旧哈希）
    assert "$2a$12$xq6hKRNnsn3lz4qI5lxtL.4TW1MGbtKAE6Ey/fuHqxEQ.EIC..zV6" not in sql
    assert "'author@example.com'" not in sql
    assert "__BOOTSTRAP_ADMIN_EMAIL__" in sql


def test_admin_email_from_conf_not_script_ssot(v10):
    """管理员邮箱 SSOT 是 conf bootstrapAdmin.email，脚本不得再写死常量。"""
    import yaml

    email = v10.load_bootstrap_admin_email()
    assert "@" in email
    src = Path(v10.__file__).read_text(encoding="utf-8")
    assert 'ADMIN_EMAIL = "' not in src
    conf_path = v10._find_repo_root() / "conf" / "auth" / "task-auth" / "config.yaml"
    tracked = yaml.safe_load(conf_path.read_text(encoding="utf-8"))
    tracked_email = str((tracked.get("bootstrapAdmin") or {}).get("email") or "").strip()
    local_path = v10._find_repo_root() / "conf-local" / "auth" / "task-auth" / "config.yaml"
    if local_path.is_file():
        local = yaml.safe_load(local_path.read_text(encoding="utf-8")) or {}
        local_email = str((local.get("bootstrapAdmin") or {}).get("email") or "").strip()
        if local_email:
            tracked_email = local_email
    assert email == tracked_email
