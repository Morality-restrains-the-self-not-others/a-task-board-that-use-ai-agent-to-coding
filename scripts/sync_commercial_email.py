#!/usr/bin/env python3
r"""商业联系邮箱单源同步（OPT-20260824-087）。

根仓 `README.md` 的「商务联系」行与根仓 `COMMERCIAL.md` 的「邮件联系」行是
**唯一真源（SSOT）**；所有子仓 README/COMMERCIAL 副本必须与之保持一致。
本脚本把 SSOT 邮箱同步到各子仓副本，并提供 `--check` 只读校验模式（可作门禁）。

用法:
  python scripts/sync-commercial-email.py            # 同步（就地改写不一致的副本）
  python scripts/sync-commercial-email.py --dry-run  # 只打印将改动的行，不写盘
  python scripts/sync-commercial-email.py --check    # 只读校验，有漂移则 exit 1
  python scripts/sync-commercial-email.py --repo taskAuth  # 仅同步指定子仓

SSOT 提取规则:
  - README  行匹配 `^商务联系[:：]\s*<email>`
  - COMMERCIAL 行匹配 `^1. 邮件联系 <email>`（支持纯文本与 `[email](mailto:email)` 两种格式）
  - 两者必须一致；不一致时报错，提示先修正根仓。
"""
from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

# 邮箱正则：覆盖纯文本与 markdown 链接文本/mailto 两处。
EMAIL_RE = re.compile(r"[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}")

README_BIZ_RE = re.compile(r"^(商务联系[:：]\s*).*$")
COMMERCIAL_BIZ_RE = re.compile(r"^(1\.\s*邮件联系\s+).*$")

# 各子仓相对路径。README/COMMERCIAL 二者其一存在即参与校验。
DOC_FILES = ("README.md", "COMMERCIAL.md")


def ssot_emails(root: Path) -> dict[str, str]:
    """从根仓 README + COMMERCIAL 提取 SSOT 邮箱。返回 {README/COMMERCIAL: email}。"""
    out: dict[str, str] = {}
    readme = root / "README.md"
    if readme.exists():
        for line in readme.read_text(encoding="utf-8").splitlines():
            m = re.match(r"^商务联系[:：]\s*([A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,})", line)
            if m:
                out["README"] = m.group(1)
                break
    commercial = root / "COMMERCIAL.md"
    if commercial.exists():
        for line in commercial.read_text(encoding="utf-8").splitlines():
            m = re.match(
                r"^1\.\s*邮件联系\s+\[?([A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,})",
                line,
            )
            if m:
                out["COMMERCIAL"] = m.group(1)
                break
    if not out:
        raise ValueError(f"SSOT 邮箱缺失：根仓 README/COMMERCIAL 未找到「商务联系/邮件联系」行（{root}）")
    return out


def find_subrepos(root: Path) -> list[Path]:
    """列出含 README/COMMERCIAL 文档的子仓目录（排除根仓自身）。"""
    out = []
    for child in sorted(root.iterdir()):
        if not child.is_dir() or child.name.startswith("."):
            continue
        if any((child / f).exists() for f in DOC_FILES):
            out.append(child)
    return out


def sync_file(path: Path, email: str, dry_run: bool) -> tuple[bool, str]:
    """同步单个文档文件中的邮箱。返回 (changed, reason)。"""
    text = path.read_text(encoding="utf-8")
    lines = text.splitlines()
    changed_any = False
    for i, line in enumerate(lines):
        if README_BIZ_RE.match(line) or COMMERCIAL_BIZ_RE.match(line):
            new = EMAIL_RE.sub(email, line)
            if new != line:
                lines[i] = new
                changed_any = True
    if not changed_any:
        return False, ""
    if not dry_run:
        path.write_text("\n".join(lines) + ("\n" if text.endswith("\n") else ""), encoding="utf-8")
    return True, "\n".join(lines)


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description="商业联系邮箱单源同步")
    ap.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1],
                    help="meta 根仓路径（默认脚本上级目录）")
    ap.add_argument("--check", action="store_true", help="只读校验，有漂移则 exit 1")
    ap.add_argument("--dry-run", action="store_true", help="只打印将改动，不写盘")
    ap.add_argument("--repo", default="", help="仅处理指定子仓名（默认全部）")
    args = ap.parse_args(argv)

    root: Path = args.root
    emails = ssot_emails(root)
    if emails.get("README") and emails.get("COMMERCIAL") and emails["README"] != emails["COMMERCIAL"]:
        print(f"ERROR: SSOT 不一致：README={emails['README']} COMMERCIAL={emails['COMMERCIAL']}，请先修正根仓", file=sys.stderr)
        return 2
    email = emails.get("README") or emails.get("COMMERCIAL")
    assert email

    subrepos = find_subrepos(root)
    drift = 0
    for repo in subrepos:
        name = repo.name
        if args.repo and name != args.repo:
            continue
        for fname in DOC_FILES:
            path = repo / fname
            if not path.exists():
                continue
            changed, _ = sync_file(path, email, args.dry_run or args.check)
            if changed:
                drift += 1
                marker = "WILL-CHANGE" if (args.dry_run or args.check) else "CHANGED"
                print(f"{marker} {name}/{fname}")
    if args.check:
        if drift:
            print(f"DRIFT: {drift} 个文件与 SSOT ({email}) 不一致")
            return 1
        print(f"OK: 全部副本与 SSOT ({email}) 一致")
    elif drift == 0:
        print(f"OK: 全部副本已与 SSOT ({email}) 一致")
    else:
        print(f"SYNCED: {drift} 个文件已更新为 SSOT ({email})")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
