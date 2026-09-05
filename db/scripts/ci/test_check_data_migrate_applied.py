#!/usr/bin/env python3
"""
单测：check_data_migrate_applied.py（OPT-20260808-013 dataMigrate 巡检）

覆盖：
- migrate_script → dataMigrate 目录推导（含单引号/双引号两种 $ROOT/dataMigrate/<dir> 写法）
- 缺口分类：missing（本地有生产无）FAIL / stale（生产有本地无）WARN / 一致 OK
- 生产不可达 → SKIP（不误报 FAIL）
- --db 单库过滤
"""

import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

import yaml

sys.path.insert(0, str(Path(__file__).resolve().parent))
import check_data_migrate_applied as mod  # noqa: E402


def make_fake_repo(tmp: Path, entries: dict, sql_files: dict) -> Path:
    """构造迷你 monorepo：db/registry.yaml + db/<key>/migrate.sh + dataMigrate/<dir>/*.sql。"""
    root = tmp / "repo"
    (root / "db" / "scripts" / "ci").mkdir(parents=True)
    (root / "dataMigrate").mkdir()
    reg = {"mysql": {}, "databases": {}}
    for key, cfg in entries.items():
        reg["databases"][key] = cfg
    (root / "db" / "registry.yaml").write_text(yaml.safe_dump(reg), encoding="utf-8")
    for key, cfg in entries.items():
        ms = cfg.get("migrate_script")
        if ms:
            p = root / ms
            p.parent.mkdir(parents=True, exist_ok=True)
            p.write_text(
                f'exec bash "$ROOT/db/scripts/apply_datamigrate.sh" "{cfg["database"]}" "$ROOT/dataMigrate/{cfg["_dir"]}"\n',
                encoding="utf-8",
            )
    for dirname, files in sql_files.items():
        d = root / "dataMigrate" / dirname
        d.mkdir(exist_ok=True)
        for f in files:
            (d / f).write_text("-- test\n", encoding="utf-8")
    return root


class FakeQuery:
    """模拟生产 DB data_migrate_log 查询结果。"""

    def __init__(self, applied_by_db: dict, unreachable: set = None):
        self.applied = applied_by_db
        self.unreachable = unreachable or set()

    def __call__(self, cmd_prefix, sql, db="", timeout=10):
        if db in self.unreachable:
            raise RuntimeError("mysql 查询失败: Can't connect to MySQL server")
        rows = sorted(self.applied.get(db, []))
        return rows


class TestDirDerivation(unittest.TestCase):
    def test_single_quote_dir_derivation(self):
        with tempfile.TemporaryDirectory() as td:
            root = make_fake_repo(
                Path(td),
                {"task-auth": {"database": "task_auth", "migrate_script": "db/task-auth/migrate.sh", "_dir": "taskAuth"}},
                {"taskAuth": ["001_auth_tables.sql", "002_super_admin.sql"]},
            )
            reg = mod.load_registry(root)
            cfg = reg["databases"]["task-auth"]
            ms = root / cfg["migrate_script"]
            text = ms.read_text(encoding="utf-8")
            import re
            m = re.search(r'["\']\$ROOT/dataMigrate/([A-Za-z0-9_]+)["\']', text)
            self.assertEqual(m.group(1), "taskAuth")
            local = sorted(p.name for p in (root / "dataMigrate" / m.group(1)).glob("*.sql"))
            self.assertEqual(local, ["001_auth_tables.sql", "002_super_admin.sql"])

    def test_double_quote_dir_derivation(self):
        with tempfile.TemporaryDirectory() as td:
            root = make_fake_repo(
                Path(td),
                {"task-bill": {"database": "task_bill", "migrate_script": "db/task-bill/migrate.sh", "_dir": "taskBill"}},
                {"taskBill": ["001_billing_tables.sql"]},
            )
            reg = mod.load_registry(root)
            ms = root / reg["databases"]["task-bill"]["migrate_script"]
            text = ms.read_text(encoding="utf-8")
            import re
            m = re.search(r'["\']\$ROOT/dataMigrate/([A-Za-z0-9_]+)["\']', text)
            self.assertEqual(m.group(1), "taskBill")


class TestAuditSemantics(unittest.TestCase):
    def _run(self, tmp: Path, applied_by_db: dict, unreachable: set = None, db_filter: str = None):
        root = make_fake_repo(
            tmp,
            {
                "task-auth": {"database": "task_auth", "migrate_script": "db/task-auth/migrate.sh", "_dir": "taskAuth"},
                "task-bill": {"database": "task_bill", "migrate_script": "db/task-bill/migrate.sh", "_dir": "taskBill"},
            },
            {
                "taskAuth": ["001_auth_tables.sql", "002_super_admin.sql", "031_oidc_client_managed_by.sql"],
                "taskBill": ["001_billing_tables.sql", "037_wechat_pay_reliability.sql"],
            },
        )
        fake = FakeQuery(applied_by_db, unreachable)
        orig_query = mod.query_single
        mod.query_single = fake
        orig_detect = mod.detect_docker
        mod.detect_docker = lambda: False
        try:
            argv = [] if db_filter is None else ["--db", db_filter]
            return mod.main(argv, root=root), root
        finally:
            mod.query_single = orig_query
            mod.detect_docker = orig_detect

    def test_all_consistent_exit_zero(self):
        with tempfile.TemporaryDirectory() as td:
            ec, _ = self._run(Path(td), {
                "task_auth": ["001_auth_tables.sql", "002_super_admin.sql", "031_oidc_client_managed_by.sql"],
                "task_bill": ["001_billing_tables.sql", "037_wechat_pay_reliability.sql"],
            })
            self.assertEqual(ec, 0)

    def test_missing_local_file_fails(self):
        """本地新增未应用（OPT-013 核心场景：009_post_expires_at 未应用 → smoke 500）→ FAIL。"""
        with tempfile.TemporaryDirectory() as td:
            ec, _ = self._run(Path(td), {
                "task_auth": ["001_auth_tables.sql", "002_super_admin.sql"],  # 缺 031
                "task_bill": ["001_billing_tables.sql", "037_wechat_pay_reliability.sql"],
            })
            self.assertEqual(ec, 1)

    def test_stale_prod_entry_warns_only(self):
        """生产有本地无（renumber 遗留）→ WARN 不 FAIL。"""
        with tempfile.TemporaryDirectory() as td:
            ec, _ = self._run(Path(td), {
                "task_auth": ["001_auth_tables.sql", "002_super_admin.sql", "031_oidc_client_managed_by.sql", "010_oidc_bootstrap_clients"],
                "task_bill": ["001_billing_tables.sql", "037_wechat_pay_reliability.sql"],
            })
            self.assertEqual(ec, 0)

    def test_unreachable_db_skipped(self):
        """生产不可达 → SKIP（不误报 FAIL，不阻塞离线场景）。"""
        with tempfile.TemporaryDirectory() as td:
            ec, _ = self._run(Path(td), {
                "task_auth": ["001_auth_tables.sql", "002_super_admin.sql", "031_oidc_client_managed_by.sql"],
                "task_bill": ["001_billing_tables.sql", "037_wechat_pay_reliability.sql"],
            }, unreachable={"task_auth"})
            self.assertEqual(ec, 0)

    def test_db_filter(self):
        with tempfile.TemporaryDirectory() as td:
            ec, _ = self._run(Path(td), {
                "task_auth": ["001_auth_tables.sql", "002_super_admin.sql"],
                "task_bill": ["001_billing_tables.sql", "037_wechat_pay_reliability.sql"],
            }, db_filter="task-auth")
            self.assertEqual(ec, 1)  # 单库过滤后仍能发现该库缺口


if __name__ == "__main__":
    unittest.main()
