#!/usr/bin/env python3
"""Unit tests for check_deployment_doc_paths.py (OPT-20260824-018)."""

from __future__ import annotations

import tempfile
import unittest
from pathlib import Path

from check_deployment_doc_paths import (
    check_doc,
    parse_candidates,
    resolve,
)


class ParseCandidatesTests(unittest.TestCase):
    def test_extracts_bash_paths_and_cd_context(self) -> None:
        doc = """## x
```bash
cd taskGateway
bash run.sh start
bash runAll/scripts/install-hooks-all.sh
```
```bash
bash runAll/scripts/install-hooks-all.sh
```
"""
        cands = parse_candidates(doc)
        pairs = [(t, cwd) for _l, t, cwd in cands]
        # 同一代码块内 cd 后续命令按 shell 语义相对 cwd 解析
        self.assertIn(("run.sh", "taskGateway"), pairs)
        self.assertIn(("runAll/scripts/install-hooks-all.sh", "taskGateway"), pairs)
        # 新代码块重置 cwd → 根相对
        self.assertIn(("runAll/scripts/install-hooks-all.sh", None), pairs)

    def test_cd_and_bash_same_line(self) -> None:
        doc = """```bash
cd taskSSE && bash run.sh start
```"""
        cands = parse_candidates(doc)
        self.assertEqual(cands, [(2, "run.sh", "taskSSE")])

    def test_fence_resets_cwd(self) -> None:
        doc = """```bash
cd taskEvents
bash run.sh build
```
```bash
bash runAll/scripts/conf-sync-all.sh
```
"""
        cands = parse_candidates(doc)
        by_token = {t: cwd for _l, t, cwd in cands}
        # 第二个代码块 cwd 应重置为 None
        self.assertEqual(by_token.get("runAll/scripts/conf-sync-all.sh"), None)

    def test_placeholder_cd_clears_cwd(self) -> None:
        doc = """```bash
cd <service-dir>
bash run.sh start
```"""
        cands = parse_candidates(doc)
        by_token = {t: cwd for _l, t, cwd in cands}
        self.assertEqual(by_token.get("run.sh"), None)

    def test_skips_flags_and_placeholders(self) -> None:
        doc = """```bash
bash -c 'lsof -ti:8003'
bash run.sh build <event>/<intent>
```"""
        cands = parse_candidates(doc)
        # `-c` 跳过；`run.sh` 无占位符被捕获但 cwd=None 不可解析（resolve 返回 None，不校验）
        self.assertEqual(cands, [(3, "run.sh", None)])

    def test_dotslash_token(self) -> None:
        doc = """```bash
cd go_relayToTrae
./build.sh
./bin/go_relayToTrae
```"""
        cands = parse_candidates(doc)
        toks = [t for _l, t, _c in cands]
        self.assertIn("./build.sh", toks)
        self.assertIn("./bin/go_relayToTrae", toks)

    def test_ignores_prose_outside_fence(self) -> None:
        doc = "bash run.sh start (prose, not in fence)\n"
        self.assertEqual(parse_candidates(doc), [])


class ResolveTests(unittest.TestCase):
    def setUp(self) -> None:
        self._tmp = tempfile.TemporaryDirectory()
        self.root = Path(self._tmp.name)

    def tearDown(self) -> None:
        self._tmp.cleanup()

    def test_root_relative(self) -> None:
        p = resolve("runAll/scripts/install-hooks-all.sh", None, self.root)
        self.assertEqual(p, self.root / "runAll/scripts/install-hooks-all.sh")

    def test_cwd_relative_bare(self) -> None:
        p = resolve("run.sh", "taskGateway", self.root)
        self.assertEqual(p, self.root / "taskGateway" / "run.sh")

    def test_cwd_relative_dotslash(self) -> None:
        p = resolve("./build.sh", "go_relayToTrae", self.root)
        self.assertEqual(p, self.root / "go_relayToTrae" / "build.sh")

    def test_bare_without_cwd_is_none(self) -> None:
        self.assertIsNone(resolve("run.sh", None, self.root))


class CheckDocTests(unittest.TestCase):
    def setUp(self) -> None:
        self._tmp = tempfile.TemporaryDirectory()
        self.root = Path(self._tmp.name)

    def tearDown(self) -> None:
        self._tmp.cleanup()

    def _write(self, rel: str, content: str) -> None:
        p = self.root / rel
        p.parent.mkdir(parents=True, exist_ok=True)
        p.write_text(content, encoding="utf-8")

    def test_all_paths_exist(self) -> None:
        self._write("docs/runbooks/manual-full-deployment.md", "```bash\ncd taskGateway\nbash run.sh start\n```\n")
        self._write("taskGateway/run.sh", "#!/bin/bash\n")
        self.assertEqual(check_doc(self.root, "docs/runbooks/manual-full-deployment.md"), [])

    def test_missing_path_reported(self) -> None:
        self._write("docs/runbooks/manual-full-deployment.md", "```bash\nbash runAll/scripts/install-hooks-all.sh\n```\n")
        hits = check_doc(self.root, "docs/runbooks/manual-full-deployment.md")
        self.assertEqual(len(hits), 1)
        self.assertIn("runAll/scripts/install-hooks-all.sh", hits[0])

    def test_missing_doc_file_reported(self) -> None:
        hits = check_doc(self.root, "docs/runbooks/manual-full-deployment.md")
        self.assertEqual(len(hits), 1)
        self.assertIn("missing", hits[0])


if __name__ == "__main__":
    unittest.main()
