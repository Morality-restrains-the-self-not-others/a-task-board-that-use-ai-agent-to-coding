#!/usr/bin/env python3
"""Self-test for scripts/seed-daydaymoney-deploy.sh."""

from __future__ import annotations

import os
import subprocess
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
SEED = ROOT / "scripts" / "seed-daydaymoney-deploy.sh"


def test_excludes_local_yaml() -> None:
    with tempfile.TemporaryDirectory() as raw:
        conf = Path(raw) / "conf"
        conf.mkdir()
        (conf / "base.yaml").write_text("scheme: https\n", encoding="utf-8")
        (conf / "config.local.yaml").write_text("password: hunter2-prod\n", encoding="utf-8")
        (conf / "gitLabRootPwd.md").write_text("Password: x\n", encoding="utf-8")
        dest = Path(raw) / "seed"
        env = os.environ.copy()
        env["CONF_SRC"] = str(conf)
        subprocess.run(["bash", str(SEED), str(dest)], check=True, env=env)
        assert (dest / "envs" / "current" / "conf" / "base.yaml").is_file()
        assert not (dest / "envs" / "current" / "conf" / "config.local.yaml").exists()
        assert not (dest / "envs" / "current" / "conf" / "gitLabRootPwd.md").exists()


def main() -> int:
    try:
        test_excludes_local_yaml()
        print("ok test_excludes_local_yaml")
        return 0
    except Exception as exc:
        print(f"FAIL test_excludes_local_yaml: {exc}")
        return 1


if __name__ == "__main__":
    sys.exit(main())
