"""回归测试：010_verify_super_admin.py 随机密码安全策略（OPT-20260824-001）。

覆盖：
  - 022 种子使用待填哨兵，不得写死共享 bcrypt / 弱明文哈希
  - 已知弱哈希检测 _is_known_weak_hash 命中/不命中
  - 哨兵 / 历史共享哈希须触发轮换
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


def test_seed_sql_uses_pending_sentinel(v10):
    """022 种子须含待填哨兵，不得再含历史共享 bcrypt 或弱哈希。"""
    sql = (Path(__file__).parent / "022_seed_bootstrap_admin.sql").read_text(encoding="utf-8")
    assert v10.BOOTSTRAP_ADMIN_PASSWORD_PENDING in sql
    assert v10.LEGACY_SHARED_ADMIN_PASSWORD_HASH not in sql
    assert "$2a$12$xq6hKRNnsn3lz4qI5lxtL.4TW1MGbtKAE6Ey/fuHqxEQ.EIC..zV6" not in sql
    assert "'author@example.com'" not in sql
    assert "__BOOTSTRAP_ADMIN_EMAIL__" in sql


def test_weak_hash_detection_hits(v10):
    assert v10._is_known_weak_hash(
        "$2a$12$xq6hKRNnsn3lz4qI5lxtL.4TW1MGbtKAE6Ey/fuHqxEQ.EIC..zV6"
    ) is not None
    assert v10._is_known_weak_hash(
        "$2a$12$LR3V2Zi5lb/s/nFMuOjA2eF5CyNiTRxDRs8mjhua7XKVAkDkvS0C."
    ) is not None


def test_weak_hash_detection_misses_valid_looking(v10):
    # 非弱、非哨兵的合法 bcrypt 形状不应被标为弱明文
    assert v10._is_known_weak_hash(v10.LEGACY_SHARED_ADMIN_PASSWORD_HASH) is None
    assert v10._is_known_weak_hash("") is None
    assert v10._is_known_weak_hash("not-a-bcrypt-hash") is None
    assert v10._is_known_weak_hash(None) is None


def test_needs_rotation_for_sentinel_and_legacy(v10):
    assert v10._needs_password_rotation(v10.BOOTSTRAP_ADMIN_PASSWORD_PENDING) == "pending-sentinel"
    assert v10._needs_password_rotation(v10.LEGACY_SHARED_ADMIN_PASSWORD_HASH) == "legacy-shared-hash"
    assert v10._needs_password_rotation("") == "empty"
    assert v10._needs_password_rotation(None) == "empty"
    assert v10._needs_password_rotation(
        "$2a$12$xq6hKRNnsn3lz4qI5lxtL.4TW1MGbtKAE6Ey/fuHqxEQ.EIC..zV6"
    ).startswith("weak:")


def test_hash_consistent_with_seed_sql(v10):
    """022 哨兵策略一致性护栏须通过。"""
    assert v10._verify_hash_consistent_with_seed_sql() is True


def test_generate_random_password_hash_unique(v10):
    a = v10._generate_random_password_hash()
    b = v10._generate_random_password_hash()
    assert a.startswith("$2")
    assert b.startswith("$2")
    assert a != b


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
