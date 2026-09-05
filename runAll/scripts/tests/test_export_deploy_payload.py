"""export-deploy-payload.sh copies recipes into daydaymoney-deploy without source or data."""

from __future__ import annotations

import os
import subprocess
from pathlib import Path

SCRIPTS = Path(__file__).resolve().parents[1]
EXPORT = SCRIPTS / "export-deploy-payload.sh"


def _write(p: Path, text: str = "ok\n") -> None:
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(text, encoding="utf-8")
    if p.suffix == ".sh":
        p.chmod(0o755)


def _fake_meta(root: Path) -> None:
    _write(root / "dockerInfra" / "mysql" / "run.sh")
    _write(root / "dockerInfra" / "mysql" / "data" / "ibdata1", "LIVE")
    _write(root / "gitService" / "run.sh")
    _write(root / "gitService" / "gitlab-ce" / "huge.go", "package main\n")
    _write(root / "gitService" / "gitlab_home" / "secret", "nope")
    _write(root / "AiMonitor" / "run.sh")
    _write(root / "taskGateway" / "run.sh")
    _write(root / "taskGateway" / "certs" / "dev-gateway.pem", "PEM")
    _write(root / "taskSSE" / "run.sh")
    _write(root / "taskEvents" / "run.sh")
    _write(root / "taskFE" / "docker-compose.yml", "services: {}\n")
    _write(root / "taskFE" / "nginx" / "nginx.conf")
    _write(root / "taskFE" / "app" / "scripts" / "runall-lifecycle.sh")
    _write(root / "taskFE" / "app" / "src" / "main.js", "secret src")
    _write(root / "dataMigrate" / "taskAuth" / "001.sql", "SELECT 1;\n")
    _write(root / "dataMigrate" / "taskAuth" / "skip.go", "package x\n")
    _write(root / "db" / "registry.yaml", "version: '2'\n")
    _write(root / "db" / "task-auth" / "migrate.sh")
    _write(root / "db" / "task-auth" / "oidc_signing_key.pem", "PEM")
    _write(root / "db" / "load" / "registry.go", "package load\n")
    _write(root / "runAll" / "scripts" / "conf-read.py", "print(1)\n")
    _write(
        root / "runAll" / "run.sh",
        "#!/bin/bash\nsetsid -f nohup ./bin/runAll --config ../conf/runAll.yaml\n",
    )
    _write(root / "runAll" / "src" / "main.go", "package main\n")
    _write(root / "trae-agent" / "onlineServiceJS" / "run.sh")
    _write(root / "trae-agent" / "onlineServiceJS" / "Dockerfile", "FROM scratch\n")
    _write(root / "trae-agent" / "onlineServiceJS" / "buildDocker.sh")
    _write(root / "trae-agent" / "onlineServiceJS" / "src" / "server.mjs", "export {}\n")
    _write(
        root / "trae-agent" / "onlineServiceJS" / "docker" / "code-server" / "code-server-4.96.4-linux-amd64.tar.gz",
        "too-big-for-github\n",
    )
    _write(root / "trae-agent" / "pyproject.toml", "[project]\nname='x'\n")
    _write(root / "trae-agent" / "trae_agent" / "__init__.py", "")


def test_export_omits_source_data_and_secrets(tmp_path):
    meta = tmp_path / "meta"
    dest = tmp_path / "daydaymoney-deploy"
    dest.mkdir()
    _fake_meta(meta)

    r = subprocess.run(
        ["bash", str(EXPORT), str(dest)],
        cwd=str(meta),
        env={**os.environ, "META_ROOT": str(meta)},
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode == 0, r.stderr + r.stdout

    env = dest / "envs" / "current"
    assert (env / "dockerInfra" / "mysql" / "run.sh").is_file()
    assert not (env / "dockerInfra" / "mysql" / "data").exists()
    assert (env / "gitService" / "run.sh").is_file()
    assert not (env / "gitService" / "gitlab-ce").exists()
    assert not (env / "gitService" / "gitlab_home").exists()
    assert not (env / "taskGateway" / "certs" / "dev-gateway.pem").exists()
    assert not (env / "taskFE" / "app" / "src").exists()
    assert (env / "taskFE" / "nginx" / "nginx.conf").is_file()
    assert (env / "dataMigrate" / "taskAuth" / "001.sql").is_file()
    assert not list(env.rglob("*.go"))
    assert not (env / "db" / "load").exists()
    assert (env / "db" / "registry.yaml").is_file()
    assert not (env / "db" / "task-auth" / "oidc_signing_key.pem").exists()
    assert (dest / "scripts" / "layout.sh").is_file()
    assert (dest / "scripts" / "up.sh").is_file()
    assert os.access(dest / "scripts" / "up.sh", os.X_OK)
    up_body = (dest / "scripts" / "up.sh").read_text(encoding="utf-8")
    assert "write-cutover-env.sh" in up_body
    assert 'cat > "$DEPLOY_ROOT/cutover.env"' not in up_body
    helper = dest / "scripts" / "write-cutover-env.sh"
    assert helper.is_file(), "up.sh sources write-cutover-env.sh from the same scripts/ dir"
    assert os.access(helper, os.X_OK)
    assert "export SOURCE_ROOT" in helper.read_text(encoding="utf-8")
    runall_run = env / "runAll" / "run.sh"
    assert runall_run.is_file()
    assert os.access(runall_run, os.X_OK)
    assert "setsid -f nohup" in runall_run.read_text(encoding="utf-8")
    assert (dest / "scripts" / "install-local-artifacts.sh").is_file()
    assert os.access(dest / "scripts" / "install-local-artifacts.sh", os.X_OK)
    readme = dest / "secrets.example" / "README.md"
    assert readme.is_file()
    assert not (dest / "secrets.example" / "HOST_SECRETS.md").exists()
    body = readme.read_text(encoding="utf-8")
    assert "conf-local" in body
    assert "HOST_SECRETS.md" not in body
    assert (env / "trae-agent" / "onlineServiceJS" / "run.sh").is_file()
    assert os.access(env / "trae-agent" / "onlineServiceJS" / "run.sh", os.X_OK)
    assert (env / "trae-agent" / "onlineServiceJS" / "Dockerfile").is_file()
    assert (env / "trae-agent" / "onlineServiceJS" / "buildDocker.sh").is_file()
    assert (env / "trae-agent" / "onlineServiceJS" / "src" / "server.mjs").is_file()
    assert (env / "trae-agent" / "pyproject.toml").is_file()
    assert (env / "trae-agent" / "trae_agent" / "__init__.py").is_file()
    gi = (dest / ".gitignore").read_text(encoding="utf-8")
    assert "/conf-local/" in gi
    assert "/deploy-binaries/" in gi
    assert "docker/code-server/*.tar.gz" in gi
    assert "/taskAiProvider/frontend/dist/" in gi
    assert "/taskFE/app/public/" in gi
    assert not list((env / "trae-agent" / "onlineServiceJS" / "docker" / "code-server").glob("*.tar.gz"))
    top = (dest / "README.md").read_text(encoding="utf-8")
    assert "rsync -a" in top
    assert "conf-local/" in top
    assert "artifacts/" in top
    assert "HOST_SECRETS" not in top
    assert not (dest / "secrets.example" / "HOST_SECRETS.md").exists()


def test_export_requires_dest(tmp_path):
    r = subprocess.run(
        ["bash", str(EXPORT)],
        cwd=str(tmp_path),
        capture_output=True,
        text=True,
        check=False,
    )
    assert r.returncode != 0
    assert "usage" in r.stderr.lower() or "dest" in r.stderr.lower()
