#!/usr/bin/env python3
"""P4 layout must pre-create taskGateway/logs as 0777.

Docker bind-mount of a missing path creates root:root 755; APISIX uid 636
then cannot open error.log (nginx emerg, container Exited 1).
"""
from pathlib import Path

SCRIPT = Path(__file__).resolve().parent / "layout-from-config-repo.sh"


def test_materialize_precreates_writable_gateway_logs():
    text = SCRIPT.read_text(encoding="utf-8")
    assert "taskGateway/logs" in text
    assert "chmod 777" in text


if __name__ == "__main__":
    test_materialize_precreates_writable_gateway_logs()
    print("PASS test_materialize_precreates_writable_gateway_logs")
