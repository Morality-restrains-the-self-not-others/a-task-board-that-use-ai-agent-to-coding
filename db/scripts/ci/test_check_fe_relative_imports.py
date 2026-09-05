#!/usr/bin/env python3
"""Self-test for check_fe_relative_imports (no pytest required)."""

from __future__ import annotations

import importlib.util
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
CHECKER = ROOT / "db" / "scripts" / "ci" / "check_fe_relative_imports.py"


def _load():
    spec = importlib.util.spec_from_file_location("check_fe_relative_imports", CHECKER)
    assert spec and spec.loader
    mod = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = mod
    spec.loader.exec_module(mod)
    return mod


def test_resolve_extension_variants() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as d:
        base = Path(d)
        (base / "a.js").write_text("", encoding="utf-8")
        (base / "b.vue").write_text("", encoding="utf-8")
        (base / "sub").mkdir()
        (base / "sub" / "index.js").write_text("", encoding="utf-8")
        assert mod._resolve(base, "./a") is True
        assert mod._resolve(base, "./a.js") is True
        assert mod._resolve(base, "./b") is True
        assert mod._resolve(base, "./sub") is True
        assert mod._resolve(base, "./nope") is False


def test_scan_finds_unresolved_import() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as d:
        src = Path(d) / "src"
        (src / "views").mkdir(parents=True)
        (src / "views" / "Bad.vue").write_text(
            "<script setup>\nimport { missingThing } from '../utils/missingModule.js'\n</script>",
            encoding="utf-8",
        )
        findings = mod.scan_dir(src)
        assert len(findings) == 1, findings
        assert "missingModule" in findings[0][2]


def test_scan_skips_package_and_alias_imports() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as d:
        src = Path(d) / "src"
        (src / "views").mkdir(parents=True)
        (src / "utils").mkdir()
        (src / "views" / "Ok.vue").write_text(
            "import { ref } from 'vue'\n"
            "import { apiFetch } from '@/utils/apiUtils.js'\n"
            "import util from '../utils/existing.js'\n",
            encoding="utf-8",
        )
        (src / "utils" / "existing.js").write_text("", encoding="utf-8")
        assert mod.scan_dir(src) == []


def test_scan_ignores_query_and_dynamic_import() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as d:
        src = Path(d) / "src"
        (src / "views").mkdir(parents=True)
        (src / "components").mkdir()
        (src / "views" / "Dyn.vue").write_text(
            "const c = () => import('../components/Widget.vue?raw')\n"
            "const p = () => import('./Lazy.vue')\n",
            encoding="utf-8",
        )
        (src / "components" / "Widget.vue").write_text("<template><div/></template>", encoding="utf-8")
        (src / "views" / "Lazy.vue").write_text("<template><div/></template>", encoding="utf-8")
        assert mod.scan_dir(src) == []


def main() -> int:
    tests = [
        test_resolve_extension_variants,
        test_scan_finds_unresolved_import,
        test_scan_skips_package_and_alias_imports,
        test_scan_ignores_query_and_dynamic_import,
    ]
    failed = 0
    for fn in tests:
        try:
            fn()
            print(f"PASS {fn.__name__}")
        except Exception as e:  # noqa: BLE001
            failed += 1
            print(f"FAIL {fn.__name__}: {e}", file=sys.stderr)
    if failed:
        print(f"{failed}/{len(tests)} failed", file=sys.stderr)
        return 1
    print(f"OK {len(tests)} tests")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
