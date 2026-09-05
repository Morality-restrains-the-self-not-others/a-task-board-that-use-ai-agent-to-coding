"""ADR-0052: task-auth migrate/init prefer last-good ELF on a source-less deploy root."""

from __future__ import annotations

from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]


def test_task_auth_migrate_prefers_deploy_elf_before_go_run():
    text = (ROOT / "db" / "task-auth" / "migrate.sh").read_text(encoding="utf-8")
    elf = text.find('exec "$ROOT/bin/taskAuth" migrate')
    go_run = text.find("go run ./src migrate")
    assert elf >= 0, "migrate.sh must exec $ROOT/bin/taskAuth migrate on P4"
    assert go_run >= 0, "source-tree fallback go run must remain"
    assert elf < go_run


def test_task_auth_init_prefers_deploy_elf_before_run_sh():
    text = (ROOT / "db" / "task-auth" / "init.sh").read_text(encoding="utf-8")
    elf = text.find('exec "$ROOT/bin/taskAuth" bootstrap-admin')
    run_sh = text.find('bash "$ROOT/taskAuth/run.sh" bootstrap-admin')
    assert elf >= 0, "init.sh must exec $ROOT/bin/taskAuth bootstrap-admin on P4"
    assert run_sh >= 0
    assert elf < run_sh


def test_materialize_p4_layouts_from_config_repo():
    text = (ROOT / "runAll" / "scripts" / "materialize-p4-root.sh").read_text(encoding="utf-8")
    assert "layout recipes from config repo" in text
    assert "COPY_ELFS_FROM_META" in text
    assert "the source ram-work tree" in text
    assert 'META/dataMigrate/' not in text
