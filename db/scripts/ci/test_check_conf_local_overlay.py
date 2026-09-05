#!/usr/bin/env python3
"""Self-test for check_conf_local_overlay (no pytest required)."""

from __future__ import annotations

import importlib.util
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
CHECKER = ROOT / "db" / "scripts" / "ci" / "check_conf_local_overlay.py"


def _load():
    spec = importlib.util.spec_from_file_location("check_conf_local_overlay", CHECKER)
    assert spec and spec.loader
    mod = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = mod
    spec.loader.exec_module(mod)
    return mod


RAW_GO = """package main
import "os"
func load() {
    b, _ := os.ReadFile(filepath.Join(root, "conf", "auth", "task-auth", "email.yaml"))
    _ = b
}
"""

OVERLAY_GO = """package main
import "os"
func load() {
    var cfg Cfg
    _ = confload.UnmarshalYAMLMerged(root, "auth/task-auth/email.yaml", &cfg)
}
"""

RAW_PY = """
import yaml
from pathlib import Path
def load(root):
    with open(root / "conf" / "runAll.yaml") as f:
        return yaml.safe_load(f)
"""

OVERLAY_PY = """
from conf_local import overlay_conf_file
def load(root):
    return overlay_conf_file(root, "runAll.yaml")
"""

CONFIG_LOCAL_GO = """package main
func load() {
    b, _ := os.ReadFile(filepath.Join(appDir, "config.local.yaml"))
    _ = b
}
"""


def test_raw_go_conf_yaml_is_violation() -> None:
    mod = _load()
    kind, _ = mod.classify("taskEvents/config/conf_yaml.go", RAW_GO)
    assert kind == "violation"


def test_ssot_go_overlay_is_ok() -> None:
    mod = _load()
    kind, _ = mod.classify("taskBill/src/paypal_pay.go", OVERLAY_GO)
    assert kind == "ok"


def test_raw_python_conf_yaml_is_violation() -> None:
    mod = _load()
    kind, _ = mod.classify("runAll/scripts/ensure_services_healthy.py", RAW_PY)
    assert kind == "violation"


def test_python_overlay_helper_is_ok() -> None:
    mod = _load()
    kind, _ = mod.classify("runAll/scripts/conf_loader.py", OVERLAY_PY)
    assert kind == "ok"


def test_test_file_is_ok() -> None:
    mod = _load()
    kind, _ = mod.classify("taskBill/src/paypal_pay_test.go", RAW_GO)
    assert kind == "ok"


def test_confload_impl_is_ok() -> None:
    mod = _load()
    kind, _ = mod.classify("shareLib/confload/load.go", RAW_GO)
    assert kind == "ok"


def test_waiver_is_ok() -> None:
    mod = _load()
    src = RAW_GO + "\n// Conf-Local-Overlay-OK: custom merge pending OPT-20260901-012\n"
    kind, _ = mod.classify("taskBill/src/wechat_pay.go", src)
    assert kind == "ok"


def test_custom_wechat_overlay_token_is_ok() -> None:
    mod = _load()
    src = RAW_GO + "\nfunc overlayWechatLocalConfig() {}\n"
    kind, _ = mod.classify("taskBill/src/wechat_pay.go", src)
    assert kind == "ok"


def test_config_local_yaml_load_is_violation() -> None:
    mod = _load()
    kind, reason = mod.classify("taskSSE/src/config.mjs", CONFIG_LOCAL_GO)
    assert kind == "violation"
    assert "config.local.yaml" in reason


def test_quoted_conf_local_path_is_ok() -> None:
    mod = _load()
    src = """
import yaml
from pathlib import Path
def load(root):
    cfg = yaml.safe_load((root / "conf" / "auth" / "task-auth" / "config.yaml").read_text())
    local = yaml.safe_load((root / "conf-local" / "auth" / "task-auth" / "config.yaml").read_text())
    return cfg, local
"""
    kind, _ = mod.classify("dataMigrate/taskAuth/bootstrap_admin_email.py", src)
    assert kind == "ok"


def test_proc_readfile_with_conf_flag_is_ok() -> None:
    mod = _load()
    src = """package main
func scan() {
    cmdline, _ := os.ReadFile(filepath.Join("/proc", "1", "cmdline"))
    _ = cmdline
}
var configPath = flag.String("config", "conf/runAll.yaml", "Path to YAML")
"""
    kind, _ = mod.classify("runAll/src/main.go", src)
    assert kind == "ok"


def test_ci_inspector_is_ok() -> None:
    mod = _load()
    kind, _ = mod.classify("db/scripts/ci/check_gitlab_sso_only_no_self_signup.py", RAW_PY)
    assert kind == "ok"


def test_pem_read_without_yaml_is_ok() -> None:
    mod = _load()
    src = """package main
func load() {
    b, _ := os.ReadFile(filepath.Join(root, "conf-local", "auth", "task-auth", "jwt.pem"))
    _ = b
}
"""
    kind, _ = mod.classify("taskAuth/infrastructure/jwt.go", src)
    assert kind == "ok"


def test_missing_meta_is_violation() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as tmp:
        hits = mod.collect_violations(Path(tmp))
    assert any("missing meta" in h for h in hits), hits


def test_live_repo_passes() -> None:
    hits = _load().collect_violations(ROOT)
    assert hits == [], hits


def main() -> int:
    tests = [
        test_raw_go_conf_yaml_is_violation,
        test_ssot_go_overlay_is_ok,
        test_raw_python_conf_yaml_is_violation,
        test_python_overlay_helper_is_ok,
        test_test_file_is_ok,
        test_confload_impl_is_ok,
        test_waiver_is_ok,
        test_custom_wechat_overlay_token_is_ok,
        test_config_local_yaml_load_is_violation,
        test_quoted_conf_local_path_is_ok,
        test_proc_readfile_with_conf_flag_is_ok,
        test_ci_inspector_is_ok,
        test_pem_read_without_yaml_is_ok,
        test_missing_meta_is_violation,
        test_live_repo_passes,
    ]
    failed = 0
    for t in tests:
        try:
            t()
            print(f"ok {t.__name__}")
        except Exception as exc:
            failed += 1
            print(f"FAIL {t.__name__}: {exc}")
    if failed:
        print(f"failed: {failed}/{len(tests)}")
        return 1
    print(f"test_check_conf_local_overlay ok: {len(tests)} tests")
    return 0


if __name__ == "__main__":
    sys.exit(main())
