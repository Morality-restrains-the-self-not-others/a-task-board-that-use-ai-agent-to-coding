"""prepare-ram-deploy.sh must assemble from daydaymoney-deploy, not symlink ram-work."""

from __future__ import annotations

import os
import subprocess
from pathlib import Path

PREPARE = Path(__file__).resolve().parents[1] / "prepare-ram-deploy.sh"


def test_prepare_source_has_no_runtime_symlinks_to_meta():
    text = PREPARE.read_text(encoding="utf-8")
    assert "RUNTIME_LINKS" not in text
    assert "ln -s \"$src\" \"$dest\"" not in text
    assert "layout-from-config-repo.sh" in text or "scripts/layout.sh" in text
    assert "clone_config_repo" in text or "git clone" in text


def test_prepare_default_deploy_root_is_home_daydaymoney():
    text = PREPARE.read_text(encoding="utf-8")
    assert "${DEPLOY_ROOT:-$HOME/bin/daydaymoney-deploy}" in text
    assert "/tmp/ram-deploy" not in text


def test_prepare_delegates_cutover_env_to_ssot_helper():
    """OPT-20260901-004: 不再自带 cutover.env heredoc，必须走 write-cutover-env.sh。"""
    text = PREPARE.read_text(encoding="utf-8")
    assert "write-cutover-env.sh" in text
    assert 'cat > "$DEPLOY_ROOT/cutover.env"' not in text
    assert 'cat > \"$DEPLOY_ROOT/cutover.env\"' not in text


def test_sibling_scripts_default_home_daydaymoney():
    scripts = Path(__file__).resolve().parents[1]
    mat = (scripts / "materialize-p4-root.sh").read_text(encoding="utf-8")
    assert "${DEPLOY_ROOT:-$HOME/bin/daydaymoney-deploy}" in mat
    assert "/tmp/ram-deploy" not in mat
    chk = (scripts / "check_p4_deploy_root.sh").read_text(encoding="utf-8")
    assert "${DEPLOY_ROOT:-$HOME/bin/daydaymoney-deploy}" in chk
    assert "/tmp/ram-deploy" not in chk
    promtail = (scripts / "runall-local-promtail.sh").read_text(encoding="utf-8")
    assert "${DEPLOY_ROOT:-$HOME/bin/daydaymoney-deploy}/logs" in promtail
    assert "/tmp/ram-deploy" not in promtail
