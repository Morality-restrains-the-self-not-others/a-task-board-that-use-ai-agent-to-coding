#!/usr/bin/env python3
"""禁止在源码中硬编码密钥（元规则 57 / ADR-0046）。

密钥须落在 conf-local/、环境变量或密钥管理系统。
本门禁扫描第一方源码中的高置信泄露模式，以及生产代码里把 secret 标识符赋成字面量。

用法:
  python3 db/scripts/ci/check_no_hardcoded_secrets.py
  python3 db/scripts/ci/test_check_no_hardcoded_secrets.py
"""
from __future__ import annotations

import argparse
import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]

META_FILES = (
    ".ai/01_project_constraints/62_no_hardcoded_secrets.md",
    ".cursor/rules/no-hardcoded-secrets.mdc",
    "docs/adr/0046-no-hardcoded-secrets.md",
)

IGNORE_REL = {
    "db/scripts/ci/check_no_hardcoded_secrets.py",
    "db/scripts/ci/test_check_no_hardcoded_secrets.py",
}

SKIP_DIR_PARTS = frozenset(
    {
        ".git",
        "node_modules",
        "vendor",
        "third_party",
        "gitlab-ce",
        "gitlab_home",
        "sdk",
        "__pycache__",
        ".venv",
        "venv",
        "dist",
        "build",
        ".tox",
        ".pytest_cache",
        "e2e-tests",
        "playwright",
        "coverage",
        "tmp",
        "logs",
        ".daydaymoney-deploy-seed",
    }
)

SOURCE_SUFFIXES = frozenset(
    {
        ".go",
        ".py",
        ".js",
        ".jsx",
        ".mjs",
        ".cjs",
        ".ts",
        ".tsx",
        ".vue",
        ".sh",
        ".rb",
        ".java",
    }
)

MAX_FILE_BYTES = 1_000_000

WAIVER_RE = re.compile(r"Secret-Hardcode-OK:\s*\S+")

AWS_EXAMPLE_KEYS = frozenset({"AKIAIOSFODNN7EXAMPLE"})

HIGH_CONFIDENCE: tuple[tuple[str, re.Pattern[str]], ...] = (
    (
        "pem-private-key",
        re.compile(r"-----BEGIN (?:RSA |EC |OPENSSH |DSA |ENCRYPTED )?PRIVATE KEY-----"),
    ),
    ("aws-access-key", re.compile(r"\b(AKIA[0-9A-Z]{16})\b")),
    ("github-pat", re.compile(r"\b(?:ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9]{36,}\b")),
    ("github-fine-grained", re.compile(r"\bgithub_pat_[A-Za-z0-9_]{20,}\b")),
    ("slack-token", re.compile(r"\bxox[baprs]-[A-Za-z0-9-]{10,}\b")),
    ("stripe-live", re.compile(r"\bsk_live_[0-9A-Za-z]{16,}\b")),
    ("google-api-key", re.compile(r"\bAIza[0-9A-Za-z_-]{35}\b")),
)

# (?<!-) avoids Vue/HTML kebab props like show-email-password="..."
ASSIGNMENT_RE = re.compile(
    r"(?i)(?<!-)(?<![A-Za-z0-9_])("
    r"api[_-]?key|access[_-]?key(?:[_-](?:id|secret))?|"
    r"client[_-]?secret|app[_-]?secret|private[_-]?key|"
    r"secret[_-]?key|auth[_-]?token|access[_-]?token|"
    r"refresh[_-]?token|password|"
    r"internal[_-]?secret|webhook[_-]?secret|"
    r"jwt[_-]?secret|encryption[_-]?key|master[_-]?key|"
    r"mysql[_-]?password|redis[_-]?password|smtp[_-]?password|"
    r"db[_-]?password"
    r")\s*[:=]\s*['\"]([^'\"]{8,})['\"]"
)

PLACEHOLDER_RE = re.compile(
    r"(?i)^("
    r"test|dummy|fake|example|changeme|xxx+|your[-_].*|insert[-_].*|"
    r"todo|placeholder|secret|password|passwd|none|null|n/?a|"
    r"redact.*|replace.*|<.*>|\$\{.*\}|%\w|%\(.*\)|"
    r"test[-_].*|dummy[-_].*|fake[-_].*|example[-_].*|"
    r"sk-test-.*|sk-config-key"
    r")$"
)

DOTENV_ALLOW_NAMES = frozenset(
    {".env.example", ".env.sample", ".env.template"}
)


def is_placeholder(value: str) -> bool:
    stripped = value.strip()
    if not stripped:
        return True
    if PLACEHOLDER_RE.match(stripped):
        return True
    if stripped.startswith("${") or stripped.startswith("$("):
        return True
    if re.fullmatch(r"\$[A-Za-z_][A-Za-z0-9_]*", stripped):
        return True
    if re.fullmatch(r"__[A-Za-z0-9_]+__", stripped):
        return True
    lower = stripped.lower()
    if lower.startswith(
        ("test-", "dummy-", "fake-", "example-", "your-", "changeme", "dev-local")
    ):
        return True
    if re.fullmatch(r"[xX.*]+", stripped):
        return True
    return False


def posix_rel(path: Path, root: Path) -> str:
    return path.relative_to(root).as_posix()


def should_skip_dir(path: Path, root: Path) -> bool:
    try:
        rel = path.relative_to(root)
    except ValueError:
        return True
    return any(part in SKIP_DIR_PARTS for part in rel.parts)


def is_allowed_secret_store(rel: str) -> bool:
    if rel.startswith("conf/"):
        return True
    name = Path(rel).name
    if name.endswith(".local.yaml") or name.endswith(".local.yml"):
        return True
    if name in DOTENV_ALLOW_NAMES:
        return True
    if name == ".env" or name.startswith(".env."):
        return True
    return False


def is_example_template(name: str) -> bool:
    """Config templates like trae_config.yaml.example.

    Placeholders (assignment values) stay exempt, but HIGH_CONFIDENCE leak
    patterns must still be scanned — example files are the most likely spot
    for someone to paste a real PEM / sk_live_ / AKIA token.
    """
    return name.endswith(".example")


def is_test_path(rel: str) -> bool:
    name = Path(rel).name
    if name.endswith("_test.go") or name.startswith(("test_", "test-")):
        return True
    if ".test." in name or ".spec." in name or name.endswith(".test.sh"):
        return True
    parts = rel.split("/")
    return any(p in {"testdata", "fixtures", "tests", "test"} for p in parts)


def line_has_waiver(line: str, prev: str) -> bool:
    return bool(WAIVER_RE.search(line) or WAIVER_RE.search(prev))


def iter_scan_files(root: Path):
    for path in root.rglob("*"):
        if not path.is_file():
            continue
        if should_skip_dir(path, root):
            continue
        try:
            rel = posix_rel(path, root)
        except ValueError:
            continue
        if rel in IGNORE_REL:
            continue
        if is_allowed_secret_store(rel):
            continue
        if "/public/releases/" in f"/{rel}/":
            continue
        suffix = path.suffix.lower()
        name = path.name
        example = is_example_template(name)
        high_conf_only = example or suffix not in SOURCE_SUFFIXES
        if not example and high_conf_only and suffix not in {".yml", ".yaml", ".json", ".toml", ".xml"}:
            if not name.startswith("Dockerfile"):
                continue
        try:
            if path.stat().st_size > MAX_FILE_BYTES:
                continue
        except OSError:
            continue
        yield path, rel, high_conf_only


def scan_text(rel: str, text: str, *, assignment: bool) -> list[str]:
    hits: list[str] = []
    prev = ""
    for lineno, line in enumerate(text.splitlines(), 1):
        if line_has_waiver(line, prev):
            prev = line
            continue
        for kind, pattern in HIGH_CONFIDENCE:
            for match in pattern.finditer(line):
                token = match.group(1) if match.lastindex else match.group(0)
                if kind == "aws-access-key" and token in AWS_EXAMPLE_KEYS:
                    continue
                hits.append(f"{rel}:{lineno}: {kind}")
        if assignment and not is_test_path(rel):
            for match in ASSIGNMENT_RE.finditer(line):
                literal = match.group(2)
                if is_placeholder(literal):
                    continue
                if re.search(r"[\s\u4e00-\u9fff]", literal):
                    continue
                if literal.startswith("data-testid"):
                    continue
                hits.append(f"{rel}:{lineno}: assignment {match.group(1)}=<literal>")
        prev = line
    return hits


def tracked_dotenv_violations(root: Path) -> list[str]:
    git_dir = root / ".git"
    if not git_dir.exists():
        return []
    try:
        raw = subprocess.check_output(
            ["git", "-C", str(root), "ls-files", "-z"],
            stderr=subprocess.DEVNULL,
        )
    except (OSError, subprocess.CalledProcessError):
        return []
    hits: list[str] = []
    for rel in raw.decode("utf-8", "replace").split("\0"):
        if not rel:
            continue
        name = Path(rel).name
        if name in DOTENV_ALLOW_NAMES or name.endswith(".example"):
            continue
        if name == ".env" or name.startswith(".env."):
            hits.append(
                f"{rel}: committed dotenv; keep secrets in gitignored "
                ".env / conf/**/config.local.yaml"
            )
    return hits


def collect_violations(root: Path, *, check_meta: bool = True) -> list[str]:
    hits: list[str] = []
    if check_meta:
        for rel in META_FILES:
            if not (root / rel).is_file():
                hits.append(f"{rel}: missing meta file")
    hits.extend(tracked_dotenv_violations(root))
    for path, rel, high_conf_only in iter_scan_files(root):
        try:
            text = path.read_text(encoding="utf-8", errors="replace")
        except OSError:
            continue
        hits.extend(scan_text(rel, text, assignment=not high_conf_only))
    return hits


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=ROOT)
    args = parser.parse_args(argv)
    root = args.root.resolve()
    hits = collect_violations(root, check_meta=True)
    if hits:
        print("VIOLATION (rule 62_no_hardcoded_secrets.md / ADR-0046):")
        for hit in hits[:80]:
            print(f"  - {hit}")
        if len(hits) > 80:
            print(f"  ... +{len(hits) - 80} more")
        return 1
    print("ok: no hardcoded secrets in first-party source")
    return 0


if __name__ == "__main__":
    sys.exit(main())
