#!/usr/bin/env python3
"""Print host from conf/infra/*/config.yaml (expand ${INFRA_HOST:-default})."""
from __future__ import annotations

import os
import re
import sys
from pathlib import Path

_SCRIPTS = Path(__file__).resolve().parents[2] / "runAll" / "scripts"
if str(_SCRIPTS) not in sys.path:
    sys.path.insert(0, str(_SCRIPTS))
from conf_local import overlay_conf_file  # noqa: E402

_ENV_DEFAULT = re.compile(r"^\$\{INFRA_HOST:-([^}]*)\}$")


def infra_host_from_conf(path: str, environ: dict[str, str] | None = None) -> str:
    env = environ if environ is not None else os.environ
    raw = overlay_conf_file(Path(path))
    if not isinstance(raw, dict):
        raise ValueError(f"infra config must be a mapping: {path}")
    host = str(raw.get("host") or "").strip()
    if not host:
        raise ValueError(f"missing host in {path}")
    m = _ENV_DEFAULT.match(host)
    if m:
        override = (env.get("INFRA_HOST") or "").strip()
        return override if override else m.group(1)
    return host


def main() -> int:
    if len(sys.argv) != 2:
        print("usage: infra_conf_host.py <conf/infra/<app>/config.yaml>", file=sys.stderr)
        return 2
    print(infra_host_from_conf(sys.argv[1]))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
