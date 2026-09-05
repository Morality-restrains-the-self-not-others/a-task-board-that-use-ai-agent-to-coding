"""collect-deploy-binaries.sh copies Release payloads into one directory."""

from __future__ import annotations

import hashlib
import io
import os
import subprocess
import tarfile
from pathlib import Path

import yaml

SCRIPTS = Path(__file__).resolve().parents[1]
COLLECT = SCRIPTS / "collect-deploy-binaries.sh"


def test_collect_default_ram_deploy_is_home_daydaymoney():
    text = COLLECT.read_text(encoding="utf-8")
    assert "${RAM_DEPLOY:-${DEPLOY_ROOT:-$HOME/bin/daydaymoney-deploy}}" in text
    assert "/tmp/ram-deploy" not in text


def _write(p: Path, text: str = "ok\n") -> None:
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(text, encoding="utf-8")


def _tiny_targz(path: Path, members: dict[str, str]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with tarfile.open(path, "w:gz") as tar:
        for name, text in members.items():
            data = text.encode("utf-8")
            info = tarfile.TarInfo(name=name)
            info.size = len(data)
            tar.addfile(info, io.BytesIO(data))


def _tar_names(path: Path) -> set[str]:
    with tarfile.open(path, "r:gz") as tar:
        return {n.lstrip("./") for n in tar.getnames() if n not in (".", "./")}


def _env(tmp_path: Path, **extra: str) -> dict[str, str]:
    drop = {
        "COLLECT_SRC",
        "COLLECT_STAGING",
        "COLLECT_SOFT",
        "RAM_DEPLOY",
        "META_ROOT",
        "DEPLOY_ROOT",
        "CONFIG_REPO",
        "SKIP_COLLECT_DEPLOY_BINARIES",
        "COLLECT_DEPLOY_BINARIES_REQUIRED",
        "SESSION_META_ROOT",
        "CI",
        "COLLECT_SKIP_SHA",
    }
    env = {k: v for k, v in os.environ.items() if k not in drop}
    env["COLLECT_SRC"] = str(tmp_path / "stage")
    env["COLLECT_STAGING"] = str(tmp_path / "no-staging")
    env["RAM_DEPLOY"] = str(tmp_path / "empty-ram")
    env["META_ROOT"] = str(tmp_path / "meta")
    env.update(extra)
    return env


def test_collect_copies_named_payloads(tmp_path):
    stage = tmp_path / "stage"
    dest = tmp_path / "out"
    _write(stage / "runAll", "elf-runall\n")
    (stage / "runAll").chmod(0o755)
    _write(stage / "taskAuth", "elf-auth\n")
    _write(stage / "taskEvents-bin.tar.gz", "fake-tar\n")
    (tmp_path / "empty-ram").mkdir()
    (tmp_path / "meta").mkdir()

    r = subprocess.run(
        ["bash", str(COLLECT), str(dest)],
        env=_env(tmp_path),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    assert (dest / "runAll").read_text(encoding="utf-8") == "elf-runall\n"
    assert (dest / "taskAuth").read_text(encoding="utf-8") == "elf-auth\n"
    assert (dest / "taskEvents-bin.tar.gz").read_text(encoding="utf-8") == "fake-tar\n"
    assert (dest / "MANIFEST.txt").is_file()


def test_collect_fails_without_runall(tmp_path):
    stage = tmp_path / "stage"
    stage.mkdir()
    dest = tmp_path / "out"
    (tmp_path / "empty-ram").mkdir()
    (tmp_path / "meta").mkdir()
    r = subprocess.run(
        ["bash", str(COLLECT), str(dest)],
        env=_env(tmp_path),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode != 0
    assert "missing runAll" in (r.stderr + r.stdout)


def test_collect_skips_unchanged_file(tmp_path):
    stage = tmp_path / "stage"
    dest = tmp_path / "out"
    _write(stage / "runAll", "same\n")
    (stage / "runAll").chmod(0o755)
    (tmp_path / "empty-ram").mkdir()
    (tmp_path / "meta").mkdir()
    env = _env(tmp_path)
    subprocess.run(["bash", str(COLLECT), str(dest)], env=env, check=True, capture_output=True)
    first_mtime = (dest / "runAll").stat().st_mtime
    r = subprocess.run(
        ["bash", str(COLLECT), str(dest)],
        env=env,
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    assert "unchanged runAll" in (r.stderr + r.stdout)
    assert (dest / "runAll").stat().st_mtime == first_mtime


def test_collect_picks_newer_source(tmp_path):
    dest = tmp_path / "out"
    staging = tmp_path / "staging"
    ram = tmp_path / "ram"
    _write(staging / "runAll", "older\n")
    (staging / "runAll").chmod(0o755)
    _write(ram / "artifacts" / "runAll", "newer\n")
    (ram / "artifacts" / "runAll").chmod(0o755)
    os.utime(staging / "runAll", (1_700_000_000, 1_700_000_000))
    os.utime(ram / "artifacts" / "runAll", (1_800_000_000, 1_800_000_000))
    (tmp_path / "meta").mkdir()
    r = subprocess.run(
        ["bash", str(COLLECT), str(dest)],
        env=_env(
            tmp_path,
            COLLECT_SRC="",
            COLLECT_STAGING=str(staging),
            RAM_DEPLOY=str(ram),
        ),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    assert (dest / "runAll").read_text(encoding="utf-8") == "newer\n"


def test_collect_soft_missing_runall(tmp_path):
    (tmp_path / "stage").mkdir()
    (tmp_path / "empty-ram").mkdir()
    (tmp_path / "meta").mkdir()
    dest = tmp_path / "out"
    r = subprocess.run(
        ["bash", str(COLLECT), str(dest)],
        env=_env(tmp_path, COLLECT_SOFT="1"),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    assert "missing runAll" in (r.stderr + r.stdout)


def test_collect_repacks_taskevents_when_live_has_more_workers(tmp_path):
    dest = tmp_path / "out"
    dest.mkdir()
    staging = tmp_path / "staging"
    stale = {
        "old/1_x/task-events-old": "old\n",
    }
    _tiny_targz(staging / "taskEvents-bin.tar.gz", stale)
    _tiny_targz(dest / "taskEvents-bin.tar.gz", stale)
    meta = tmp_path / "meta"
    _write(meta / "bin" / "runAll", "elf\n")
    (meta / "bin" / "runAll").chmod(0o755)
    live = meta / "taskEvents" / "bin"
    old_bin = live / "old" / "1_x" / "task-events-old"
    new_bin = (
        live
        / "project_deleted"
        / "1_detach_task_projects"
        / "task-events-project-deleted-1-detach-task-projects"
    )
    _write(old_bin, "old\n")
    _write(new_bin, "new-worker\n")
    old_bin.chmod(0o755)
    new_bin.chmod(0o755)
    (tmp_path / "empty-ram").mkdir()
    r = subprocess.run(
        ["bash", str(COLLECT), str(dest)],
        env=_env(
            tmp_path,
            COLLECT_SRC="",
            COLLECT_STAGING=str(staging),
            RAM_DEPLOY=str(tmp_path / "empty-ram"),
            META_ROOT=str(meta),
        ),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    names = _tar_names(dest / "taskEvents-bin.tar.gz")
    assert any("project_deleted/1_detach_task_projects" in n for n in names), names
    assert "packed taskEvents-bin.tar.gz" in (r.stderr + r.stdout)


def test_collect_skips_taskevents_repack_when_worker_count_matches(tmp_path):
    dest = tmp_path / "out"
    dest.mkdir()
    meta = tmp_path / "meta"
    _write(meta / "bin" / "runAll", "elf\n")
    (meta / "bin" / "runAll").chmod(0o755)
    live_bin = (
        meta
        / "taskEvents"
        / "bin"
        / "old"
        / "1_x"
        / "task-events-old"
    )
    _write(live_bin, "old\n")
    live_bin.chmod(0o755)
    _tiny_targz(
        dest / "taskEvents-bin.tar.gz",
        {"old/1_x/task-events-old": "old\n"},
    )
    (dest / "taskEvents-bin.tar.gz.workers").write_text("1\n", encoding="utf-8")
    future = 2_000_000_000
    os.utime(dest / "taskEvents-bin.tar.gz", (future, future))
    os.utime(live_bin, (1_700_000_000, 1_700_000_000))
    before = (dest / "taskEvents-bin.tar.gz").read_bytes()
    (tmp_path / "empty-ram").mkdir()
    (tmp_path / "stage").mkdir()
    r = subprocess.run(
        ["bash", str(COLLECT), str(dest)],
        env=_env(
            tmp_path,
            COLLECT_SRC="",
            COLLECT_STAGING=str(tmp_path / "no-staging"),
            RAM_DEPLOY=str(tmp_path / "empty-ram"),
            META_ROOT=str(meta),
        ),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    assert "unchanged taskEvents-bin.tar.gz" in (r.stderr + r.stdout)
    assert (dest / "taskEvents-bin.tar.gz").read_bytes() == before


def test_collect_packs_provider_frontend_dist_from_meta(tmp_path):
    dest = tmp_path / "out"
    meta = tmp_path / "meta"
    _write(meta / "bin" / "runAll", "elf\n")
    (meta / "bin" / "runAll").chmod(0o755)
    _write(meta / "taskAiProvider" / "frontend" / "dist" / "index.html", "<html>provider</html>\n")
    _write(meta / "taskAiProvider" / "frontend" / "dist" / "assets" / "app.js", "console.log(1)\n")
    (tmp_path / "empty-ram").mkdir()
    (tmp_path / "stage").mkdir()
    r = subprocess.run(
        ["bash", str(COLLECT), str(dest)],
        env=_env(
            tmp_path,
            COLLECT_SRC="",
            COLLECT_STAGING=str(tmp_path / "no-staging"),
            RAM_DEPLOY=str(tmp_path / "empty-ram"),
            META_ROOT=str(meta),
        ),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    archive = dest / "taskAiProvider-frontend-dist.tar.gz"
    assert archive.is_file(), r.stderr + r.stdout
    names = _tar_names(archive)
    assert "index.html" in names, names
    assert any(n.endswith("app.js") for n in names), names
    assert "packed taskAiProvider-frontend-dist.tar.gz" in (r.stderr + r.stdout)


def test_collect_repacks_provider_frontend_when_live_newer(tmp_path):
    dest = tmp_path / "out"
    dest.mkdir()
    _tiny_targz(dest / "taskAiProvider-frontend-dist.tar.gz", {"index.html": "<html>stale</html>\n"})
    os.utime(dest / "taskAiProvider-frontend-dist.tar.gz", (1_700_000_000, 1_700_000_000))
    meta = tmp_path / "meta"
    _write(meta / "bin" / "runAll", "elf\n")
    (meta / "bin" / "runAll").chmod(0o755)
    live = meta / "taskAiProvider" / "frontend" / "dist" / "index.html"
    _write(live, "<html>fresh</html>\n")
    os.utime(live, (1_800_000_000, 1_800_000_000))
    (tmp_path / "empty-ram").mkdir()
    (tmp_path / "stage").mkdir()
    r = subprocess.run(
        ["bash", str(COLLECT), str(dest)],
        env=_env(
            tmp_path,
            COLLECT_SRC="",
            COLLECT_STAGING=str(tmp_path / "no-staging"),
            RAM_DEPLOY=str(tmp_path / "empty-ram"),
            META_ROOT=str(meta),
        ),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    names = _tar_names(dest / "taskAiProvider-frontend-dist.tar.gz")
    assert "index.html" in names, names
    with tarfile.open(dest / "taskAiProvider-frontend-dist.tar.gz", "r:gz") as tar:
        member = next(n for n in tar.getnames() if n.lstrip("./").endswith("index.html"))
        body = tar.extractfile(member).read().decode("utf-8")
    assert body == "<html>fresh</html>\n"
    assert "packed taskAiProvider-frontend-dist.tar.gz" in (r.stderr + r.stdout)


def test_collect_repacks_taskfe_dist_when_live_has_more_files(tmp_path):
    """OPT-20260831-017: 已有 taskFE-dist.tar.gz 但 live public 文件数更多时必须重打包。"""
    dest = tmp_path / "out"
    dest.mkdir()
    _tiny_targz(dest / "taskFE-dist.tar.gz", {"index.html": "<html>stale</html>\n"})
    # mtime 设为未来，确保触发条件是文件数差异而非 mtime。
    os.utime(dest / "taskFE-dist.tar.gz", (2_000_000_000, 2_000_000_000))
    meta = tmp_path / "meta"
    _write(meta / "bin" / "runAll", "elf\n")
    (meta / "bin" / "runAll").chmod(0o755)
    live = meta / "taskFE" / "app" / "public"
    _write(live / "index.html", "<html>fresh</html>\n")
    _write(live / "assets" / "app.js", "console.log(1)\n")
    (tmp_path / "empty-ram").mkdir()
    (tmp_path / "stage").mkdir()
    r = subprocess.run(
        ["bash", str(COLLECT), str(dest)],
        env=_env(
            tmp_path,
            COLLECT_SRC="",
            COLLECT_STAGING=str(tmp_path / "no-staging"),
            RAM_DEPLOY=str(tmp_path / "empty-ram"),
            META_ROOT=str(meta),
        ),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    names = _tar_names(dest / "taskFE-dist.tar.gz")
    assert "index.html" in names, names
    assert any(n.endswith("app.js") for n in names), names
    assert "packed taskFE-dist.tar.gz" in (r.stderr + r.stdout)


def test_collect_packs_taskfe_when_only_html_symlink_index_exists(tmp_path):
    """atomic-vite-build 只写 public/html/index.html，不得因缺根 index.html 跳过源码树。"""
    dest = tmp_path / "out"
    dest.mkdir()
    _tiny_targz(dest / "taskFE-dist.tar.gz", {"index.html": "<html>stale</html>\n"})
    os.utime(dest / "taskFE-dist.tar.gz", (2_000_000_000, 2_000_000_000))
    meta = tmp_path / "meta"
    _write(meta / "bin" / "runAll", "elf\n")
    (meta / "bin" / "runAll").chmod(0o755)
    live = meta / "taskFE" / "app" / "public"
    release = live / "releases" / "20260902142508"
    _write(release / "index.html", "<html>badge-fix</html>\n")
    _write(release / "TaskDetailContent.logic.js", "checkFailedRepoUrls\n")
    (live / "html").symlink_to("releases/20260902142508")
    (tmp_path / "empty-ram").mkdir()
    (tmp_path / "stage").mkdir()
    r = subprocess.run(
        ["bash", str(COLLECT), str(dest)],
        env=_env(
            tmp_path,
            COLLECT_SRC="",
            COLLECT_STAGING=str(tmp_path / "no-staging"),
            RAM_DEPLOY=str(tmp_path / "empty-ram"),
            META_ROOT=str(meta),
        ),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    names = _tar_names(dest / "taskFE-dist.tar.gz")
    assert any(n.endswith("index.html") for n in names), names
    assert any("TaskDetailContent.logic.js" in n for n in names), names
    assert "packed taskFE-dist.tar.gz" in (r.stderr + r.stdout)
    with tarfile.open(dest / "taskFE-dist.tar.gz", "r:gz") as tar:
        member = next(n for n in tar.getnames() if n.lstrip("./").endswith("index.html"))
        body = tar.extractfile(member).read().decode("utf-8")
    assert "badge-fix" in body


def _write_releases_yaml(meta: Path, artifacts: dict) -> None:
    conf_ex = meta / "conf.example"
    conf_ex.mkdir(parents=True, exist_ok=True)
    (conf_ex / "releases.yaml").write_text(
        yaml.safe_dump({"artifacts": artifacts}, allow_unicode=True),
        encoding="utf-8",
    )


def test_collect_verifies_matching_sha_against_releases_yaml(tmp_path):
    """OPT-20260831-016: 已拷文件 sha 与 releases.yaml pin 一致 → exit 0。"""
    dest = tmp_path / "out"
    stage = tmp_path / "stage"
    payload = "elf-runall\n"
    _write(stage / "runAll", payload)
    (stage / "runAll").chmod(0o755)
    meta = tmp_path / "meta"
    _write_releases_yaml(
        meta,
        {
            "runAll": {
                "package": "github://task2money/daydaymoney-deploy/runAll@deploy-tag",
                "sha": hashlib.sha256(payload.encode()).hexdigest(),
            }
        },
    )
    (tmp_path / "empty-ram").mkdir()
    r = subprocess.run(
        ["bash", str(COLLECT), str(dest)],
        env=_env(tmp_path),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    assert "1 个文件与 releases.yaml 一致" in (r.stderr + r.stdout)


def test_collect_fails_on_sha_mismatch(tmp_path):
    """OPT-20260831-016: 拷入的 ELF 与 pin sha 不一致 → 非零退出。"""
    dest = tmp_path / "out"
    stage = tmp_path / "stage"
    _write(stage / "runAll", "elf-runall\n")
    (stage / "runAll").chmod(0o755)
    meta = tmp_path / "meta"
    _write_releases_yaml(
        meta,
        {
            "runAll": {
                "package": "github://task2money/daydaymoney-deploy/runAll@deploy-tag",
                "sha": "0" * 64,
            }
        },
    )
    (tmp_path / "empty-ram").mkdir()
    r = subprocess.run(
        ["bash", str(COLLECT), str(dest)],
        env=_env(tmp_path),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode != 0, r.stderr + r.stdout
    assert "SHA MISMATCH" in (r.stderr + r.stdout)


def test_collect_skip_sha_allows_mismatch(tmp_path):
    dest = tmp_path / "out"
    stage = tmp_path / "stage"
    _write(stage / "runAll", "elf-runall\n")
    (stage / "runAll").chmod(0o755)
    meta = tmp_path / "meta"
    _write_releases_yaml(
        meta,
        {
            "runAll": {
                "package": "github://task2money/daydaymoney-deploy/runAll@deploy-tag",
                "sha": "0" * 64,
            }
        },
    )
    (tmp_path / "empty-ram").mkdir()
    r = subprocess.run(
        ["bash", str(COLLECT), str(dest)],
        env=_env(tmp_path, COLLECT_SKIP_SHA="1"),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    assert "skip sha-verify" in (r.stderr + r.stdout)
