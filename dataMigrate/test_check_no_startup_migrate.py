#!/usr/bin/env python3
"""Unit tests for check_no_startup_migrate.py"""
from __future__ import annotations

from pathlib import Path

import check_no_startup_migrate as chk


def test_detects_migrate_inside_opendb():
    src = """
package main
func openDB(dsn string) error {
    db, _ = sql.Open("mysql", dsn)
    if err := runDataMigrate(repoRoot()); err != nil { return err }
    return nil
}
"""
    vs = chk.find_open_func_migrate_calls(src, Path("/tmp/ram-work/taskX/src/db.go"))
    assert any("runDataMigrate" in v for v in vs), vs


def test_clean_opendb_ok():
    src = """
package main
func openDB(dsn string) error {
    db, _ = sql.Open("mysql", dsn)
    return db.Ping()
}
func runDataMigrate(repoRoot string) error { return nil }
"""
    vs = chk.find_open_func_migrate_calls(src, Path("/tmp/ram-work/taskX/src/db.go"))
    assert vs == [], vs


def test_main_migrate_cli_allowed():
    src = """
package main
func main() {
    if len(os.Args) > 1 && os.Args[1] == "migrate" {
        runDataMigrateFromDir(dsn, root)
        return
    }
    openDB(dsn)
}
"""
    vs = chk.find_server_main_migrate_calls(src, Path("/tmp/ram-work/taskX/src/main.go"))
    assert vs == [], vs


def test_main_server_migrate_forbidden():
    src = """
package main
func main() {
    if err := runDataMigrateFromDir(dsn, root); err != nil { log.Fatal(err) }
    openDB(dsn)
}
"""
    vs = chk.find_server_main_migrate_calls(src, Path("/tmp/ram-work/taskX/src/main.go"))
    assert len(vs) >= 1, vs


if __name__ == "__main__":
    test_detects_migrate_inside_opendb()
    test_clean_opendb_ok()
    test_main_migrate_cli_allowed()
    test_main_server_migrate_forbidden()
    print("ok")
