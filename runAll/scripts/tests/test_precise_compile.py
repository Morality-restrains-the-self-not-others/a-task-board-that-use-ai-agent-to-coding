"""precise-compile.sh builds registered services into deploy-binaries/."""

from __future__ import annotations

import json
import os
import subprocess
import sys
from pathlib import Path

SCRIPTS = Path(__file__).resolve().parents[1]
PREC_PY = SCRIPTS / "precise_compile.py"
PREC_SH = SCRIPTS / "precise-compile.sh"
WRAPPER = SCRIPTS.parents[1] / "scripts" / "precise-compile.sh"

sys.path.insert(0, str(SCRIPTS))
from precise_compile import (  # noqa: E402
    load_services,
    read_registry,
    select_services,
    working_dir_alias,
)

MIN_YAML = """
groups:
  - name: infrastructure
    services:
      - name: docker-mysql
        start_command: "bash dockerInfra/mysql/run.sh start"
        working_dir: .
  - name: platform
    services:
      - name: task-auth
        build_command: "./build.sh"
        start_command: "./bin/taskAuth"
        working_dir: taskAuth
      - name: taskFE
        build_command: "bash scripts/runall-lifecycle.sh build"
        start_command: "bash scripts/runall-lifecycle.sh start"
        working_dir: taskFE/app
"""


def test_working_dir_alias_first_segment():
    assert working_dir_alias("taskFE/app") == "taskFE"
    assert working_dir_alias("taskAuth") == "taskAuth"
    assert working_dir_alias(".") == ""


def test_load_services_reads_build_command():
    svcs = load_services(MIN_YAML)
    by_name = {s["name"]: s for s in svcs}
    assert by_name["task-auth"]["working_dir"] == "taskAuth"
    assert by_name["task-auth"]["build_command"] == "./build.sh"
    assert by_name["task-auth"]["alias"] == "taskAuth"
    assert by_name["docker-mysql"]["build_command"] == ""


def test_select_by_runall_name_or_dir_alias():
    svcs = load_services(MIN_YAML)
    by_auth = select_services(svcs, ["task-auth"])
    assert [s["name"] for s in by_auth] == ["task-auth"]
    by_alias = select_services(svcs, ["taskAuth"])
    assert [s["name"] for s in by_alias] == ["task-auth"]
    by_fe = select_services(svcs, ["taskFE"])
    assert [s["name"] for s in by_fe] == ["taskFE"]


def test_select_all_skips_services_without_build_command():
    svcs = load_services(MIN_YAML)
    got = select_services(svcs, None, all_buildable=True)
    names = [s["name"] for s in got]
    assert "task-auth" in names
    assert "taskFE" in names
    assert "docker-mysql" not in names


def test_read_registry_strips_comments(tmp_path):
    p = tmp_path / "precise_restart_services.txt"
    p.write_text("# hi\n\ntask-auth\ntaskFE\n", encoding="utf-8")
    assert read_registry(p) == ["task-auth", "taskFE"]


def test_cli_resolve_json(tmp_path):
    conf = tmp_path / "runAll.yaml"
    conf.write_text(MIN_YAML, encoding="utf-8")
    r = subprocess.run(
        [sys.executable, str(PREC_PY), "--conf", str(conf), "--names", "taskAuth", "--json"],
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr
    rows = json.loads(r.stdout)
    assert rows[0]["name"] == "task-auth"
    assert rows[0]["build_command"] == "./build.sh"


def test_cli_empty_selection_nonzero(tmp_path):
    conf = tmp_path / "runAll.yaml"
    conf.write_text(MIN_YAML, encoding="utf-8")
    empty = tmp_path / "empty.txt"
    empty.write_text("", encoding="utf-8")
    r = subprocess.run(
        [
            sys.executable,
            str(PREC_PY),
            "--conf",
            str(conf),
            "--registry",
            str(empty),
            "--json",
        ],
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode != 0


def _write(p: Path, text: str) -> None:
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(text, encoding="utf-8")


def _compile_env(meta: Path, dest: Path) -> dict[str, str]:
    drop = {
        "COLLECT_SRC",
        "COLLECT_STAGING",
        "COLLECT_SOFT",
        "RAM_DEPLOY",
        "META_ROOT",
        "DEPLOY_MODE",
        "PRECISE_COMPILE_DEST",
        "COLLECT_SKIP_SHA",
    }
    env = {k: v for k, v in os.environ.items() if k not in drop}
    env["META_ROOT"] = str(meta)
    env["PRECISE_COMPILE_DEST"] = str(dest)
    env["RAM_DEPLOY"] = str(meta / "no-ram")
    env["COLLECT_STAGING"] = str(meta / "no-staging")
    env["COLLECT_SRC"] = ""
    return env


def test_precise_compile_runs_fake_build_and_collects(tmp_path):
    meta = tmp_path / "meta"
    dest = tmp_path / "deploy-binaries"
    _write(meta / "conf" / "runAll.yaml", MIN_YAML)
    _write(
        meta / "taskAuth" / "build.sh",
        "#!/usr/bin/env bash\nset -euo pipefail\nmkdir -p bin\nprintf compiled\\\\n > bin/taskAuth\nchmod +x bin/taskAuth\n",
    )
    (meta / "taskAuth" / "build.sh").chmod(0o755)
    _write(meta / "bin" / "runAll", "elf-runall\n")
    (meta / "bin" / "runAll").chmod(0o755)
    _write(meta / ".runall" / "precise_restart_services.txt", "task-auth\n")
    r = subprocess.run(
        ["bash", str(PREC_SH)],
        env=_compile_env(meta, dest),
        capture_output=True,
        text=True,
        check=False,
        cwd=str(meta),
    )
    assert r.returncode == 0, r.stderr + r.stdout
    assert (dest / "taskAuth").read_text(encoding="utf-8") == "compiled\n"
    assert (dest / "runAll").is_file()
    assert (dest / "MANIFEST.txt").is_file()
    assert "precise-compile" in (r.stderr + r.stdout)


def test_precise_compile_install_into_deploy_root(tmp_path):
    meta = tmp_path / "meta"
    dest = tmp_path / "deploy-binaries"
    deploy = tmp_path / "deploy"
    _write(meta / "conf" / "runAll.yaml", MIN_YAML)
    _write(
        meta / "taskAuth" / "build.sh",
        "#!/usr/bin/env bash\nset -euo pipefail\nmkdir -p bin\nprintf compiled\\\\n > bin/taskAuth\nchmod +x bin/taskAuth\n",
    )
    (meta / "taskAuth" / "build.sh").chmod(0o755)
    _write(meta / "bin" / "runAll", "elf-runall\n")
    (meta / "bin" / "runAll").chmod(0o755)
    _write(meta / ".runall" / "precise_restart_services.txt", "task-auth\n")
    deploy.mkdir(parents=True, exist_ok=True)
    env = _compile_env(meta, dest)
    env["DEPLOY_ROOT"] = str(deploy)
    r = subprocess.run(
        ["bash", str(PREC_SH), "--install"],
        env=env,
        capture_output=True,
        text=True,
        check=False,
        cwd=str(meta),
    )
    assert r.returncode == 0, r.stderr + r.stdout
    # collect 仍写源码仓 deploy-binaries
    assert (dest / "taskAuth").read_text(encoding="utf-8") == "compiled\n"
    # --install 把 ELF 装进 deploy root 的 bin/
    assert (deploy / "bin" / "taskAuth").read_text(encoding="utf-8") == "compiled\n"
    assert (deploy / "bin" / "runAll").is_file()
    assert "install" in (r.stderr + r.stdout)


def test_precise_compile_skips_release_sha_pin(tmp_path):
    meta = tmp_path / "meta"
    dest = tmp_path / "deploy-binaries"
    _write(meta / "conf" / "runAll.yaml", MIN_YAML)
    _write(
        meta / "taskAuth" / "build.sh",
        "#!/usr/bin/env bash\nmkdir -p bin\necho x > bin/taskAuth\nchmod +x bin/taskAuth\n",
    )
    (meta / "taskAuth" / "build.sh").chmod(0o755)
    _write(meta / "bin" / "runAll", "elf-runall\n")
    (meta / "bin" / "runAll").chmod(0o755)
    conf_ex = meta / "conf.example"
    conf_ex.mkdir(parents=True, exist_ok=True)
    (conf_ex / "releases.yaml").write_text(
        "artifacts:\n  runAll:\n    package: github://x/runAll@t\n    sha: "
        + ("0" * 64)
        + "\n",
        encoding="utf-8",
    )
    r = subprocess.run(
        ["bash", str(PREC_SH), "task-auth"],
        env=_compile_env(meta, dest),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    assert "SHA MISMATCH" not in (r.stderr + r.stdout)


def test_wrapper_delegates_to_runall_script():
    text = WRAPPER.read_text(encoding="utf-8")
    assert "runAll/scripts/precise-compile.sh" in text
