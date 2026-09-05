"""OPT-20260901-013: 022 SQL 邮箱占位符由 conf 渲染。"""
from pathlib import Path

import bootstrap_admin_email as bae

_HERE = Path(__file__).resolve().parent


def test_022_uses_placeholder_not_literal_email():
    sql = (_HERE / "022_seed_bootstrap_admin.sql").read_text(encoding="utf-8")
    assert bae.PLACEHOLDER in sql
    assert "'author@example.com'" not in sql
    assert sql.count(bae.PLACEHOLDER) == 1


def test_render_sql_substitutes_email():
    src = "VALUES ('email', '__BOOTSTRAP_ADMIN_EMAIL__', 'hash')"
    out = bae.render_sql(src, "ops@example.com")
    assert out == "VALUES ('email', 'ops@example.com', 'hash')"
    assert bae.PLACEHOLDER not in out


def test_render_sql_rejects_quote_in_email():
    src = f"id='{bae.PLACEHOLDER}'"
    try:
        bae.render_sql(src, "bad'admin@example.com")
    except ValueError as e:
        assert "SQL-unsafe" in str(e)
    else:
        raise AssertionError("expected ValueError")


def test_render_sql_noop_without_placeholder():
    src = "SELECT 1;"
    assert bae.render_sql(src, "a@b.com") == src


def test_cli_render_matches_conf_email():
    import subprocess
    import sys

    email = bae.load_bootstrap_admin_email()
    proc = subprocess.run(
        [sys.executable, str(_HERE / "bootstrap_admin_email.py"), "render", str(_HERE / "022_seed_bootstrap_admin.sql")],
        check=True,
        capture_output=True,
        text=True,
    )
    assert email in proc.stdout
    assert bae.PLACEHOLDER not in proc.stdout


def test_apply_datamigrate_wires_renderer():
    root = bae.find_repo_root()
    sh = (root / "db" / "scripts" / "apply_datamigrate.sh").read_text(encoding="utf-8")
    assert "bootstrap_admin_email.py" in sh
    assert "render_sql_file" in sh


def test_loader_uses_conf_local_overlay():
    src = (_HERE / "bootstrap_admin_email.py").read_text(encoding="utf-8")
    assert "overlay_conf_file" in src


def test_nested_bootstrap_admin_overlay_deep_merged(tmp_path):
    """OPT-20260901-014: conf-local nested bootstrapAdmin keys must survive.

    Shallow dict.update would let a conf-local skeleton replace the whole
    bootstrapAdmin map and drop nested keys; overlay_conf_file deep-merges so
    the conf-local email wins while unrelated nested keys are preserved.
    """
    conf_dir = tmp_path / "conf" / "auth" / "task-auth"
    conf_dir.mkdir(parents=True)
    (tmp_path / "conf" / "base.yaml").write_text("{}", encoding="utf-8")
    (conf_dir / "config.yaml").write_text(
        "bootstrapAdmin:\n"
        "  email: skeleton@example.com\n"
        "  smtp:\n"
        "    port: 465\n",
        encoding="utf-8",
    )
    loc_dir = tmp_path / "conf-local" / "auth" / "task-auth"
    loc_dir.mkdir(parents=True)
    (loc_dir / "config.yaml").write_text(
        "bootstrapAdmin:\n"
        "  email: overlay@example.com\n"
        "  smtp:\n"
        "    host: smtp.overlay.example.com\n",
        encoding="utf-8",
    )

    email = bae.load_bootstrap_admin_email(tmp_path)
    assert email == "overlay@example.com"

    # Direct check that nested keys deep-merge instead of being dropped.
    scripts = (_HERE / ".." / ".." / "runAll" / "scripts").resolve()
    import sys

    if str(scripts) not in sys.path:
        sys.path.insert(0, str(scripts))
    from conf_local import overlay_conf_file

    cfg = overlay_conf_file(tmp_path / "conf" / "auth" / "task-auth" / "config.yaml")
    ba = cfg["bootstrapAdmin"]
    assert ba["email"] == "overlay@example.com"
    assert ba["smtp"]["port"] == 465, "skeleton nested key must survive deep merge"
    assert ba["smtp"]["host"] == "smtp.overlay.example.com"
