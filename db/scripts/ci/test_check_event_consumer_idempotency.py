#!/usr/bin/env python3
"""Self-test for check_event_consumer_idempotency (no pytest required)."""

from __future__ import annotations

import importlib.util
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
CHECKER = ROOT / "db" / "scripts" / "ci" / "check_event_consumer_idempotency.py"
CONSTRAINT = ROOT / ".ai" / "01_project_constraints" / "54_event_consumer_idempotency.md"
CURSOR_RULE = ROOT / ".cursor" / "rules" / "event-consumer-idempotency.mdc"
ADR = ROOT / "docs" / "adr" / "0015-event-consumer-idempotency.md"


def _load():
    spec = importlib.util.spec_from_file_location("check_event_consumer_idempotency", CHECKER)
    assert spec and spec.loader
    mod = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = mod
    spec.loader.exec_module(mod)
    return mod


def test_extract_runintent_four_args() -> None:
    mod = _load()
    src = (
        'eventbin.RunIntent("company_created", "1_set", h, '
        "consumer.IdempotencyKeyFromEnvelope)\n"
    )
    calls = mod.extract_eventbin_calls(src)
    assert len(calls) == 1
    assert calls[0].kind == "RunIntent"
    assert len(calls[0].args) == 4
    assert "IdempotencyKeyFromEnvelope" in calls[0].args[3]


def test_extract_runintent_multiline_trailing_comma() -> None:
    mod = _load()
    src = """
	eventbin.RunIntent(
		"task_created",
		"2_create_task_post",
		h,
		consumer.IdempotencyKeyFromEnvelope,
	)
"""
    calls = mod.extract_eventbin_calls(src)
    assert len(calls) == 1
    assert len(calls[0].args) == 4
    assert calls[0].args[3].endswith("IdempotencyKeyFromEnvelope")


def test_runintent_missing_keyfn_is_violation() -> None:
    mod = _load()
    src = 'eventbin.RunIntent("x", "1_y", handler)\n'
    hits = mod.classify_eventbin_src("taskEvents/cmd/x/1_y/main.go", src)
    assert hits == ["missing_keyfn"]


def test_runintent_nil_keyfn_is_violation() -> None:
    mod = _load()
    src = 'eventbin.RunIntent("x", "1_y", handler, nil)\n'
    hits = mod.classify_eventbin_src("taskEvents/cmd/x/1_y/main.go", src)
    assert hits == ["nil_keyfn"]


def test_runintent_with_keyfn_ok() -> None:
    mod = _load()
    src = (
        'eventbin.RunIntent("x", "1_y", handler, '
        "consumer.IdempotencyKeyFromEnvelope)\n"
    )
    assert mod.classify_eventbin_src("taskEvents/cmd/x/1_y/main.go", src) == []


def test_kafka_reader_in_business_service_is_violation() -> None:
    mod = _load()
    src = "r := kafka.NewReader(kafka.ReaderConfig{Topic: t})\n"
    kind = mod.classify_kafka_reader("taskBill/src/worker.go", src)
    assert kind == "violation"


def test_kafka_reader_in_taskevents_broker_is_ok() -> None:
    mod = _load()
    src = "kb.readers = append(kb.readers, kafka.NewReader(kafka.ReaderConfig{}))\n"
    kind = mod.classify_kafka_reader("taskEvents/broker/kafka.go", src)
    assert kind == "ok"


def test_kafka_reader_in_third_party_and_test_is_ok() -> None:
    mod = _load()
    src = "r := kafka.NewReader(kafka.ReaderConfig{})\n"
    assert mod.classify_kafka_reader("taskAuth/third_party/kafka-go/logger.go", src) == "ok"
    assert mod.classify_kafka_reader("taskBill/src/worker_test.go", src) == "ok"


def test_generic_key_order_rejects_user_id_before_task_id() -> None:
    mod = _load()
    bad = 'for _, field := range []string{"event_id", "user_id", "task_id", "company_id"} {'
    ok, reason = mod.check_generic_key_field_order(bad)
    assert not ok
    assert "user_id" in reason


def test_generic_key_order_accepts_task_id_before_tenant_fields() -> None:
    mod = _load()
    good = (
        'for _, field := range []string{"event_id", "transaction_id", '
        '"task_id", "user_id", "company_id"} {'
    )
    ok, reason = mod.check_generic_key_field_order(good)
    assert ok, reason


def test_dispatch_requires_seen_mark_and_skip_log() -> None:
    mod = _load()
    bad = "func Handle() { return }\n"
    assert mod.check_dispatch_idempotency(bad) != []
    good = (
        'if s.Idempotency.Seen(key) {\n'
        '\ttracelog.EmitComponent("warn", "idempotency skip", "consumer", tid, m)\n'
        "\treturn DispatchSuccess, nil\n"
        "}\n"
        "s.Idempotency.Mark(key)\n"
    )
    assert mod.check_dispatch_idempotency(good) == []


def test_runner_must_wire_idempotent_dispatch() -> None:
    mod = _load()
    bad = "func Run() {}\n"
    assert mod.check_runner_wires_store(bad) != []
    good = (
        "dispatch := domain.IdempotentDispatchService{\n"
        "\tIdempotency: idempotency.NewMemoryStore(),\n"
        "}\n"
    )
    assert mod.check_runner_wires_store(good) == []


def test_scan_tmp_tree_finds_bypass_consumer() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as d:
        root = Path(d)
        (root / "taskBill" / "src").mkdir(parents=True)
        (root / "taskBill" / "src" / "worker.go").write_text(
            "package main\nfunc f() { _ = kafka.NewReader(kafka.ReaderConfig{}) }\n",
            encoding="utf-8",
        )
        hits = mod.scan_kafka_reader_violations(root)
        assert any("taskBill/src/worker.go" in h for h in hits)


def test_meta_rule_files_exist() -> None:
    assert CONSTRAINT.is_file(), CONSTRAINT
    assert CURSOR_RULE.is_file(), CURSOR_RULE
    assert ADR.is_file(), ADR
    body = CONSTRAINT.read_text(encoding="utf-8")
    assert "业务重复边界" in body
    assert "IdempotentDispatchService" in body
    assert "company_id" in body
    mdc = CURSOR_RULE.read_text(encoding="utf-8")
    assert "alwaysApply: true" in mdc
    adr = ADR.read_text(encoding="utf-8")
    assert "at-least-once" in adr


def test_key_regression_tests_named() -> None:
    mod = _load()
    missing = mod.check_key_regression_tests(ROOT)
    assert missing == [], missing


if __name__ == "__main__":
    tests = [v for k, v in sorted(globals().items()) if k.startswith("test_")]
    failed = 0
    for t in tests:
        try:
            t()
            print(f"PASS {t.__name__}")
        except Exception as exc:
            failed += 1
            print(f"FAIL {t.__name__}: {exc}")
    print(f"{'ok' if failed == 0 else 'FAILED'} ({len(tests) - failed}/{len(tests)} passed)")
    sys.exit(1 if failed else 0)
