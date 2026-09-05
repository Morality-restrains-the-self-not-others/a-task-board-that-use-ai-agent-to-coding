#!/usr/bin/env python3
"""Load conf/infra/mysql/config.yaml and print docker-mysql env exports.

SSOT: conf/infra/mysql/config.yaml. Consumed by dockerInfra/mysql/run.sh.
"""
from __future__ import annotations

import sys
from pathlib import Path

_SCRIPTS = Path(__file__).resolve().parents[2] / "runAll" / "scripts"
if str(_SCRIPTS) not in sys.path:
    sys.path.insert(0, str(_SCRIPTS))
from conf_local import overlay_conf_file  # noqa: E402

_DEFAULTS = {
    "maxConnections": 500,
    "innodbBufferPoolSize": "256M",
    "innodbLogFileSize": "128M",
    "innodbFlushLogAtTrxCommit": 2,
    "syncBinlog": 0,
    "skipLogBin": True,
}


def load_mysql_conf(path: str) -> dict:
    raw = overlay_conf_file(Path(path))
    if not isinstance(raw, dict):
        raise ValueError(f"mysql config must be a mapping: {path}")
    out = dict(_DEFAULTS)
    for key in _DEFAULTS:
        if key in raw and raw[key] is not None:
            out[key] = raw[key]
    return out


def export_lines(conf: dict) -> list[str]:
    skip = "1" if conf["skipLogBin"] else "0"
    return [
        f"MYSQL_MAX_CONNECTIONS={int(conf['maxConnections'])}",
        f"MYSQL_INNODB_BUFFER_POOL_SIZE={conf['innodbBufferPoolSize']}",
        f"MYSQL_INNODB_LOG_FILE_SIZE={conf['innodbLogFileSize']}",
        f"MYSQL_INNODB_FLUSH_LOG_AT_TRX_COMMIT={int(conf['innodbFlushLogAtTrxCommit'])}",
        f"MYSQL_SYNC_BINLOG={int(conf['syncBinlog'])}",
        f"MYSQL_SKIP_LOG_BIN={skip}",
    ]


def main() -> int:
    if len(sys.argv) != 2:
        print("usage: load_mysql_conf.py <conf/infra/mysql/config.yaml>", file=sys.stderr)
        return 2
    for line in export_lines(load_mysql_conf(sys.argv[1])):
        print(f"export {line}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
