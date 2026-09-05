#!/usr/bin/env python3
"""Self-test for check_vue_template_undefined_identifiers (no pytest required)."""

from __future__ import annotations

import importlib.util
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
CHECKER = ROOT / "db" / "scripts" / "ci" / "check_vue_template_undefined_identifiers.py"


def _load():
    spec = importlib.util.spec_from_file_location("check_vue_template_undefined_identifiers", CHECKER)
    assert spec and spec.loader
    mod = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = mod
    spec.loader.exec_module(mod)
    return mod


def _write(root: Path, name: str, content: str) -> Path:
    p = root / name
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(content, encoding="utf-8")
    return p


def test_undefined_bare_call_flagged_with_line() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as d:
        p = _write(Path(d), "Bad.vue", """<template>
  <div>
    <span>{{ missingFn(x) }}</span>
    <button @click="otherFn()">go</button>
  </div>
</template>

<script setup>
const ok = 1
</script>
""")
        findings = mod.analyze_sfc(p)
        ids = {ident for ident, _ in findings}
        assert ids == {"missingFn", "otherFn"}, ids
        lines = {line for ident, line in findings}
        # <template> 在第 1 行：插值在第 3 行，事件在第 4 行
        assert 3 in lines and 4 in lines, lines


def test_prop_function_not_flagged() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as d:
        p = _write(Path(d), "Prop.vue", """<template>
  <div>{{ formatDate(record.usage_time) }}{{ labelFn(row) }}</div>
</template>

<script setup>
const props = defineProps({
  records: { type: Array, default: () => [] },
  formatDate: { type: Function, required: true },
  labelFn: { type: Function, default: (m) => m?.name || '' },
})
defineEmits(['change-page'])
</script>
""")
        assert mod.analyze_sfc(p) == [], mod.analyze_sfc(p)


def test_emit_name_call_not_flagged() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as d:
        p = _write(Path(d), "Emit.vue", """<template>
  <div @click="emit('close')">x</div>
</template>

<script setup>
const emit = defineEmits(['close'])
</script>
""")
        assert mod.analyze_sfc(p) == []


def test_import_and_local_and_destructure_not_flagged() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as d:
        p = _write(Path(d), "Ok.vue", """<template>
  <div>{{ fmt(a) }}{{ fromImport(b) }}{{ fromComposable(c) }}</div>
</template>

<script setup>
import { apiFetch } from '../utils/apiUtils.js'
import { formatDate as fmt } from '../utils/format.js'
import { useThing } from '../composables/useThing.js'
const { fromComposable } = useThing()
function fromImport(v) { return v }
</script>
""")
        assert mod.analyze_sfc(p) == [], mod.analyze_sfc(p)


def test_member_call_base_and_globals_not_flagged() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as d:
        p = _write(Path(d), "Global.vue", """<template>
  <div>{{ Math.max(a, b) }}{{ JSON.stringify(x) }}{{ $t('k') }}{{ obj.method(v) }}</div>
</template>

<script setup>
const obj = { method(v) { return v } }
</script>
""")
        assert mod.analyze_sfc(p) == [], mod.analyze_sfc(p)


def test_no_script_setup_skipped() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as d:
        p = _write(Path(d), "Options.vue", """<template>
  <div>{{ mysteryFn(x) }}</div>
</template>

<script>
export default { name: 'Options' }
</script>
""")
        assert mod.analyze_sfc(p) == []


def test_known_issue_exemption_removed_flagged() -> None:
    """OPT-20260812-007 后 KNOWN_ISSUES 已清空：此前豁免的 formatDate 未定义模式必须重新被拦。"""
    mod = _load()
    with tempfile.TemporaryDirectory() as d:
        p = _write(Path(d), "SystemAdminReferralPerformanceDrawer.vue", """<template>
  <table>
    <tr><td>{{ formatDate(item.consumed_at) }}</td></tr>
  </table>
</template>

<script setup>
import { ref } from 'vue'
const data = ref(null)
</script>
""")
        findings = mod.analyze_sfc(p, rel="components/SystemAdminReferralPerformanceDrawer.vue")
        ids = {ident for ident, _ in findings}
        assert ids == {"formatDate"}, ids


def test_historical_cents_bug_pattern_flagged() -> None:
    """复现 BillingDashboard centsToYuanInternalStr 白屏根因：模板调用未定义函数必须被拦。"""
    mod = _load()
    with tempfile.TemporaryDirectory() as d:
        p = _write(Path(d), "BillingDashboard.vue", """<template>
  <div>
    <span v-for="tx in transactions" :key="tx.id">{{ centsToYuanInternalStr(tx.amount) }}</span>
  </div>
</template>

<script setup>
import { ref } from 'vue'
const transactions = ref([])
function formatDate() {}
</script>
""")
        findings = mod.analyze_sfc(p)
        ids = {ident for ident, _ in findings}
        assert ids == {"centsToYuanInternalStr"}, ids


def main() -> int:
    tests = [
        test_undefined_bare_call_flagged_with_line,
        test_prop_function_not_flagged,
        test_emit_name_call_not_flagged,
        test_import_and_local_and_destructure_not_flagged,
        test_member_call_base_and_globals_not_flagged,
        test_no_script_setup_skipped,
        test_known_issue_exemption_removed_flagged,
        test_historical_cents_bug_pattern_flagged,
    ]
    failed = 0
    for fn in tests:
        try:
            fn()
            print(f"PASS {fn.__name__}")
        except Exception as e:  # noqa: BLE001 — selftest reporter
            failed += 1
            print(f"FAIL {fn.__name__}: {e}", file=sys.stderr)
    if failed:
        print(f"{failed}/{len(tests)} failed", file=sys.stderr)
        return 1
    print(f"OK {len(tests)} tests")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
