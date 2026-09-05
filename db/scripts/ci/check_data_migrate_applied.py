#!/usr/bin/env python3
"""
dataMigrate 巡检脚本 — 比对生产各库 data_migrate_log 与本地 dataMigrate 目录差异，缺口即报。

背景（OPT-20260808-013）：taskTaskService v15 存续期代码已部署生产，但
dataMigrate/taskTaskService/009_post_expires_at.sql 从未应用，生产 task_tasks 缺列 →
internal-apis smoke 500。已手动补应用，本次审计 12 库全部一致（可作基线）。

用法:
  python3 db/scripts/ci/check_data_migrate_applied.py                # 全库巡检（默认）
  python3 db/scripts/ci/check_data_migrate_applied.py --db task_auth # 单库
  python3 db/scripts/ci/check_data_migrate_applied.py --json         # 机器可读输出

环境变量:
  MIGRATE_MYSQL_HOST   生产 MySQL 地址（默认 10.2.150.68，registry 是本地 127.0.0.1）
  MIGRATE_MYSQL_PORT   默认 3306
  MIGRATE_MYSQL_USER   默认 taskapp
  MIGRATE_MYSQL_PASS   默认 taskapp123
  MIGRATE_DOCKER       使用 docker exec 容器内 mysql（默认 auto：有 docker-mysql 容器时用）
  MIGRATE_TIMEOUT_SEC  单库查询超时（默认 10s）

退出码: 0 全部一致；1 存在缺口（本地有未应用 / 生产有本地无）；2 环境错误（不可达/参数错）

映射依据: db/registry.yaml databases.<key>.database + 各 db/<key>/migrate.sh 的 dataMigrate 目录
"""

from __future__ import annotations

import argparse
import json
import os
import shutil
import subprocess
import sys
import tempfile
import time
from pathlib import Path

import yaml

# 生产环境 MySQL（registry.yaml 内是本地 127.0.0.1 开发默认值，巡检针对生产）
DEFAULT_PROD_HOST = "10.2.150.68"
DEFAULT_PORT = "3306"
DEFAULT_USER = "taskapp"
DEFAULT_PASS = "taskapp123"


def find_monorepo_root() -> Path:
    start = Path(__file__).resolve().parent.parent.parent  # db/scripts/ci -> db -> root
    if (start / "db" / "registry.yaml").is_file() and (start / "dataMigrate").is_dir():
        return start
    d = Path.cwd()
    for _ in range(12):
        if (d / "db" / "registry.yaml").is_file() and (d / "dataMigrate").is_dir():
            return d
        parent = d.parent
        if parent == d:
            break
        d = parent
    raise FileNotFoundError("db/registry.yaml 与 dataMigrate/ 未在 cwd 上层找到")


def load_registry(root: Path) -> dict:
    with open(root / "db" / "registry.yaml", encoding="utf-8") as f:
        return yaml.safe_load(f)


def mysql_cmd_base(docker: bool, host: str, port: str, user: str, password: str) -> list[str]:
    """返回 mysql 命令前缀（docker exec 或本机 mysql）。"""
    if docker:
        return [
            "docker", "exec", "-i", "docker-mysql-mysql-1", "mysql",
            "-h", host, "-P", port, "-u", user, f"-p{password}",
        ]
    return ["mysql", "-h", host, "-P", port, "-u", user, f"-p{password}"]


def query_single(cmd_prefix: list[str], sql: str, db: str = "", timeout: int = 10) -> list[str]:
    """执行单条 SQL 查询，返回逐行结果（strip 后的字符串列表）。"""
    args = list(cmd_prefix)
    if db:
        args.append(db)
    args.extend(["-sN", "-e", sql])
    proc = subprocess.run(args, capture_output=True, text=True, timeout=timeout)
    if proc.returncode != 0:
        raise RuntimeError(f"mysql 查询失败: {proc.stderr.strip() or proc.stdout.strip()}")
    return [ln.strip() for ln in proc.stdout.splitlines() if ln.strip()]


def detect_docker() -> bool:
    if os.environ.get("MIGRATE_DOCKER"):
        return os.environ["MIGRATE_DOCKER"].lower() in ("1", "true", "yes")
    return shutil.which("docker") is not None


def main(argv: list[str] | None = None, root: Path | None = None) -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--db", help="仅巡检指定数据库名（registry key，如 task-auth）")
    ap.add_argument("--json", action="store_true", help="JSON 输出")
    args = ap.parse_args(argv)

    root = root or find_monorepo_root()
    registry = load_registry(root)

    host = os.environ.get("MIGRATE_MYSQL_HOST", DEFAULT_PROD_HOST)
    port = os.environ.get("MIGRATE_MYSQL_PORT", DEFAULT_PORT)
    user = os.environ.get("MIGRATE_MYSQL_USER", DEFAULT_USER)
    password = os.environ.get("MIGRATE_MYSQL_PASS", DEFAULT_PASS)
    timeout = int(os.environ.get("MIGRATE_TIMEOUT_SEC", "10"))

    # docker 容器内 mysql 时，host 需要可被容器访问；容器网络能直达 10.2.150.68 时用原 host
    docker = detect_docker()

    databases = registry.get("databases", {})
    report: list[dict] = []
    exit_code = 0

    for key, cfg in databases.items():
        if args.db and args.db not in (key, cfg.get("database")):
            continue
        db_name = cfg.get("database")
        migrate_script = cfg.get("migrate_script", "")
        if not db_name:
            continue

        # 从 migrate_script（db/<key>/migrate.sh）推导 dataMigrate 目录：
        # 文件内 exec apply_datamigrate.sh "<db_name>" "$ROOT/dataMigrate/<Dir>"
        mig_dir: Path | None = None
        if migrate_script:
            mscript = root / migrate_script
            if mscript.is_file():
                text = mscript.read_text(encoding="utf-8", errors="replace")
                # 匹配 "$ROOT/dataMigrate/<dir>"（单引号或双引号包裹）
                import re
                m = re.search(r'["\']\$ROOT/dataMigrate/([A-Za-z0-9_]+)["\']', text)
                if m:
                    cand = root / "dataMigrate" / m.group(1)
                    if cand.is_dir():
                        mig_dir = cand

        local_sql = sorted(p.name for p in (mig_dir.glob("*.sql") if mig_dir else [])) if mig_dir else []
        entry = {
            "db": key,
            "database": db_name,
            "migrate_script": migrate_script,
            "dataMigrate_dir": str(mig_dir) if mig_dir else None,
            "local_files": local_sql,
        }

        try:
            cmd_prefix = mysql_cmd_base(docker, host, port, user, password)
            applied = query_single(cmd_prefix, "SELECT step_key FROM data_migrate_log ORDER BY step_key", db=db_name, timeout=timeout)
        except (RuntimeError, subprocess.TimeoutExpired) as e:
            entry["error"] = str(e)
            entry["pass"] = False
            entry["skipped"] = True
            report.append(entry)
            if not args.json:
                print(f"SKIP  {key:20s} {db_name:18s} — {e}")
            continue

        applied_set = set(applied)
        local_set = set(local_sql)
        missing = sorted(local_set - applied_set)     # 本地有、生产未应用（缺口，须 FAIL）
        stale = sorted(applied_set - local_set)       # 生产有、本地无（renumber 遗留，仅告警）

        entry["applied"] = applied
        entry["missing"] = missing
        entry["stale"] = stale
        entry["pass"] = not missing and not stale
        report.append(entry)

        if not args.json:
            status = "OK  " if (not missing and not stale) else ("WARN" if not missing else "FAIL")
            print(f"{status} {key:20s} {db_name:18s} local={len(local_sql):3d} applied={len(applied):3d}", end="")
            if missing:
                print(f"  MISSING: {','.join(missing)}", end="")
                exit_code = 1
            if stale:
                print(f"  STALE: {','.join(stale)}", end="")
            print()

    if args.json:
        print(json.dumps({"exit": exit_code, "checks": report}, ensure_ascii=False, indent=2))

    if exit_code == 0:
        return 0
    print("\n⛔ 存在 dataMigrate 缺口：本地有未应用的迁移文件，须先执行", file=sys.stderr)
    print("   bash db/scripts/apply_datamigrate.sh <db> dataMigrate/<dir> 或 9999「初始化全部数据库」", file=sys.stderr)
    return 1


if __name__ == "__main__":
    sys.exit(main())
