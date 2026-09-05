"""Validate AiMonitor promtail config contains runAll scrape pipeline."""

from __future__ import annotations

from pathlib import Path

import yaml


def test_promtail_has_runall_scrape_and_trace_pipeline() -> None:
    path = Path(__file__).resolve().parents[1] / "promtail" / "promtail.yaml"
    data = yaml.safe_load(path.read_text(encoding="utf-8"))
    assert isinstance(data, dict)

    scrape = data.get("scrape_configs") or []
    assert scrape, "scrape_configs required"

    runall = next((j for j in scrape if j.get("job_name") == "runall-services"), None)
    assert runall is not None, "runall-services job required"

    labels = runall["static_configs"][0]["labels"]
    assert "/var/log/runall/*.log" in labels.get("__path__", "")

    stages = runall.get("pipeline_stages") or []
    stage_types = {next(iter(s.keys())) for s in stages if isinstance(s, dict) and s}
    assert "regex" in stage_types
    assert "json" in stage_types
    assert "template" in stage_types

    label_stage = next((s for s in stages if "labels" in s), None)
    assert label_stage is not None
    assert "trace_id" in label_stage["labels"]
    assert "level" in label_stage["labels"]

    plain = next(
        (
            s
            for s in stages
            if "regex" in s and "plain_level" in str(s.get("regex", {}).get("expression", ""))
        ),
        None,
    )
    assert plain is not None, "plaintext level fallback required"

    output_stage = next((s for s in stages if "output" in s), None)
    assert output_stage is not None
    assert output_stage["output"].get("source") == "payload"

    meta_stage = next((s for s in stages if "structured_metadata" in s), None)
    assert meta_stage is not None
    assert "msg" in meta_stage["structured_metadata"]

    relabel = runall.get("relabel_configs") or []
    assert any(r.get("target_label") == "service" for r in relabel)

    clients = data.get("clients") or []
    assert any("loki" in str(c.get("url", "")) for c in clients)
