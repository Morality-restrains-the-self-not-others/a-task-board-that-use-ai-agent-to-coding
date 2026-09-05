"""up.sh installs hand-copied artifacts without GitHub."""

from __future__ import annotations

import io
import os
import shutil
import subprocess
import sys
import tarfile
import time
from pathlib import Path

_TESTS = Path(__file__).resolve().parent
if str(_TESTS) not in sys.path:
    sys.path.insert(0, str(_TESTS))
from test_up_from_config_repo import INSTALL, _clean_env, _seed_repo, _write


def _tiny_targz(path: Path, members: dict[str, str | bytes]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with tarfile.open(path, "w:gz") as tar:
        for name, text in members.items():
            data = text.encode("utf-8") if isinstance(text, str) else text
            info = tarfile.TarInfo(name=name)
            info.size = len(data)
            tar.addfile(info, io.BytesIO(data))


def _hold_running_elf(path: Path) -> subprocess.Popen:
    path.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy(os.path.realpath(sys.executable), path)
    path.chmod(0o755)
    proc = subprocess.Popen(
        [str(path), "-c", "import time; time.sleep(60)"],
        start_new_session=True,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
    deadline = time.time() + 2
    while time.time() < deadline:
        if proc.poll() is None:
            return proc
        time.sleep(0.05)
    raise AssertionError("holder ELF exited immediately")


def _fake_gh_must_not_run(path: Path) -> None:
    path.write_text(
        "#!/usr/bin/env bash\necho 'github must not run' >&2\nexit 1\n",
        encoding="utf-8",
    )
    path.chmod(0o755)


def test_up_installs_from_clone_artifacts_without_github(tmp_path):
    assert INSTALL.is_file()
    repo = tmp_path / "daydaymoney-deploy"
    _seed_repo(repo)
    _write(
        repo / "envs" / "current" / "releases.yaml",
        "artifacts:\n  runAll:\n    package: github://example/x/runAll@tag\n",
    )
    arts = repo / "artifacts"
    _write(arts / "runAll", "local-runall\n")
    (arts / "runAll").chmod(0o755)
    _write(arts / "taskAuth", "local-auth\n")
    (arts / "taskAuth").chmod(0o755)
    _tiny_targz(arts / "taskEvents-bin.tar.gz", {"worker": "evt\n"})
    _tiny_targz(arts / "taskFE-dist.tar.gz", {"index.html": "<html>fe</html>\n"})
    _tiny_targz(
        arts / "taskAiProvider-frontend-dist.tar.gz",
        {"index.html": "<html>provider</html>\n"},
    )
    fake_gh = tmp_path / "fake-gh"
    _fake_gh_must_not_run(fake_gh)

    r = subprocess.run(
        ["bash", str(repo / "scripts" / "up.sh")],
        env=_clean_env(SYNC_ARTIFACTS="1", START="0", GH=str(fake_gh), CURL="/bin/false"),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    assert (repo / "bin" / "runAll").read_text(encoding="utf-8") == "local-runall\n"
    assert (repo / "bin" / "taskAuth").read_text(encoding="utf-8") == "local-auth\n"
    assert (repo / "taskEvents" / "bin" / "worker").read_text(encoding="utf-8") == "evt\n"
    assert (repo / "taskFE" / "app" / "public" / "index.html").read_text(
        encoding="utf-8"
    ) == "<html>fe</html>\n"
    assert (repo / "taskAiProvider" / "frontend" / "dist" / "index.html").read_text(
        encoding="utf-8"
    ) == "<html>provider</html>\n"
    assert "github must not run" not in (r.stderr + r.stdout)
    assert "skip GitHub deploy-sync" in (r.stderr + r.stdout)


def test_up_installs_from_deploy_binaries_dir(tmp_path):
    repo = tmp_path / "daydaymoney-deploy"
    _seed_repo(repo)
    src = repo / "deploy-binaries"
    _write(src / "runAll", "from-deploy-binaries\n")
    (src / "runAll").chmod(0o755)
    fake_gh = tmp_path / "fake-gh"
    _fake_gh_must_not_run(fake_gh)
    r = subprocess.run(
        ["bash", str(repo / "scripts" / "up.sh")],
        env=_clean_env(SYNC_ARTIFACTS="1", START="0", GH=str(fake_gh), CURL="/bin/false"),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    assert (repo / "bin" / "runAll").read_text(encoding="utf-8") == "from-deploy-binaries\n"
    assert (repo / "artifacts" / "runAll").is_file()


def test_up_refuses_escaping_tarball(tmp_path):
    repo = tmp_path / "daydaymoney-deploy"
    _seed_repo(repo)
    arts = repo / "artifacts"
    _write(arts / "runAll", "ok\n")
    (arts / "runAll").chmod(0o755)
    _tiny_targz(arts / "taskEvents-bin.tar.gz", {"../evil": "x\n"})
    r = subprocess.run(
        ["bash", str(repo / "scripts" / "up.sh")],
        env=_clean_env(SYNC_ARTIFACTS="0", START="0"),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode != 0
    assert "path escape" in (r.stderr + r.stdout)


def test_install_fails_when_taskevents_archive_missing_intent(tmp_path):
    root = tmp_path / "deploy"
    arts = tmp_path / "arts"
    _write(
        root / "taskEvents" / "run.sh",
        "#!/bin/bash\nINTENT_PATHS=(\n  project_deleted/1_detach_task_projects\n)\n",
    )
    _write(arts / "runAll", "ok\n")
    (arts / "runAll").chmod(0o755)
    _tiny_targz(
        arts / "taskEvents-bin.tar.gz",
        {"workspace_machine_idle/1_recycle/task-events-x": "x\n"},
    )
    r = subprocess.run(
        ["bash", str(INSTALL), str(arts), str(root)],
        env=_clean_env(),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode != 0
    assert "project_deleted/1_detach_task_projects" in (r.stderr + r.stdout)
    assert "missing worker" in (r.stderr + r.stdout)


def test_install_accepts_taskevents_archive_with_listed_intent(tmp_path):
    root = tmp_path / "deploy"
    arts = tmp_path / "arts"
    _write(
        root / "taskEvents" / "run.sh",
        "#!/bin/bash\nINTENT_PATHS=(\n  project_deleted/1_detach_task_projects\n)\n",
    )
    _write(arts / "runAll", "ok\n")
    (arts / "runAll").chmod(0o755)
    rel = (
        "project_deleted/1_detach_task_projects/"
        "task-events-project-deleted-1-detach-task-projects"
    )
    _tiny_targz(arts / "taskEvents-bin.tar.gz", {rel: "ok\n"})
    r = subprocess.run(
        ["bash", str(INSTALL), str(arts), str(root)],
        env=_clean_env(),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    assert (root / "taskEvents" / "bin" / Path(rel)).read_text(encoding="utf-8") == "ok\n"


def test_install_replaces_running_elf(tmp_path):
    root = tmp_path / "deploy"
    arts = tmp_path / "arts"
    busy = root / "bin" / "go_relayToTrae"
    proc = _hold_running_elf(busy)
    try:
        arts.mkdir(parents=True)
        shutil.copy(os.path.realpath("/bin/true"), arts / "go_relayToTrae")
        (arts / "go_relayToTrae").chmod(0o755)
        r = subprocess.run(
            ["bash", str(INSTALL), str(arts), str(root)],
            env=_clean_env(),
            capture_output=True,
            text=True,
            check=False,
        )
        assert r.returncode == 0, r.stderr + r.stdout
        assert (root / "bin" / "go_relayToTrae").read_bytes() == (
            arts / "go_relayToTrae"
        ).read_bytes()
        assert proc.poll() is None
    finally:
        proc.terminate()
        proc.wait(timeout=5)


def test_install_unpacks_over_running_worker(tmp_path):
    root = tmp_path / "deploy"
    arts = tmp_path / "arts"
    rel = (
        "project_deleted/1_detach_task_projects/"
        "task-events-project-deleted-1-detach-task-projects"
    )
    _write(
        root / "taskEvents" / "run.sh",
        "#!/bin/bash\nINTENT_PATHS=(\n  project_deleted/1_detach_task_projects\n)\n",
    )
    busy = root / "taskEvents" / "bin" / Path(rel)
    proc = _hold_running_elf(busy)
    try:
        _tiny_targz(
            arts / "taskEvents-bin.tar.gz",
            {rel: Path(os.path.realpath("/bin/true")).read_bytes()},
        )
        r = subprocess.run(
            ["bash", str(INSTALL), str(arts), str(root)],
            env=_clean_env(),
            capture_output=True,
            text=True,
            check=False,
        )
        assert r.returncode == 0, r.stderr + r.stdout
        assert (root / "taskEvents" / "bin" / Path(rel)).read_bytes() == Path(
            os.path.realpath("/bin/true")
        ).read_bytes()
        assert proc.poll() is None
    finally:
        proc.terminate()
        proc.wait(timeout=5)


def test_install_unpacks_provider_frontend_dist_without_pin(tmp_path):
    root = tmp_path / "deploy"
    root.mkdir()
    arts = tmp_path / "arts"
    _write(arts / "runAll", "ok\n")
    (arts / "runAll").chmod(0o755)
    _tiny_targz(
        arts / "taskAiProvider-frontend-dist.tar.gz",
        {"index.html": "<html>vendor-spa</html>\n", "assets/app.js": "ok\n"},
    )
    r = subprocess.run(
        ["bash", str(INSTALL), str(arts), str(root)],
        env=_clean_env(),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    assert (root / "taskAiProvider" / "frontend" / "dist" / "index.html").read_text(
        encoding="utf-8"
    ) == "<html>vendor-spa</html>\n"
    assert (root / "taskAiProvider" / "frontend" / "dist" / "assets" / "app.js").read_text(
        encoding="utf-8"
    ) == "ok\n"
