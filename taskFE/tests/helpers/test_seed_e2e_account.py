"""回归测试：seed_e2e_account.py 补齐公司默认 progress/deliverable 体系（OPT-20260813-008）。

修复前：seed 直插 tenant_company_member 无 company-created 事件链，公司缺
`project_progress_systems_default_tenant` / `project_deliverable_systems_default_tenant`
默认行 → 新建 workspace 后 ensureWorkspaceProgress 因 findTenantDefaultProgress 为空
跳过绑定 → 任务创建报「进度列不属于当前工作空间」、面板「未加载交付物类别」。

修复后：`build_ensure_defaults_sql` 幂等补齐事件链产物（默认进度体系含列 +
默认交付物体系 + 默认工作空间 + workspace 进度绑定）。本测试断言生成 SQL 覆盖上述
全部产物，且行 ID 稳定可重复执行（修复前该函数不存在 → import 失败，测试全红）。
"""
import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).parent))

from seed_e2e_account import build_ensure_defaults_sql  # noqa: E402

COMPANY = "850256677331562496"


@pytest.fixture(scope="module")
def sql() -> str:
    return build_ensure_defaults_sql(COMPANY)


def test_generates_tenant_default_deliverable_mapping(sql):
    # company-created intent 1：独立表 upsert 租户默认交付物体系
    assert "project_deliverable_systems_default_tenant" in sql
    assert f"ddst_{COMPANY}" in sql
    assert "'ds_default_global'" in sql
    assert "ON DUPLICATE KEY UPDATE deliverable_id='ds_default_global'" in sql


def test_generates_tenant_default_progress_mapping(sql):
    # company-created intent 2：target_type='system' 指向 ps_default_system
    assert "project_progress_systems_default_tenant" in sql
    assert f"psdt_{COMPANY}" in sql
    assert "'system'" in sql
    assert "'ps_default_system'" in sql
    assert "ON DUPLICATE KEY UPDATE target_type='system', target_id='ps_default_system'" in sql


def test_generates_system_level_defaults_with_columns(sql):
    # 系统级默认体系（与 dataMigrate 007_seed_defaults.sql 同源，含列）
    assert "'ds_default_global'" in sql
    assert "'ps_default_system'" in sql
    assert "project_deliverable_columns" in sql
    assert "project_progress_columns" in sql
    assert "待处理" in sql and "进行中" in sql and "已完成" in sql and "已取消" in sql
    assert "价值流" in sql and "子工作项 / 子交付物" in sql


def test_binds_all_workspaces_to_default_progress(sql):
    # ensureWorkspaceProgress 的绑定产物：每 workspace 一行 project_progress_systems_workspace
    assert "project_progress_systems_workspace" in sql
    assert f"psw_{COMPANY}_" in sql
    assert "'system'" in sql
    assert "'ps_default_system'" in sql


def test_creates_default_workspace_when_none(sql):
    # company-created intent 3：无工作空间时创建默认工作空间并带 deliverable 绑定
    assert "ws_e2e_default" in sql
    assert "'ds_default_global'" in sql
    assert "is_default" in sql
    assert "WHERE NOT EXISTS" in sql


def test_uses_stable_ids_for_idempotency(sql):
    # 映射/绑定行 ID 稳定派生，重复执行不产生重复行
    assert f"ddst_{COMPANY}" in sql
    assert f"psdt_{COMPANY}" in sql
    assert "INSERT IGNORE" in sql


# ── OPT-20260824-001：超管密码重置（随机密码种子后 E2E 环境回填） ──────────

def test_superadmin_reset_constants():
    """超管 E2E 重置常量必须齐全：邮箱 + bcrypt 哈希；明文只读环境变量（OPT-20260827-015）。"""
    import os

    import pytest
    from seed_e2e_account import (
        SUPERADMIN_EMAIL,
        SUPERADMIN_PLAIN_PASSWORD,
        SUPERADMIN_BCRYPT_HASH,
    )
    assert SUPERADMIN_EMAIL == "author@example.com"
    plain = os.environ.get("SUPERADMIN_PLAIN_PASSWORD")
    # 明文常量来自只读环境变量，禁止硬编码回退
    assert SUPERADMIN_PLAIN_PASSWORD == plain
    if not plain:
        pytest.skip("SUPERADMIN_PLAIN_PASSWORD env not set — skipping bcrypt consistency")
    # 哈希必须确实对应明文（否则重置后登录仍会 400）
    import bcrypt
    assert bcrypt.checkpw(plain.encode(), SUPERADMIN_BCRYPT_HASH.encode()) is True


def test_superadmin_reset_sql_targets_admin_row():
    """重置 SQL 必须只命中超管 email 登录方式（带 binding_voided_at 守卫）。"""
    import inspect
    from seed_e2e_account import reset_superadmin_password

    src = inspect.getsource(reset_superadmin_password)
    assert "author@example.com" in src
    assert "UPDATE task_auth.auth_login_method" in src
    assert "binding_voided_at IS NULL" in src
    assert "method_type = 'email'" in src
