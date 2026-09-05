#!/usr/bin/env python3
"""Unit tests for GitLab workhorse git-upload-pack traffic shipper parser."""
from __future__ import annotations

import json
import sys
from pathlib import Path

SCRIPT = Path(__file__).resolve().parent / "ship_gitlab_git_traffic.py"
sys.path.insert(0, str(SCRIPT.parent))
import ship_gitlab_git_traffic as ship  # noqa: E402


def test_project_path_from_upload_pack_uri():
    got = ship.project_path_from_uri("/example-user/somanyad.git/git-upload-pack")
    assert got == "example-user/somanyad", got


def test_project_path_from_info_refs():
    got = ship.project_path_from_uri(
        "/tenant-877397588196749312/app.git/info/refs?service=git-upload-pack"
    )
    assert got == "tenant-877397588196749312/app", got


def test_project_path_skips_receive_pack():
    assert ship.project_path_from_uri("/g/r.git/git-receive-pack") is None


def test_parse_access_line_meters_public_clone():
    line = json.dumps({
        "msg": "access",
        "method": "POST",
        "status": 200,
        "uri": "/example-user/somanyad.git/git-upload-pack",
        "written_bytes": 11272192,
        "correlation_id": "01CLONECORR01",
        "remote_ip": "203.0.113.9",
        "host": "gitlab-tencent-sh-1.daydaymoney.com",
        "user_agent": "git/2.43.0",
    })
    rec = ship.parse_workhorse_line(
        line,
        intranet_hosts=set(),
        public_host="gitlab-tencent-sh-1.daydaymoney.com",
    )
    assert rec is not None
    assert rec["project_path"] == "example-user/somanyad"
    assert rec["bytes"] == 11272192
    assert rec["idempotency_key"] == "wh:01CLONECORR01"
    assert rec["from_ci"] is False
    assert rec["is_intranet"] is False
    assert rec["gitlab_username"] == "example-user"


def test_parse_skips_non_git_access():
    line = json.dumps({
        "msg": "access",
        "uri": "/users/sign_in",
        "written_bytes": 2026,
        "correlation_id": "x",
        "status": 200,
        "remote_ip": "1.1.1.1",
        "host": "gitlab.daydaymoney.com",
    })
    assert ship.parse_workhorse_line(line, set(), "gitlab.daydaymoney.com") is None


def test_parse_skips_docker_nat_not_intranet():
    line = json.dumps({
        "msg": "access",
        "uri": "/g/r.git/git-upload-pack",
        "written_bytes": 1000,
        "correlation_id": "nat1",
        "status": 200,
        "remote_ip": "172.21.0.1",
        "host": "gitlab-tencent-sh-1.daydaymoney.com",
        "user_agent": "git/2.43.0",
    })
    rec = ship.parse_workhorse_line(
        line, set(), "gitlab-tencent-sh-1.daydaymoney.com"
    )
    assert rec is not None
    assert rec["is_intranet"] is False


def test_parse_skips_vpc_intranet():
    line = json.dumps({
        "msg": "access",
        "uri": "/g/r.git/git-upload-pack",
        "written_bytes": 1000,
        "correlation_id": "vpc1",
        "status": 200,
        "remote_ip": "10.2.150.80",
        "host": "gitlab-tencent-sh-1.daydaymoney.com",
        "user_agent": "git/2.43.0",
    })
    rec = ship.parse_workhorse_line(
        line, set(), "gitlab-tencent-sh-1.daydaymoney.com"
    )
    assert rec is None  # intranet: do not ship


def test_parse_skips_gitlab_ci_user_agent():
    line = json.dumps({
        "msg": "access",
        "uri": "/g/r.git/git-upload-pack",
        "written_bytes": 5000,
        "correlation_id": "ci1",
        "status": 200,
        "remote_ip": "203.0.113.2",
        "host": "gitlab.daydaymoney.com",
        "user_agent": "gitlab-runner 17.0.0",
    })
    assert ship.parse_workhorse_line(line, set(), "gitlab.daydaymoney.com") is None


def test_parse_git_traffic_without_uri_skipped():
    line = json.dumps({
        "msg": "git_traffic",
        "service": "git-upload-pack",
        "written_bytes": 0,
        "project_id": 2,
    })
    assert ship.parse_workhorse_line(line, set(), "gitlab.daydaymoney.com") is None


def test_parse_svlogd_prefix():
    payload = {
        "msg": "access",
        "uri": "/tenant-1/p.git/git-upload-pack",
        "written_bytes": 64,
        "correlation_id": "pref",
        "status": 200,
        "remote_ip": "8.8.8.8",
        "host": "gitlab.daydaymoney.com",
        "user_agent": "git/2.40",
    }
    line = "@400000006a8b0bb43ab45fac " + json.dumps(payload)
    rec = ship.parse_workhorse_line(line, set(), "gitlab.daydaymoney.com")
    assert rec is not None
    assert rec["project_path"] == "tenant-1/p"
    assert rec["bytes"] == 64


def test_should_advance_offset_on_client_error():
    assert ship.should_advance_offset(200) is True
    assert ship.should_advance_offset(400) is True
    assert ship.should_advance_offset(402) is True
    assert ship.should_advance_offset(502) is False
    assert ship.should_advance_offset(0) is False


def test_parse_gitlab_shell_handbook_written_bytes():
    line = json.dumps({
        "command": "git-upload-pack",
        "written_bytes": 4096,
        "project": "example-user/somanyad",
        "correlation_id": "01SSHCORR01",
        "remote_ip": "203.0.113.9",
        "username": "example-user",
        "time": "2026-08-28T00:00:00Z",
    })
    rec = ship.parse_gitlab_shell_line(
        line, intranet_hosts=set(), public_host="gitlab.daydaymoney.com"
    )
    assert rec is not None
    assert rec["project_path"] == "example-user/somanyad"
    assert rec["bytes"] == 4096
    assert rec["idempotency_key"] == "ssh:01SSHCORR01"
    assert not rec["idempotency_key"].startswith("wh:")
    assert rec["gitlab_username"] == "example-user"


def test_parse_gitlab_shell_gl_project_path():
    line = json.dumps({
        "command": "git-upload-pack",
        "gl_project_path": "tenant-877397588196749312/app.git",
        "written_bytes": 8192,
        "correlation_id": "ssh2",
        "remote_ip": "198.51.100.4",
    })
    rec = ship.parse_gitlab_shell_line(line, set(), "gitlab.daydaymoney.com")
    assert rec is not None
    assert rec["project_path"] == "tenant-877397588196749312/app"
    assert rec["idempotency_key"] == "ssh:ssh2"


def test_parse_gitlab_shell_skips_without_written_bytes():
    line = json.dumps({
        "command": "git-upload-pack",
        "gl_project_path": "root/simple-ci",
        "msg": "executing git command",
        "correlation_id": "old1",
        "remote_ip": "203.0.113.9",
    })
    assert ship.parse_gitlab_shell_line(line, set(), "gitlab.daydaymoney.com") is None


def test_parse_gitlab_shell_skips_receive_pack():
    line = json.dumps({
        "command": "git-receive-pack",
        "project": "g/r",
        "written_bytes": 999,
        "correlation_id": "push1",
        "remote_ip": "203.0.113.9",
    })
    assert ship.parse_gitlab_shell_line(line, set(), "gitlab.daydaymoney.com") is None


def test_parse_gitlab_shell_skips_vpc_intranet():
    line = json.dumps({
        "command": "git-upload-pack",
        "project": "g/r",
        "written_bytes": 1000,
        "correlation_id": "vpcssh",
        "remote_ip": "10.2.150.80",
    })
    assert ship.parse_gitlab_shell_line(line, set(), "gitlab.daydaymoney.com") is None


def test_parse_gitlab_shell_skips_ci_username():
    line = json.dumps({
        "command": "git-upload-pack",
        "project": "g/r",
        "written_bytes": 5000,
        "correlation_id": "cissh",
        "remote_ip": "203.0.113.2",
        "username": "gitlab-ci",
    })
    assert ship.parse_gitlab_shell_line(line, set(), "gitlab.daydaymoney.com") is None


def test_parse_gitlab_shell_docker_nat_not_intranet():
    line = json.dumps({
        "command": "git-upload-pack",
        "project": "g/r",
        "written_bytes": 1000,
        "correlation_id": "natssh",
        "remote_ip": "172.21.0.1",
    })
    rec = ship.parse_gitlab_shell_line(line, set(), "gitlab.daydaymoney.com")
    assert rec is not None
    assert rec["bytes"] == 1000


if __name__ == "__main__":
    tests = [v for k, v in globals().items() if k.startswith("test_")]
    failed = 0
    for fn in tests:
        try:
            fn()
            print("ok", fn.__name__)
        except Exception as e:
            failed += 1
            print("FAIL", fn.__name__, e)
    if failed:
        sys.exit(1)
