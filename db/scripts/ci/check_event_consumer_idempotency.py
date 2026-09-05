#!/usr/bin/env python3
"""事件消费者必须幂等消费（元规则 49 / ADR-0015）。

Kafka at-least-once 下，重复投递不得产生第二次副作用。消费必须走
taskEvents 共享 runner（IdempotentDispatchService），幂等键须与业务重复
边界同粒度。

用法:
  python3 db/scripts/ci/check_event_consumer_idempotency.py
"""
from __future__ import annotations

import argparse
import os
import re
import sys
from dataclasses import dataclass
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]

KAFKA_READER_RE = re.compile(
    r"kafka\.NewReader\s*\(|(?:\A|[^.\w])kafka\.Reader\s*\{|&kafka\.Reader\b"
)
EVENTBIN_CALL_RE = re.compile(r"eventbin\.(RunIntent|Run)\s*\(")
GENERIC_FIELDS_RE = re.compile(
    r"for\s+_,\s+field\s+:=\s+range\s+\[\]string\{([^}]+)\}",
    re.MULTILINE,
)
STRING_LIT_RE = re.compile(r'"([^"]+)"')

SKIP_DIR_PARTS = frozenset(
    {
        "third_party",
        "vendor",
        "node_modules",
        "gitlab-ce",
        "gitlab_home",
        ".git",
    }
)
ALLOWED_KAFKA_READER_PREFIXES = (
    "taskEvents/broker/",
    "taskEvents/consumer/",
)
REQUIRED_KEY_TESTS = (
    "TestIdempotencyKeyFromEnvelopeCloudServerStoppedIgnoresCompanyID",
    "TestIdempotencyKeyFromEnvelopeTaskCreatedIgnoresUserID",
)
META_FILES = (
    ".ai/01_project_constraints/54_event_consumer_idempotency.md",
    ".cursor/rules/event-consumer-idempotency.mdc",
    "docs/adr/0015-event-consumer-idempotency.md",
)

SERVICE_TOP_EXACT = frozenset({"go_relayToTrae", "go_run_container"})


@dataclass(frozen=True)
class EventbinCall:
    kind: str
    args: list[str]


def _is_service_rel(rel: str) -> bool:
    top = rel.split("/", 1)[0]
    if top.endswith("-wt"):
        return False
    if top in SERVICE_TOP_EXACT:
        return True
    return top.startswith("task")


def _skipped_path(rel: str, parts: tuple[str, ...]) -> bool:
    if rel.endswith("_test.go"):
        return True
    if any(p in SKIP_DIR_PARTS for p in parts):
        return True
    return any(rel.startswith(prefix) for prefix in ALLOWED_KAFKA_READER_PREFIXES)


def classify_kafka_reader(rel: str, src: str) -> str:
    """violation | ok"""
    parts = tuple(Path(rel).parts)
    if _skipped_path(rel, parts):
        return "ok"
    if not _is_service_rel(rel):
        return "ok"
    if not KAFKA_READER_RE.search(src):
        return "ok"
    return "violation"


def _split_top_level_args(inside: str) -> list[str]:
    args: list[str] = []
    buf: list[str] = []
    depth = 0
    in_str = False
    i = 0
    while i < len(inside):
        c = inside[i]
        if in_str:
            buf.append(c)
            if c == "\\" and i + 1 < len(inside):
                buf.append(inside[i + 1])
                i += 2
                continue
            if c == '"':
                in_str = False
            i += 1
            continue
        if c == '"':
            in_str = True
            buf.append(c)
            i += 1
            continue
        if c in "([{":
            depth += 1
            buf.append(c)
            i += 1
            continue
        if c in ")]}":
            depth -= 1
            buf.append(c)
            i += 1
            continue
        if c == "," and depth == 0:
            piece = "".join(buf).strip()
            if piece:
                args.append(piece)
            buf = []
            i += 1
            continue
        buf.append(c)
        i += 1
    piece = "".join(buf).strip()
    if piece:
        args.append(piece)
    return args


def extract_eventbin_calls(src: str) -> list[EventbinCall]:
    calls: list[EventbinCall] = []
    for m in EVENTBIN_CALL_RE.finditer(src):
        open_idx = m.end() - 1  # points at '('
        depth = 0
        i = open_idx
        while i < len(src):
            c = src[i]
            if c == "(":
                depth += 1
            elif c == ")":
                depth -= 1
                if depth == 0:
                    inside = src[open_idx + 1 : i]
                    args = _split_top_level_args(inside)
                    calls.append(EventbinCall(kind=m.group(1), args=args))
                    break
            i += 1
    return calls


def classify_eventbin_src(rel: str, src: str) -> list[str]:
    _ = rel
    hits: list[str] = []
    for call in extract_eventbin_calls(src):
        need = 4 if call.kind == "RunIntent" else 3
        if len(call.args) < need:
            hits.append("missing_keyfn")
            continue
        last = call.args[need - 1].strip()
        if last == "nil":
            hits.append("nil_keyfn")
    return hits


def check_generic_key_field_order(src: str) -> tuple[bool, str]:
    m = GENERIC_FIELDS_RE.search(src)
    if not m:
        return False, "generic []string field loop not found"
    fields = STRING_LIT_RE.findall(m.group(1))
    if "event_id" not in fields:
        return False, "event_id missing from generic field order"
    idx = {name: i for i, name in enumerate(fields)}
    event_i = idx["event_id"]
    for coarse in ("user_id", "company_id", "tenant_id"):
        if coarse in idx and idx[coarse] < event_i:
            return False, f"{coarse} appears before event_id"
        if "task_id" in idx and coarse in idx and idx[coarse] < idx["task_id"]:
            return False, f"{coarse} appears before task_id"
    return True, ""


def check_dispatch_idempotency(src: str) -> list[str]:
    missing: list[str] = []
    if ".Seen(" not in src and "Idempotency.Seen" not in src:
        missing.append("Seen")
    if ".Mark(" not in src and "Idempotency.Mark" not in src:
        missing.append("Mark")
    if "idempotency skip" not in src:
        missing.append("skip_log")
    return missing


def check_runner_wires_store(src: str) -> list[str]:
    missing: list[str] = []
    if "IdempotentDispatchService" not in src:
        missing.append("IdempotentDispatchService")
    if "NewMemoryStore" not in src and "IdempotencyStorePort" not in src:
        missing.append("idempotency_store")
    return missing


def check_key_regression_tests(root: Path) -> list[str]:
    path = root / "taskEvents" / "consumer" / "key_test.go"
    if not path.is_file():
        return ["taskEvents/consumer/key_test.go missing"]
    text = path.read_text(encoding="utf-8", errors="replace")
    return [name for name in REQUIRED_KEY_TESTS if name not in text]


def scan_kafka_reader_violations(root: Path) -> list[str]:
    hits: list[str] = []
    for dirpath, dirnames, filenames in os.walk(root):
        dirnames[:] = [d for d in dirnames if d not in SKIP_DIR_PARTS]
        for name in filenames:
            if not name.endswith(".go"):
                continue
            path = Path(dirpath) / name
            try:
                rel = str(path.relative_to(root))
            except ValueError:
                continue
            src = path.read_text(encoding="utf-8", errors="replace")
            if classify_kafka_reader(rel, src) == "violation":
                hits.append(rel)
    return hits


def scan_eventbin_violations(root: Path) -> list[tuple[str, str]]:
    hits: list[tuple[str, str]] = []
    cmd = root / "taskEvents" / "cmd"
    if not cmd.is_dir():
        return hits
    for path in cmd.rglob("main.go"):
        rel = str(path.relative_to(root))
        src = path.read_text(encoding="utf-8", errors="replace")
        for kind in classify_eventbin_src(rel, src):
            hits.append((rel, kind))
    return hits


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.parse_args(argv)

    errors: list[str] = []

    for rel in META_FILES:
        if not (ROOT / rel).is_file():
            errors.append(f"missing {rel}")

    for rel in scan_kafka_reader_violations(ROOT):
        errors.append(
            f"{rel}: Kafka Reader/NewReader outside taskEvents broker/consumer "
            "(consume via eventbin.RunIntent + IdempotentDispatchService)"
        )

    for rel, kind in scan_eventbin_violations(ROOT):
        if kind == "missing_keyfn":
            errors.append(f"{rel}: eventbin.RunIntent/Run missing idempotency keyFn")
        elif kind == "nil_keyfn":
            errors.append(f"{rel}: eventbin keyFn must not be nil")

    key_go = ROOT / "taskEvents" / "consumer" / "key.go"
    if key_go.is_file():
        ok, reason = check_generic_key_field_order(
            key_go.read_text(encoding="utf-8", errors="replace")
        )
        if not ok:
            errors.append(f"taskEvents/consumer/key.go: {reason}")
    else:
        errors.append("taskEvents/consumer/key.go missing")

    dispatch = ROOT / "taskEvents" / "domain" / "dispatch_service.go"
    if dispatch.is_file():
        missing = check_dispatch_idempotency(
            dispatch.read_text(encoding="utf-8", errors="replace")
        )
        if missing:
            errors.append(
                "taskEvents/domain/dispatch_service.go missing "
                + ",".join(missing)
            )
    else:
        errors.append("taskEvents/domain/dispatch_service.go missing")

    runner = ROOT / "taskEvents" / "consumer" / "runner.go"
    if runner.is_file():
        missing = check_runner_wires_store(
            runner.read_text(encoding="utf-8", errors="replace")
        )
        if missing:
            errors.append(
                "taskEvents/consumer/runner.go missing " + ",".join(missing)
            )
    else:
        errors.append("taskEvents/consumer/runner.go missing")

    for item in check_key_regression_tests(ROOT):
        errors.append(item)

    if errors:
        for e in errors:
            print(f"ERROR {e}")
        print(f"event-consumer-idempotency: {len(errors)} violation(s)")
        return 1
    print("ok: event consumers use shared idempotent dispatch")
    return 0


if __name__ == "__main__":
    sys.exit(main())
