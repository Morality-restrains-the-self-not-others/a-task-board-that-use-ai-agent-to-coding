#!/usr/bin/env python3
"""CI gate: ensure at least one non-empty GitLab provider YAML exists.

Prevents accidental deletion of all GitLab provider configs (e.g., when
"cleaning up unused files"), which causes OAuth start-from-gateway to fail
with `bad_state [debug: missing_allowed_host]`.

Related failure: .ai/09_failure_experience/02_runtime_errors/
    48_gitlab_oauth_bad_state_missing_allowed_host.md
"""

from __future__ import annotations

import sys
from pathlib import Path

try:
    import yaml
except ImportError:
    print("ERROR: PyYAML required", file=sys.stderr)
    raise SystemExit(2)


def monorepo_root() -> Path:
    here = Path(__file__).resolve()
    for parent in here.parents:
        if (parent / "db" / "registry.yaml").is_file():
            return parent
    raise FileNotFoundError("db/registry.yaml not found")


def main() -> int:
    root = monorepo_root()
    providers_dir = root / "conf" / "auth" / "git-oauth" / "providers"

    if not providers_dir.is_dir():
        print(f"FAIL: providers directory not found: {providers_dir}", file=sys.stderr)
        return 1

    gitlab_providers: list[tuple[str, dict]] = []
    empty_or_invalid: list[str] = []

    for path in sorted(providers_dir.glob("*.yaml")):
        if path.name.endswith(".ai.md"):
            continue
        try:
            raw = path.read_text(encoding="utf-8").strip()
        except OSError as e:
            print(f"FAIL: cannot read {path.relative_to(root)}: {e}", file=sys.stderr)
            return 1

        if not raw:
            empty_or_invalid.append(str(path.relative_to(root)))
            continue

        try:
            data = yaml.safe_load(raw)
        except yaml.YAMLError as e:
            print(f"FAIL: invalid YAML in {path.relative_to(root)}: {e}", file=sys.stderr)
            return 1

        if not isinstance(data, dict):
            empty_or_invalid.append(str(path.relative_to(root)))
            continue

        if not data:
            empty_or_invalid.append(str(path.relative_to(root)))
            continue

        provider = data.get("provider", "").strip().lower()
        if provider == "gitlab":
            gitlab_providers.append((str(path.relative_to(root)), data))

    # Check 1: at least one non-empty GitLab provider YAML
    if not gitlab_providers:
        print(
            "FAIL: no GitLab provider YAML found in conf/auth/git-oauth/providers/.\n"
            "At least one *.yaml with `provider: gitlab` is required for OAuth to work.\n"
            "Do NOT delete all *gitlab* YAML files — they are NOT unused.",
            file=sys.stderr,
        )
        return 1

    # Check 2: each GitLab provider has required fields
    required_fields = ["service_provider", "target"]
    for rel_path, data in gitlab_providers:
        missing = [f for f in required_fields if not data.get(f)]
        if missing:
            print(
                f"FAIL: {rel_path} has provider=gitlab but missing fields: {missing}",
                file=sys.stderr,
            )
            return 1
        target = data.get("target", {})
        if isinstance(target, dict):
            if not target.get("client_id"):
                print(
                    f"FAIL: {rel_path} has provider=gitlab but target.client_id is empty",
                    file=sys.stderr,
                )
                return 1

    print(
        f"OK: {len(gitlab_providers)} GitLab provider(s) found, "
        f"all have required fields"
    )
    if empty_or_invalid:
        print(
            f"  (also found {len(empty_or_invalid)} empty/invalid YAML files, "
            f"but they are not GitLab providers)"
        )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
