#!/usr/bin/env python3
"""Unit tests for run_commit_random_unit_tests.py quota and sampling helpers."""

from __future__ import annotations

import random
import tempfile
import unittest
from pathlib import Path

from run_commit_random_unit_tests import (
    discover_unit_tests,
    go_test_func_names,
    go_test_is_helper_file,
    go_test_run_pattern,
    required_fix_count,
    sample_tests,
)


class RequiredFixCountTests(unittest.TestCase):
    def test_zero(self) -> None:
        self.assertEqual(required_fix_count(0), 0)

    def test_one_through_ten(self) -> None:
        for n in range(1, 11):
            self.assertEqual(required_fix_count(n), 1, msg=f"n={n}")

    def test_eleven(self) -> None:
        self.assertEqual(required_fix_count(11), 2)

    def test_twenty_five(self) -> None:
        self.assertEqual(required_fix_count(25), 3)

    def test_custom_ratio(self) -> None:
        self.assertEqual(required_fix_count(10, ratio=0.2), 2)


class SampleTests(unittest.TestCase):
    def test_empty_pool(self) -> None:
        self.assertEqual(
            sample_tests([], ratio=0.1, min_sample=1, max_sample=8, rng=random.Random(0)),
            [],
        )

    def test_respects_min_and_max(self) -> None:
        pool = [f"t{i}.unit.test.js" for i in range(100)]
        rng = random.Random(42)
        got = sample_tests(pool, ratio=0.1, min_sample=1, max_sample=8, rng=rng)
        self.assertEqual(len(got), 8)
        self.assertEqual(len(set(got)), 8)
        self.assertTrue(set(got).issubset(set(pool)))

    def test_small_pool(self) -> None:
        pool = ["a.unit.test.js", "b.unit.test.js"]
        got = sample_tests(pool, ratio=0.1, min_sample=1, max_sample=8, rng=random.Random(1))
        self.assertEqual(len(got), 1)


class DiscoverTests(unittest.TestCase):
    def test_include_and_exclude(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            keep = root / "pkg" / "foo.unit.test.js"
            keep.parent.mkdir(parents=True)
            keep.write_text("// ok\n", encoding="utf-8")
            drop = root / "node_modules" / "x.unit.test.js"
            drop.parent.mkdir(parents=True)
            drop.write_text("// skip\n", encoding="utf-8")
            e2e = root / "e2e" / "z.unit.test.js"
            e2e.parent.mkdir(parents=True)
            e2e.write_text("// skip\n", encoding="utf-8")
            third = root / "gitService" / "gitlab-ce" / "a_test.go"
            third.parent.mkdir(parents=True)
            third.write_text("package a\n", encoding="utf-8")
            cfg = {
                "include_globs": ["**/*.unit.test.js", "**/*_test.go"],
                "exclude_globs": [
                    "**/node_modules/**",
                    "**/e2e/**",
                    "**/gitService/gitlab-ce/**",
                ],
            }
            found = discover_unit_tests(root, cfg)
            self.assertEqual(found, ["pkg/foo.unit.test.js"])


class GoTestFileScopeTests(unittest.TestCase):
    def test_extracts_top_level_test_funcs(self) -> None:
        src = (
            "package main\n"
            "func helper() {}\n"
            "func TestAlpha(t *testing.T) {}\n"
            "func TestBeta(t *testing.T) {\n"
            "\t// nested TestGamma mention is not a declaration\n"
            "}\n"
            "func TestAlpha(t *testing.T) {}\n"
            "func BenchmarkNope(b *testing.B) {}\n"
        )
        self.assertEqual(go_test_func_names(src), ["TestAlpha", "TestBeta"])

    def test_run_pattern_anchors_and_escapes(self) -> None:
        self.assertEqual(go_test_run_pattern([]), "")
        self.assertEqual(
            go_test_run_pattern(["TestA", "TestB"]),
            "^(TestA|TestB)$",
        )

    def test_helper_file_without_test_funcs(self) -> None:
        helper = "package main\nfunc setupBudgetTestDB(t *testing.T) {}\n"
        self.assertTrue(go_test_is_helper_file(helper))
        self.assertFalse(
            go_test_is_helper_file(helper + "func TestFoo(t *testing.T) {}\n"),
        )

    def test_discover_excludes_go_helpers_test(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "src").mkdir()
            (root / "src" / "saas_http_test_helpers_test.go").write_text(
                "package main\nfunc setup(t *testing.T) {}\n", encoding="utf-8"
            )
            (root / "src" / "real_test.go").write_text(
                "package main\nfunc TestFoo(t *testing.T) {}\n", encoding="utf-8"
            )
            cfg = {
                "include_globs": ["**/*_test.go"],
                "exclude_globs": ["**/*_helpers_test.go"],
            }
            found = discover_unit_tests(root, cfg)
            self.assertEqual(found, ["src/real_test.go"])


if __name__ == "__main__":
    unittest.main()
