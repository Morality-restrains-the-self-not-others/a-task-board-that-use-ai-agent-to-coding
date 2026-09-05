"""up-from-config-repo.sh overlays a manually provided secrets tree."""

from __future__ import annotations

import os
import subprocess
from pathlib import Path

SCRIPTS = Path(__file__).resolve().parents[1]
UP = SCRIPTS / "up-from-config-repo.sh"
LAYOUT = SCRIPTS / "layout-from-config-repo.sh"
INSTALL = SCRIPTS / "install-local-artifacts.sh"
WRITE_CUTOVER = SCRIPTS / "write-cutover-env.sh"

# OPT-20260901-004: cutover.env key set is SSOT; any generator drift must fail.
EXPECTED_CUTOVER_KEYS = {
    "export DEPLOY_ROOT",
    "export CONF_ROOT",
    "export MONOREPO_ROOT",
    "export DEPLOY_MODE",
    "export SOURCE_ROOT",
    "export RUNALL_SKIP_BUILD",
    "export RUNALL_BIN",
    "export RUNALL_CONFIG",
    "export RUNALL_CONSOLE_LOG",
    "export RUNALL_LOG_ROOT",
    "export RUNALL_OWNERSHIP_STORE",
    "export INFRA_HOST",
    "export GODEBUG",
}


def _cutover_keys(text: str) -> set[str]:
    keys = set()
    for line in text.splitlines():
        line = line.strip()
        if line.startswith("export "):
            keys.add(line.split("=", 1)[0])
    return keys


def _clean_env(**extra: str) -> dict[str, str]:
    drop = {
        "DEPLOY_ROOT",
        "CONFIG_REPO",
        "CONF_ROOT",
        "SECRETS_DIR",
        "ARTIFACTS_DIR",
        "TASK_EVENTS_BIN",
        "TASKFE_PUBLIC",
        "TASKAIPROVIDER_FRONTEND",
        "SYNC_ARTIFACTS",
        "START",
        "GH",
        "CURL",
        "INFRA_HOST",
        "GITHUB_TOKEN",
        "GH_TOKEN",
        "GH_DOWNLOAD_RETRIES",
        "GH_DOWNLOAD_SLEEP",
        "GITHUB_API_URL",
        "GH_GODEBUG_LOG",
        "GH_FAIL_COUNTER",
        "FORCE_DEPLOY_SYNC",
        "COLLECT_SRC",
        "RAM_DEPLOY",
        "META_ROOT",
    }
    env = {k: v for k, v in os.environ.items() if k not in drop}
    env.update(extra)
    env.setdefault("CURL", "/bin/false")
    return env


def _write(p: Path, text: str = "ok\n") -> None:
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(text, encoding="utf-8")
    if p.suffix == ".sh":
        p.chmod(0o755)


def _seed_repo(repo: Path) -> None:
    env = repo / "envs" / "current"
    _write(env / "conf" / "base.yaml", "scheme: https\n")
    _write(env / "conf" / "runAll.yaml", "groups: []\n")
    _write(env / "dockerInfra" / "mysql" / "run.sh")
    scripts = repo / "scripts"
    scripts.mkdir(parents=True)
    (scripts / "layout.sh").write_bytes(LAYOUT.read_bytes())
    (scripts / "layout.sh").chmod(0o755)
    (scripts / "up.sh").write_bytes(UP.read_bytes())
    (scripts / "up.sh").chmod(0o755)
    (scripts / "install-local-artifacts.sh").write_bytes(INSTALL.read_bytes())
    (scripts / "install-local-artifacts.sh").chmod(0o755)
    (scripts / "write-cutover-env.sh").write_bytes(WRITE_CUTOVER.read_bytes())
    (scripts / "write-cutover-env.sh").chmod(0o755)


def test_up_overlays_manual_secrets_tree(tmp_path):
    repo = tmp_path / "daydaymoney-deploy"
    _seed_repo(repo)
    secrets = repo / "secrets"
    _write(secrets / "conf-local" / "auth" / "task-auth" / "config.yaml", "sms_key: manual\n")
    _write(secrets / "conf-local" / "gateway" / "task-gateway" / "dev-gateway.pem", "PEM\n")
    _write(secrets / "conf-local" / "auth" / "task-auth" / "oidc_signing_key.pem", "KEY\n")
    _write(secrets / "taskGateway" / "certs" / "dev-gateway.pem", "LEGACY\n")
    _write(secrets / "db" / "task-auth" / "oidc_signing_key.pem", "LEGACY_KEY\n")

    r = subprocess.run(
        ["bash", str(repo / "scripts" / "up.sh")],
        env=_clean_env(SYNC_ARTIFACTS="0", START="0"),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    conf_local = repo / "conf-local" / "auth" / "task-auth" / "config.yaml"
    assert conf_local.read_text(encoding="utf-8") == "sms_key: manual\n"
    assert (repo / "conf-local" / "gateway" / "task-gateway" / "dev-gateway.pem").read_text(
        encoding="utf-8"
    ) == "PEM\n"
    assert (repo / "conf-local" / "auth" / "task-auth" / "oidc_signing_key.pem").read_text(
        encoding="utf-8"
    ) == "KEY\n"
    assert not (repo / "taskGateway" / "certs" / "dev-gateway.pem").exists()
    assert not (repo / "db" / "task-auth" / "oidc_signing_key.pem").exists()


def test_up_overlays_conf_local_tree(tmp_path):
    repo = tmp_path / "daydaymoney-deploy"
    _seed_repo(repo)
    secrets = repo / "secrets"
    _write(secrets / "conf-local" / "billing" / "paypal" / "config.yaml", "client_secret: from-local\n")

    r = subprocess.run(
        ["bash", str(repo / "scripts" / "up.sh")],
        env=_clean_env(SYNC_ARTIFACTS="0", START="0"),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    got = repo / "conf-local" / "billing" / "paypal" / "config.yaml"
    assert got.read_text(encoding="utf-8") == "client_secret: from-local\n"


def test_up_does_not_require_secrets_dir(tmp_path):
    repo = tmp_path / "daydaymoney-deploy"
    _seed_repo(repo)
    r = subprocess.run(
        ["bash", str(repo / "scripts" / "up.sh")],
        env=_clean_env(SYNC_ARTIFACTS="0", START="0"),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    assert (repo / "cutover.env").is_file()
    assert "http2client=0" in (repo / "cutover.env").read_text(encoding="utf-8")


def test_up_bootstraps_runall_via_gh(tmp_path):
    repo = tmp_path / "daydaymoney-deploy"
    _seed_repo(repo)
    _write(
        repo / "envs" / "current" / "releases.yaml",
        (
            "artifacts:\n"
            "  runAll:\n"
            "    sha: deadbeef\n"
            "    package: github://task2money/daydaymoney-deploy/runAll@deploy-20260831\n"
        ),
    )
    fake_gh = tmp_path / "fake-gh"
    fake_gh.write_text(_fake_gh_write_runall(), encoding="utf-8")
    fake_gh.chmod(0o755)

    r = subprocess.run(
        ["bash", str(repo / "scripts" / "up.sh")],
        env=_clean_env(SYNC_ARTIFACTS="0", START="0", GH=str(fake_gh)),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    runall = repo / "bin" / "runAll"
    assert runall.is_file()
    assert runall.read_text(encoding="utf-8") == "bootstrap\n"


def test_up_keeps_clone_root_conf_local(tmp_path):
    repo = tmp_path / "daydaymoney-deploy"
    _seed_repo(repo)
    _write(repo / "conf-local" / "auth" / "task-auth" / "config.yaml", "sms_key: already-here\n")
    r = subprocess.run(
        ["bash", str(repo / "scripts" / "up.sh")],
        env=_clean_env(SYNC_ARTIFACTS="0", START="0"),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    got = repo / "conf-local" / "auth" / "task-auth" / "config.yaml"
    assert got.read_text(encoding="utf-8") == "sms_key: already-here\n"


def test_write_cutover_env_writes_ssot_key_set(tmp_path):
    dep = tmp_path / "deploy"
    dep.mkdir()
    r = subprocess.run(["bash", str(WRITE_CUTOVER), str(dep)], capture_output=True, text=True)
    assert r.returncode == 0, r.stderr
    text = (dep / "cutover.env").read_text(encoding="utf-8")
    assert _cutover_keys(text) == EXPECTED_CUTOVER_KEYS
    assert "http2client=0" in text
    assert "export INFRA_HOST=${INFRA_HOST:-10.2.150.68}" in text


def test_up_cutover_env_matches_helper_key_set(tmp_path):
    repo = tmp_path / "daydaymoney-deploy"
    _seed_repo(repo)
    r = subprocess.run(
        ["bash", str(repo / "scripts" / "up.sh")],
        env=_clean_env(SYNC_ARTIFACTS="0", START="0"),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    text = (repo / "cutover.env").read_text(encoding="utf-8")
    assert _cutover_keys(text) == EXPECTED_CUTOVER_KEYS
    assert "http2client=0" in text


def test_up_applies_infra_host_env_from_conf_local(tmp_path):
    repo = tmp_path / "daydaymoney-deploy"
    _seed_repo(repo)
    _write(repo / "conf-local" / "infra-host.env", "INFRA_HOST=10.9.9.9\n")
    r = subprocess.run(
        ["bash", str(repo / "scripts" / "up.sh")],
        env=_clean_env(SYNC_ARTIFACTS="0", START="0"),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    cutover = (repo / "cutover.env").read_text(encoding="utf-8")
    assert "10.9.9.9" in cutover


def _fake_gh_write_runall() -> str:
    return (
        "#!/usr/bin/env bash\n"
        "set -euo pipefail\n"
        "dir=\"\"\n"
        "pattern=\"\"\n"
        "while [[ $# -gt 0 ]]; do\n"
        "  case \"$1\" in\n"
        "    --dir) dir=\"$2\"; shift 2 ;;\n"
        "    --pattern) pattern=\"$2\"; shift 2 ;;\n"
        "    --repo) shift 2 ;;\n"
        "    *) shift ;;\n"
        "  esac\n"
        "done\n"
        "if [[ \"$pattern\" != \"runAll\" ]]; then exit 1; fi\n"
        "printf 'bootstrap\\n' > \"$dir/runAll\"\n"
        "chmod +x \"$dir/runAll\"\n"
    )


def test_up_bootstraps_runall_disables_http2_on_gh(tmp_path):
    repo = tmp_path / "daydaymoney-deploy"
    _seed_repo(repo)
    _write(
        repo / "envs" / "current" / "releases.yaml",
        (
            "artifacts:\n"
            "  runAll:\n"
            "    sha: deadbeef\n"
            "    package: github://task2money/daydaymoney-deploy/runAll@deploy-20260831\n"
        ),
    )
    fake_gh = tmp_path / "fake-gh"
    fake_gh.write_text(
        "#!/usr/bin/env bash\n"
        "set -euo pipefail\n"
        "printf '%s\\n' \"${GODEBUG:-}\" > \"${GH_GODEBUG_LOG}\"\n"
        "if [[ \"${GODEBUG:-}\" != *http2client=0* ]]; then\n"
        "  echo 'stream error: stream ID 1; PROTOCOL_ERROR; received from peer' >&2\n"
        "  exit 1\n"
        "fi\n" + _fake_gh_write_runall().split("#!/usr/bin/env bash\n", 1)[1],
        encoding="utf-8",
    )
    fake_gh.chmod(0o755)
    golog = tmp_path / "godebug.log"
    r = subprocess.run(
        ["bash", str(repo / "scripts" / "up.sh")],
        env=_clean_env(
            SYNC_ARTIFACTS="0",
            START="0",
            GH=str(fake_gh),
            GH_GODEBUG_LOG=str(golog),
            GH_DOWNLOAD_RETRIES="1",
            GH_DOWNLOAD_SLEEP="0",
        ),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    assert "http2client=0" in golog.read_text(encoding="utf-8")
    assert (repo / "bin" / "runAll").read_text(encoding="utf-8") == "bootstrap\n"


def test_up_retries_gh_after_protocol_error(tmp_path):
    repo = tmp_path / "daydaymoney-deploy"
    _seed_repo(repo)
    _write(
        repo / "envs" / "current" / "releases.yaml",
        (
            "artifacts:\n"
            "  runAll:\n"
            "    sha: deadbeef\n"
            "    package: github://task2money/daydaymoney-deploy/runAll@deploy-20260831\n"
        ),
    )
    counter = tmp_path / "gh-count"
    fake_gh = tmp_path / "fake-gh"
    fake_gh.write_text(
        "#!/usr/bin/env bash\n"
        "set -euo pipefail\n"
        "n=0\n"
        "if [[ -f \"$GH_FAIL_COUNTER\" ]]; then n=\"$(cat \"$GH_FAIL_COUNTER\")\"; fi\n"
        "n=$((n + 1))\n"
        "printf '%s\\n' \"$n\" > \"$GH_FAIL_COUNTER\"\n"
        "if [[ \"$n\" -lt 2 ]]; then\n"
        "  echo 'stream error: stream ID 1; PROTOCOL_ERROR; received from peer' >&2\n"
        "  exit 1\n"
        "fi\n" + _fake_gh_write_runall().split("#!/usr/bin/env bash\n", 1)[1],
        encoding="utf-8",
    )
    fake_gh.chmod(0o755)
    r = subprocess.run(
        ["bash", str(repo / "scripts" / "up.sh")],
        env=_clean_env(
            SYNC_ARTIFACTS="0",
            START="0",
            GH=str(fake_gh),
            GH_FAIL_COUNTER=str(counter),
            GH_DOWNLOAD_RETRIES="3",
            GH_DOWNLOAD_SLEEP="0",
        ),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    assert counter.read_text(encoding="utf-8").strip() == "2"
    assert (repo / "bin" / "runAll").read_text(encoding="utf-8") == "bootstrap\n"


def test_up_falls_back_to_curl_http11_after_gh_protocol_error(tmp_path):
    repo = tmp_path / "daydaymoney-deploy"
    _seed_repo(repo)
    _write(
        repo / "envs" / "current" / "releases.yaml",
        (
            "artifacts:\n"
            "  runAll:\n"
            "    sha: deadbeef\n"
            "    package: github://task2money/daydaymoney-deploy/runAll@deploy-20260831\n"
        ),
    )
    fake_gh = tmp_path / "fake-gh"
    fake_gh.write_text(
        "#!/usr/bin/env bash\n"
        "echo 'stream error: stream ID 1; PROTOCOL_ERROR; received from peer' >&2\n"
        "exit 1\n",
        encoding="utf-8",
    )
    fake_gh.chmod(0o755)
    fake_curl = tmp_path / "fake-curl"
    fake_curl.write_text(
        "#!/usr/bin/env bash\n"
        "set -euo pipefail\n"
        "out=\"\"\n"
        "url=\"\"\n"
        "http11=0\n"
        "while [[ $# -gt 0 ]]; do\n"
        "  case \"$1\" in\n"
        "    --http1.1) http11=1; shift ;;\n"
        "    -o) out=\"$2\"; shift 2 ;;\n"
        "    -H) shift 2 ;;\n"
        "    -fsSL|-f|-s|-S|-L) shift ;;\n"
        "    http*|https*) url=\"$1\"; shift ;;\n"
        "    *) shift ;;\n"
        "  esac\n"
        "done\n"
        "if [[ \"$http11\" != 1 ]]; then echo 'need --http1.1' >&2; exit 1; fi\n"
        "if [[ -n \"$out\" ]]; then\n"
        "  printf 'from-curl\\n' > \"$out\"\n"
        "  chmod +x \"$out\"\n"
        "  exit 0\n"
        "fi\n"
        "printf '%s\\n' '{\"assets\":[{\"name\":\"runAll\",\"url\":\"https://example.invalid/runAll\"}]}'\n",
        encoding="utf-8",
    )
    fake_curl.chmod(0o755)
    r = subprocess.run(
        ["bash", str(repo / "scripts" / "up.sh")],
        env=_clean_env(
            SYNC_ARTIFACTS="0",
            START="0",
            GH=str(fake_gh),
            CURL=str(fake_curl),
            GITHUB_TOKEN="test-token",
            GH_DOWNLOAD_RETRIES="1",
            GH_DOWNLOAD_SLEEP="0",
        ),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    assert (repo / "bin" / "runAll").read_text(encoding="utf-8") == "from-curl\n"


def test_up_fails_when_sync_on_and_runall_missing(tmp_path):
    repo = tmp_path / "daydaymoney-deploy"
    _seed_repo(repo)
    fake_gh = tmp_path / "fake-gh"
    fake_gh.write_text("#!/usr/bin/env bash\nexit 1\n", encoding="utf-8")
    fake_gh.chmod(0o755)
    r = subprocess.run(
        ["bash", str(repo / "scripts" / "up.sh")],
        env=_clean_env(SYNC_ARTIFACTS="1", START="0", GH=str(fake_gh)),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode != 0
    assert "bin/runAll" in (r.stderr + r.stdout)


def test_up_overlays_taskfe_public_after_layout(tmp_path):
    repo = tmp_path / "daydaymoney-deploy"
    _seed_repo(repo)
    public = tmp_path / "fe-public"
    _write(public / "index.html", "<html>ok</html>\n")
    r = subprocess.run(
        ["bash", str(repo / "scripts" / "up.sh")],
        env=_clean_env(SYNC_ARTIFACTS="0", START="0", TASKFE_PUBLIC=str(public)),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    got = repo / "taskFE" / "app" / "public" / "index.html"
    assert got.read_text(encoding="utf-8") == "<html>ok</html>\n"


def test_up_overlays_provider_frontend_after_layout(tmp_path):
    repo = tmp_path / "daydaymoney-deploy"
    _seed_repo(repo)
    dist = tmp_path / "provider-dist"
    _write(dist / "index.html", "<html>vendor-spa</html>\n")
    r = subprocess.run(
        ["bash", str(repo / "scripts" / "up.sh")],
        env=_clean_env(
            SYNC_ARTIFACTS="0", START="0", TASKAIPROVIDER_FRONTEND=str(dist)
        ),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout
    got = repo / "taskAiProvider" / "frontend" / "dist" / "index.html"
    assert got.read_text(encoding="utf-8") == "<html>vendor-spa</html>\n"

