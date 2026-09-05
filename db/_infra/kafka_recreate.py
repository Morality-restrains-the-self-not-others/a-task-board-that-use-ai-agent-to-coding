#!/usr/bin/env python3
"""Delete and recreate Kafka topics for dev reset.

Topic names are derived from taskEvents/config/config.go EventTopic mapping.
No longer depends on core.kafka.config from Saas_project (which no longer exists).
"""

from __future__ import annotations

import sys
import time
from pathlib import Path


# All Kafka topics derived from Go EventTopic mapping (taskEvents/config/config.go).
# Keep in sync with: taskEvents/config/config.go EventTopic()
KAFKA_TOPICS = sorted(set([
    "ai-assistant-reply-completed",
    "billing-transaction-created",
    "cloud-platform-authorization-created",
    "cloud-server-start-auto",
    "cloud-server-started",
    "cloud-server-stopped",
    "comment-container-binding-advanced",
    "company-created",
    "container-instruction-idle-cleared",
    "container-instruction-idle-marked",
    "container-migrate-await-ready",
    "email-sent",
    "feature-params-initialized",
    "invitation-created",
    "job-step-full-archived",
    "layer-graph-snapshot-persisted",
    "project-updated",
    "registration-invite-code-issued",
    "registration-invite-code-redeemed",
    "registration-invite-policy-updated",
    "relay-lifecycle",
    "sse-message",
    "step-full-cos-config-updated",
    "task-comment-image-mentioned",
    "task-completed",
    "task-created",
    "task-deleted",
    "task-graceful-shutdown-await",
    "task-status-changed",
    "user-activated",
    "user-created",
    "user-impersonation-started",
    "user-impersonation-stopped",
    "user-inbox-message-created",
    "user-logged-in",
    "workspace-created",
    "workspace-machine-idle",
]))

# Dead Letter Topics — one per business topic, for offline inspection of
# messages that permanently failed or exhausted retries.
# Naming convention: {topic}-dlt
# Keep in sync with: taskEvents/config/config.go DeadLetterTopic()
DLT_TOPICS = sorted(set([f"{t}-dlt" for t in KAFKA_TOPICS]))


def _monorepo_root() -> Path:
    return Path(__file__).resolve().parents[2]


def _resolve_template(value: str) -> str:
    """Resolve ${VAR:-default} patterns using environment variables."""
    import os
    import re

    def _replace(match):
        var_name = match.group(1)
        default_val = match.group(2) if match.lastindex >= 2 and match.group(2) is not None else ""
        return os.environ.get(var_name, default_val)

    return re.sub(r'\$\{(\w+)(?::-([^}]*))?\}', _replace, value)


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


def _wait_for_topics_gone(admin, topics: list[str], timeout: float = 60.0) -> bool:
    """Poll until none of the given topics appear in the cluster metadata."""
    deadline = time.monotonic() + timeout
    target = set(topics)
    while time.monotonic() < deadline:
        try:
            metadata = admin.list_topics(timeout=5)
            existing = set(metadata.topics.keys())
            still_there = target & existing
            if not still_there:
                return True
            print(f"  {len(still_there)} topic(s) still present, waiting...")
        except Exception:
            pass
        time.sleep(2)
    return False


def _report_kafka_connectors(port: str = "9092") -> None:
    """Report processes holding a connection to the Kafka broker.

    Kafka defers topic deletion while any consumer/producer still holds an
    active fetch/produce connection (OPT-20260810-002: zombie consumers
    blocked clear-databases for 180s+). When deletion times out this
    diagnostic names the holders so the operator can decide whether they
    are unmanaged leftovers to kill. Read-only; never kills anything.
    """
    import shutil
    import subprocess

    if shutil.which("ss"):
        cmd = ["ss", "-tnp", "state", "established", f"( sport = :{port} or dport = :{port} )"]
        out = subprocess.run(cmd, capture_output=True, text=True, timeout=10).stdout
        lines = [ln for ln in out.splitlines() if ln.strip() and not ln.startswith("State")]
        if not lines:
            print("  (no established connections to Kafka broker detected)")
            return
        print(f"  processes with active connections to kafka:{port} (may block topic deletion):")
        for ln in lines:
            print(f"    {ln.strip()}")
        return
    if shutil.which("lsof"):
        out = subprocess.run(
            ["lsof", "-nP", "-i", f":{port}"], capture_output=True, text=True, timeout=10
        ).stdout
        lines = [ln for ln in out.splitlines() if ln.strip() and not ln.startswith("COMMAND")]
        if not lines:
            print("  (no open files on kafka port detected)")
            return
        print(f"  processes with open files on kafka:{port} (may block topic deletion):")
        for ln in lines:
            print(f"    {ln.strip()}")
        return
    print("  (ss/lsof unavailable — cannot enumerate Kafka connection holders)")


def _create_topics_with_retry(admin, new_topics, max_wait: float = 60.0) -> bool:
    """Create topics, retrying if any are still marked for deletion."""
    from confluent_kafka.admin import NewTopic

    pending = {t.topic: t for t in new_topics}
    deadline = time.monotonic() + max_wait

    while pending and time.monotonic() < deadline:
        retry = []
        topics_list = [NewTopic(name, num_partitions=1, replication_factor=1)
                       for name in pending]
        try:
            fs = admin.create_topics(topics_list)
            for topic_name, f in fs.items():
                try:
                    f.result()
                    print(f"  created: {topic_name}")
                    del pending[topic_name]
                except Exception as exc:
                    msg = str(exc)
                    # TOPIC_ALREADY_EXISTS here means the broker's internal
                    # async deletion is still in progress (the topic vanished
                    # from list_topics but is not yet gone): retry instead of
                    # failing the whole clear-db run.
                    if "marked for deletion" in msg or "already exists" in msg.lower():
                        print(f"  {topic_name} still deleting, will retry...")
                        retry.append(topic_name)
                    else:
                        print(f"  create {topic_name} failed: {exc}", file=sys.stderr)
                        return False
        except Exception as exc:
            print(f"create topics batch failed: {exc}", file=sys.stderr)
            return False

        if not retry:
            break
        time.sleep(3)

    if pending:
        print(f"timed out waiting to create: {list(pending.keys())}", file=sys.stderr)
        return False
    return True


def main() -> int:
    root = _monorepo_root()
    topics = KAFKA_TOPICS + DLT_TOPICS
    if not topics:
        print("no topics registered; skip kafka recreate")
        return 0

    bootstrap = _bootstrap_servers(root)
    from confluent_kafka.admin import AdminClient, NewTopic
    admin = AdminClient({"bootstrap.servers": bootstrap})

    # Discover existing topics (including those marked for deletion).
    try:
        metadata = admin.list_topics(timeout=10)
        existing = set(metadata.topics.keys())
    except Exception as exc:
        print(f"list topics failed: {exc}", file=sys.stderr)
        return 1

    to_delete = [t for t in topics if t in existing]
    if to_delete:
        print(f"deleting {len(to_delete)} topic(s): {to_delete}")
        fs = admin.delete_topics(to_delete, operation_timeout=30)
        for topic, f in fs.items():
            try:
                f.result()
                print(f"  delete accepted: {topic}")
            except Exception as exc:
                msg = str(exc)
                if "marked for deletion" in msg:
                    print(f"  already deleting: {topic}")
                else:
                    print(f"  delete {topic} failed: {exc}", file=sys.stderr)
                    return 1

    # Always wait for all target topics to be gone (they may be deleting
    # from a previous run even if not in current list_topics result).
    print("waiting for all target topics to be fully removed...")
    if not _wait_for_topics_gone(admin, topics):
        print("timed out waiting for topic deletion", file=sys.stderr)
        # Deletion is deferred while any process holds an active broker
        # connection — name the holders (often zombie consumers) so the
        # operator can decide whether to kill unmanaged leftovers.
        _report_kafka_connectors()
        return 1
    print("all target topics confirmed gone")

    # Recreate with retry for topics still marked for deletion.
    new_topics = [NewTopic(t, num_partitions=1, replication_factor=1) for t in topics]
    print(f"creating {len(new_topics)} topic(s)...")
    if not _create_topics_with_retry(admin, new_topics):
        return 1

    print("all kafka topics recreated successfully")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
