#!/usr/bin/env python3
"""Delete junk kafka-go-* topics polluting the dev cluster metadata.

Root cause: vendored kafka-go library integration tests (conn_test.go's
makeTopic() -> "kafka-go-%016x") were run by the random unit-test sweep
(pre-commit gate + nightly_test_sweep.py) against the live broker. Each run
auto-created dozens of `kafka-go-<16hex>` topics via DialLeader's
AllowAutoTopicCreation and never cleaned them up. 3071 accumulated.

See: docs/superpowers/specs/2026-08-10-kafka-go-topic-pollution-fix-design.md

Usage:
    python3 db/_infra/kafka_cleanup_junk_topics.py            # dry-run (default)
    python3 db/_infra/kafka_cleanup_junk_topics.py --confirm DELETE_JUNK
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

# Only delete topics matching the kafka-go test-suite naming pattern.
JUNK_TOPIC_RE = re.compile(r"^kafka-go-[0-9a-f]{16}$")


def _monorepo_root() -> Path:
    return Path(__file__).resolve().parents[2]


def _resolve_template(value: str) -> str:
    import os
    import re

    def _replace(match):
        var_name = match.group(1)
        default_val = match.group(2) if match.lastindex >= 2 and match.group(2) is not None else ""
        return os.environ.get(var_name, default_val)

    return re.sub(r"\$\{(\w+)(?::-([^}]*))?\}", _replace, value)


def _bootstrap_servers(root: Path) -> str:
    import sys
    import yaml

    path = root / "conf" / "infra" / "docker-infra" / "config.yaml"
    if path.exists():
        scripts = str(root / "runAll" / "scripts")
        if scripts not in sys.path:
            sys.path.insert(0, scripts)
        from conf_local import overlay_conf_file

        cfg = overlay_conf_file(path)
        raw = _resolve_template(yaml.dump(cfg, allow_unicode=True))
        cfg = yaml.safe_load(raw) or {}
        kafka = cfg.get("kafka") if isinstance(cfg.get("kafka"), dict) else {}
        return str(kafka.get("bootstrapServers", "localhost:9093"))
    return "localhost:9093"


def collect_junk_topics(admin) -> list[str]:
    """Return all topic names matching the junk pattern, sorted."""
    metadata = admin.list_topics(timeout=15)
    junk = [t for t in metadata.topics.keys() if JUNK_TOPIC_RE.match(t)]
    return sorted(junk)


def delete_topics(admin, topics: list[str]) -> tuple[int, list[str]]:
    """Delete topics; return (ok_count, failures)."""
    failures: list[str] = []
    ok_count = 0
    fs = admin.delete_topics(topics, operation_timeout=30)
    for topic, f in fs.items():
        try:
            f.result()
            ok_count += 1
        except Exception as exc:
            msg = str(exc)
            if "marked for deletion" in msg:
                ok_count += 1  # already deleting; counts as accepted
            else:
                failures.append(f"{topic}: {exc}")
    return ok_count, failures


def main() -> int:
    ap = argparse.ArgumentParser(description="Delete junk kafka-go-* topics")
    ap.add_argument("--confirm", default="", help="must be DELETE_JUNK to actually delete")
    args = ap.parse_args()

    root = _monorepo_root()
    bootstrap = _bootstrap_servers(root)
    from confluent_kafka.admin import AdminClient

    admin = AdminClient({"bootstrap.servers": bootstrap})

    try:
        junk = collect_junk_topics(admin)
    except Exception as exc:
        print(f"list topics failed: {exc}", file=sys.stderr)
        return 1

    print(f"junk topics matched: {len(junk)}")
    if not junk:
        print("nothing to clean — cluster is already clean")
        return 0

    for t in junk[:5]:
        print(f"  sample: {t}")
    if len(junk) > 5:
        print(f"  ... and {len(junk) - 5} more")

    if args.confirm != "DELETE_JUNK":
        print("\nDRY-RUN: no topics deleted. Re-run with --confirm DELETE_JUNK to execute.")
        return 0

    print(f"\ndeleting {len(junk)} junk topic(s)...")
    ok, failures = delete_topics(admin, junk)
    print(f"delete accepted: {ok}, failed: {len(failures)}")
    for f in failures:
        print(f"  FAIL {f}", file=sys.stderr)

    remaining = collect_junk_topics(admin)
    print(f"remaining junk topics in metadata: {len(remaining)}")
    if remaining:
        print("  (delete is async at broker level — they disappear from metadata once gone)")
    return 1 if failures else 0


if __name__ == "__main__":
    raise SystemExit(main())
