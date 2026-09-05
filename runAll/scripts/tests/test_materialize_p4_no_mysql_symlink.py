"""materialize-p4-root.sh must not bind MySQL data to the source ram-work tree."""

from __future__ import annotations

from pathlib import Path

SCRIPT = Path(__file__).resolve().parents[1] / "materialize-p4-root.sh"


def test_materialize_does_not_symlink_mysql_to_ram_work():
    text = SCRIPT.read_text(encoding="utf-8")
    assert "mysql data -> live host data dir" not in text
    assert "dockerInfra/mysql/data" not in text or "ln -sfn" not in text
    assert "layout-from-config-repo.sh" in text or "scripts/layout.sh" in text
