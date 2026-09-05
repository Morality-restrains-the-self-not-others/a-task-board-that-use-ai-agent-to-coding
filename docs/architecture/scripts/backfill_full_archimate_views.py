#!/usr/bin/env python3
"""Backfill OPT-20260904-011: rebuild v126–v132 .full.archimate with inherited Views.

Strategy (IDs were reissued per version, so ID-stable merge_full cannot update
shared concepts — we accumulate):
  1. Start from last good .full (v125).
  2. For each later version N: copy prior full → insert missing elements/rels
     from N.slice .full (+diff) by id → append missing ArchimateDiagramModel
     views from N.slice .full by name → write N.full → become next prior.

Usage (from repo root):
  python3 docs/architecture/scripts/backfill_full_archimate_views.py
  python3 docs/architecture/scripts/backfill_full_archimate_views.py --dry-run
"""

from __future__ import annotations

import argparse
import re
from pathlib import Path

ARCH = Path(__file__).resolve().parents[1]

MODEL_NAME_RE = re.compile(r'(<archimate:model\b[^>]*\bname=")([^"]*)(")')

FOLDER_FOR_TYPE = {
    "Plateau": "implementation_migration",
    "Gap": "implementation_migration",
    "WorkPackage": "implementation_migration",
    "Deliverable": "implementation_migration",
}


def folder_type(etype: str) -> str:
    if etype in FOLDER_FOR_TYPE:
        return FOLDER_FOR_TYPE[etype]
    if etype.startswith("Business"):
        return "business"
    if etype.startswith("Application") or etype == "DataObject":
        return "application"
    if etype.startswith("Technology") or etype in (
        "Artifact",
        "SystemSoftware",
        "Device",
        "Node",
        "TechnologyService",
    ):
        return "technology"
    if "Relationship" in etype:
        return "relations"
    return "application"


def existing_ids(xml: str) -> set[str]:
    return set(re.findall(r'\bid="([^"]+)"', xml))


def diagram_names(xml: str) -> list[str]:
    return re.findall(r'ArchimateDiagramModel[^>]*name="([^"]+)"', xml)


def parse_self_closing_elements(xml: str) -> list[dict]:
    """Parse self-closing <element .../> with arbitrary attr order."""
    out: list[dict] = []
    for m in re.finditer(r"<element\s+([^>]+?)/>", xml):
        attrs_blob = m.group(1)
        if "ArchimateDiagramModel" in attrs_blob:
            continue
        typem = re.search(r'xsi:type="archimate:([^"]+)"', attrs_blob)
        if not typem:
            continue
        etype = typem.group(1)
        idm = re.search(r'\bid="([^"]+)"', attrs_blob)
        if not idm:
            continue
        namem = re.search(r'\bname="([^"]*)"', attrs_blob)
        srcm = re.search(r'\bsource="([^"]+)"', attrs_blob)
        tgtm = re.search(r'\btarget="([^"]+)"', attrs_blob)
        out.append(
            {
                "etype": etype,
                "eid": idm.group(1),
                "ename": namem.group(1) if namem else "",
                "source": srcm.group(1) if srcm else None,
                "target": tgtm.group(1) if tgtm else None,
            }
        )
    return out


def _extract_balanced_element(xml: str, start: int) -> str:
    assert xml.startswith("<element", start)
    i = xml.find(">", start) + 1
    if xml[i - 2] == "/":
        return xml[start:i]
    depth_el = 1
    depth_ch = 0
    while i < len(xml):
        if xml.startswith("</element>", i):
            if depth_ch == 0:
                depth_el -= 1
                i += len("</element>")
                if depth_el == 0:
                    return xml[start:i]
            else:
                i += len("</element>")
            continue
        if xml.startswith("<element", i):
            end = xml.find(">", i) + 1
            if xml[end - 2] != "/":
                depth_el += 1
            i = end
            continue
        if xml.startswith("</child>", i):
            depth_ch -= 1
            i += len("</child>")
            continue
        if xml.startswith("<child", i):
            end = xml.find(">", i) + 1
            if xml[end - 2] != "/":
                depth_ch += 1
            i = end
            continue
        if xml.startswith("<sourceConnection", i):
            i = xml.find(">", i) + 1
            continue
        if xml.startswith("</sourceConnection>", i):
            i += len("</sourceConnection>")
            continue
        if xml.startswith("<bounds", i):
            i = xml.find(">", i) + 1
            continue
        i += 1
    raise RuntimeError(f"unclosed element at {start}")


def extract_diagram_models(xml: str) -> list[tuple[str, str]]:
    out: list[tuple[str, str]] = []
    for m in re.finditer(
        r'<element\s+xsi:type="archimate:ArchimateDiagramModel"[^>]*>',
        xml,
    ):
        name_m = re.search(r'\bname="([^"]+)"', m.group(0))
        name = name_m.group(1) if name_m else ""
        out.append((name, _extract_balanced_element(xml, m.start())))
    return out


def insert_into_folder(xml: str, ftype: str, tag: str) -> str:
    m = re.search(
        rf'(<folder[^>]*type="{re.escape(ftype)}"[^>]*>)(.*?)(</folder>)',
        xml,
        re.S,
    )
    if m:
        return xml[: m.start(3)] + "\n    " + tag + xml[m.start(3) :]
    anchor = re.search(r'<folder[^>]*type="relations"[^>]*>', xml)
    if not anchor:
        anchor = re.search(r'<folder[^>]*type="diagrams"[^>]*>', xml)
    if not anchor:
        raise RuntimeError(f"cannot insert folder {ftype}")
    names = {
        "business": "Business",
        "application": "Application",
        "technology": "Technology",
        "implementation_migration": "Implementation &amp; Migration",
        "relations": "Relations",
    }
    block = (
        f'  <folder name="{names.get(ftype, ftype)}" id="f-{ftype}-bf" type="{ftype}">\n'
        f"    {tag}\n"
        f"  </folder>\n"
    )
    return xml[: anchor.start()] + block + xml[anchor.start() :]


def append_view(xml: str, view_xml: str) -> str:
    m = re.search(
        r'(<folder[^>]*type="diagrams"[^>]*>)(.*)(</folder>)',
        xml,
        re.S,
    )
    if not m:
        raise RuntimeError("no Views folder")
    return xml[: m.start(3)] + "\n" + view_xml + "\n  " + xml[m.start(3) :]


def merge_diff_into_full(
    base: str, diff: str, slice_full: str, version: int, title: str
) -> tuple[str, int, int, int]:
    out = base
    ids = existing_ids(out)
    inserted_e = inserted_r = 0

    seen_new: set[str] = set()
    to_insert: list[dict] = []
    for src_xml in (slice_full, diff):
        for el in parse_self_closing_elements(src_xml):
            if el["eid"] in ids or el["eid"] in seen_new:
                continue
            seen_new.add(el["eid"])
            to_insert.append(el)

    for el in to_insert:
        etype, eid, ename = el["etype"], el["eid"], el["ename"]
        src, tgt = el["source"], el["target"]
        if src is not None and tgt is not None:
            if ename:
                tag = (
                    f'<element xsi:type="archimate:{etype}" name="{ename}" '
                    f'id="{eid}" source="{src}" target="{tgt}"/>'
                )
            else:
                tag = (
                    f'<element xsi:type="archimate:{etype}" id="{eid}" '
                    f'source="{src}" target="{tgt}"/>'
                )
            out = insert_into_folder(out, "relations", tag)
            inserted_r += 1
        else:
            tag = f'<element xsi:type="archimate:{etype}" id="{eid}" name="{ename}"/>'
            out = insert_into_folder(out, folder_type(etype), tag)
            inserted_e += 1
        ids.add(eid)

    have = set(diagram_names(out))
    appended = 0
    for name, block in extract_diagram_models(slice_full):
        if name in have:
            continue
        collide = [i for i in re.findall(r'\bid="([^"]+)"', block) if i in ids]
        if collide:
            for cid in collide:
                new_id = f"bf{version}-{cid}"
                block = block.replace(f'id="{cid}"', f'id="{new_id}"')
                block = block.replace(f'source="{cid}"', f'source="{new_id}"')
                block = block.replace(f'target="{cid}"', f'target="{new_id}"')
                block = re.sub(
                    rf'(targetConnections="[^"]*)\b{re.escape(cid)}\b',
                    rf"\1{new_id}",
                    block,
                )
                ids.add(new_id)
        out = append_view(out, "    " + block)
        have.add(name)
        appended += 1
        ids.update(re.findall(r'\bid="([^"]+)"', block))

    out = MODEL_NAME_RE.sub(
        rf'\1{title} v{version} 全量 (post-change, views-inherited)\3',
        out,
        count=1,
    )
    return out, inserted_e, inserted_r, appended


CHAINS = {
    "enterprise-landscape": [
        (125, "archive/v125-enterprise-landscape-20260901-1405-cursor.full.archimate", None, None),
        (
            126,
            None,
            "v126-enterprise-landscape-20260901-2244-cursor.diff.archimate",
            "v126-enterprise-landscape-20260901-2244-cursor.full.archimate",
        ),
        (
            127,
            None,
            "v127-enterprise-landscape-20260902-1335-cursor.diff.archimate",
            "v127-enterprise-landscape-20260902-1335-cursor.full.archimate",
        ),
        (
            128,
            None,
            "v128-enterprise-landscape-20260902-1410-cursor.diff.archimate",
            "v128-enterprise-landscape-20260902-1410-cursor.full.archimate",
        ),
        (
            129,
            None,
            "v129-enterprise-landscape-20260903-0040-cursor.diff.archimate",
            "v129-enterprise-landscape-20260903-0040-cursor.full.archimate",
        ),
        (
            132,
            None,
            "v132-enterprise-landscape-20260904-1920-cursor.diff.archimate",
            "v132-enterprise-landscape-20260904-1920-cursor.full.archimate",
        ),
    ],
    "application-integration": [
        (125, "archive/v125-application-integration-20260901-1405-cursor.full.archimate", None, None),
        (
            126,
            None,
            "archive/v126-application-integration-20260901-2244-cursor.diff.archimate",
            "archive/v126-application-integration-20260901-2244-cursor.full.archimate",
        ),
        (
            127,
            None,
            "archive/v127-application-integration-20260902-1335-cursor.diff.archimate",
            "archive/v127-application-integration-20260902-1335-cursor.full.archimate",
        ),
        (
            128,
            None,
            "v128-application-integration-20260902-1410-cursor.diff.archimate",
            "v128-application-integration-20260902-1410-cursor.full.archimate",
        ),
        (
            129,
            None,
            "v129-application-integration-20260903-0040-cursor.diff.archimate",
            "v129-application-integration-20260903-0040-cursor.full.archimate",
        ),
        (
            130,
            None,
            "v130-application-integration-20260904-1520-cursor.diff.archimate",
            "v130-application-integration-20260904-1520-cursor.full.archimate",
        ),
        (
            131,
            None,
            "v131-application-integration-20260904-1815-cursor.diff.archimate",
            "v131-application-integration-20260904-1815-cursor.full.archimate",
        ),
        (
            132,
            None,
            "v132-application-integration-20260904-1920-cursor.diff.archimate",
            "v132-application-integration-20260904-1920-cursor.full.archimate",
        ),
    ],
}


def resolve(rel: str | None) -> Path | None:
    if not rel:
        return None
    p = ARCH / rel
    return p if p.is_file() else None


def run_chain(view: str, *, dry_run: bool) -> None:
    steps = CHAINS[view]
    base_rel = steps[0][1]
    assert base_rel
    prior = (ARCH / base_rel).read_text(encoding="utf-8")
    print(f"[{view}] base v{steps[0][0]} views={len(diagram_names(prior))}")

    for ver, _base, diff_rel, full_rel in steps[1:]:
        diff_p = resolve(diff_rel)
        full_p = resolve(full_rel)
        if not diff_p or not full_p:
            raise FileNotFoundError(
                f"missing diff/full for {view} v{ver}: {diff_rel} {full_rel}"
            )
        # Read slice BEFORE overwrite
        slice_full = full_p.read_text(encoding="utf-8")
        diff = diff_p.read_text(encoding="utf-8")
        merged, ie, ir, av = merge_diff_into_full(
            prior, diff, slice_full, ver, view
        )
        nviews = len(diagram_names(merged))
        print(
            f"[{view}] v{ver}: +elem={ie} +rel={ir} +views={av} "
            f"total_views={nviews} → {full_p.relative_to(ARCH)}"
        )
        if nviews < len(diagram_names(prior)) + av:
            raise RuntimeError(f"view count regression at {view} v{ver}")
        if not dry_run:
            full_p.write_text(merged, encoding="utf-8")
        prior = merged


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--dry-run", action="store_true")
    args = ap.parse_args(argv)
    for view in CHAINS:
        run_chain(view, dry_run=args.dry_run)
    print("OK" if not args.dry_run else "DRY-RUN OK")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
