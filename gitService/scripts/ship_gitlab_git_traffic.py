#!/usr/bin/env python3
"""Ship GitLab git-upload-pack written_bytes to taskBill (ADR-0042).

Follows gitlab-workhorse/current (HTTP, idempotency wh:) and
gitlab-shell/gitlab-shell.log (SSH, idempotency ssh:). Does not inherit
env HTTP(S)_PROXY. Business HTTP services must not poll logs (ADR-0011);
this is an infra sidecar.
"""
from __future__ import annotations

import ipaddress
import json
import os
import sys
import threading
import time
import urllib.error
import urllib.request
from collections.abc import Callable
from typing import Any, Optional
from urllib.parse import unquote

DOWNLOAD_MARKERS = ("git-upload-pack", "git-upload-archive")
FALSE_INTRANET_NETS = (
    ipaddress.ip_network("127.0.0.0/8"),
    ipaddress.ip_network("::1/128"),
    ipaddress.ip_network("169.254.0.0/16"),
    ipaddress.ip_network("172.16.0.0/12"),
)
VPC_NETS = (
    ipaddress.ip_network("10.0.0.0/8"),
    ipaddress.ip_network("192.168.0.0/16"),
)


def _unset_env_proxy() -> None:
    for key in (
        "http_proxy",
        "https_proxy",
        "HTTP_PROXY",
        "HTTPS_PROXY",
        "ALL_PROXY",
        "all_proxy",
    ):
        os.environ.pop(key, None)


def _parse_ip(raw: str):
    s = (raw or "").strip()
    if not s:
        return None
    if "%" in s:
        s = s.split("%", 1)[0]
    if s.startswith("[") and "]" in s:
        s = s[1 : s.index("]")]
    if ":" in s and s.count(":") == 1 and not s.startswith(":"):
        host, _, port = s.rpartition(":")
        if port.isdigit():
            s = host
    try:
        return ipaddress.ip_address(s)
    except ValueError:
        return None


def _ip_in_nets(addr, nets: tuple) -> bool:
    return any(addr in net for net in nets)


def false_intranet_ip(raw: str) -> bool:
    addr = _parse_ip(raw)
    if addr is None:
        return False
    return _ip_in_nets(addr, FALSE_INTRANET_NETS)


def vpc_intranet_ip(raw: str) -> bool:
    addr = _parse_ip(raw)
    if addr is None:
        return False
    return _ip_in_nets(addr, VPC_NETS)


def normalize_host(raw: str) -> str:
    h = (raw or "").strip().lower()
    if not h:
        return ""
    if h.startswith("[") and "]" in h:
        h = h[1 : h.index("]")]
    if ":" in h and h.count(":") == 1:
        host, _, port = h.rpartition(":")
        if port.isdigit():
            h = host
    return h


def is_intranet_request(
    remote_ip: str, host: str, intranet_hosts: set[str], public_host: str
) -> bool:
    host_n = normalize_host(host)
    pub = normalize_host(public_host)
    if host_n and host_n in intranet_hosts and host_n != pub:
        return True
    if host_n and vpc_intranet_ip(host_n):
        return True
    if false_intranet_ip(remote_ip):
        return False
    return vpc_intranet_ip(remote_ip)


def is_ci_user_agent(user_agent: str) -> bool:
    ua = (user_agent or "").lower()
    return "gitlab-runner" in ua or "gitlab-ci" in ua or "gitlab-job" in ua


def project_path_from_uri(uri: str) -> Optional[str]:
    if not uri:
        return None
    raw = unquote(uri.split("?", 1)[0]).strip()
    path = raw.strip("/")
    lower = path.lower()
    suffix = None
    for marker in DOWNLOAD_MARKERS:
        token = "/" + marker
        if lower.endswith(token):
            suffix = path[-(len(token)) :]
            break
    if suffix is None:
        if lower.endswith("/info/refs") and any(
            m in (uri or "").lower() for m in DOWNLOAD_MARKERS
        ):
            suffix = path[-len("/info/refs") :]
        else:
            return None
    path = path[: -len(suffix)].rstrip("/")
    if path.lower().endswith(".git"):
        path = path[:-4]
    path = path.strip("/")
    return path or None


def json_from_log_line(line: str) -> Optional[dict[str, Any]]:
    s = (line or "").strip()
    if not s:
        return None
    if s[0] != "{":
        brace = s.find("{")
        if brace < 0:
            return None
        s = s[brace:]
    try:
        obj = json.loads(s)
    except json.JSONDecodeError:
        return None
    if not isinstance(obj, dict):
        return None
    return obj


def parse_workhorse_line(
    line: str, intranet_hosts: set[str], public_host: str
) -> Optional[dict[str, Any]]:
    rec = json_from_log_line(line)
    if rec is None:
        return None
    msg = str(rec.get("msg") or "")
    uri = str(rec.get("uri") or "")
    service = str(rec.get("service") or "")
    if msg == "git_traffic":
        if service not in DOWNLOAD_MARKERS:
            return None
        # git_traffic has no project path; skip unless uri is also present.
        if not uri:
            return None
    elif msg != "access":
        return None
    path = project_path_from_uri(uri)
    if not path:
        return None
    try:
        status = int(rec.get("status") or 0)
    except (TypeError, ValueError):
        status = 0
    if msg == "access" and status not in (200, 201):
        return None
    try:
        written = int(rec.get("written_bytes") or 0)
    except (TypeError, ValueError):
        written = 0
    if written <= 0:
        return None
    ua = str(rec.get("user_agent") or "")
    if is_ci_user_agent(ua):
        return None
    remote_ip = str(rec.get("remote_ip") or rec.get("remote_addr") or "")
    host = str(rec.get("host") or "")
    if is_intranet_request(remote_ip, host, intranet_hosts, public_host):
        return None
    corr = str(rec.get("correlation_id") or rec.get("gitaly_correlation_id") or "").strip()
    if not corr:
        corr = "anon:%s:%s:%s" % (path, rec.get("time") or "", written)
    ns = path.split("/", 1)[0]
    return {
        "project_path": path,
        "bytes": written,
        "idempotency_key": "wh:" + corr,
        "from_ci": False,
        "is_intranet": False,
        "gitlab_username": ns,
    }


def parse_gitlab_shell_line(
    line: str, intranet_hosts: set[str], public_host: str
) -> dict[str, Any] | None:
    """SSH git-upload-pack meter from gitlab-shell JSON (handbook written_bytes)."""
    rec = json_from_log_line(line)
    if rec is None:
        return None
    cmd = str(rec.get("command") or "").lower().strip()
    if "receive-pack" in cmd:
        return None
    if cmd and "upload-pack" not in cmd and cmd not in DOWNLOAD_MARKERS:
        return None
    path = str(rec.get("project") or rec.get("gl_project_path") or "").strip()
    path = path.strip("/")
    if path.lower().endswith(".git"):
        path = path[:-4]
    path = path.strip("/")
    if not path:
        return None
    try:
        written = int(rec.get("written_bytes") or 0)
    except (TypeError, ValueError):
        written = 0
    if written <= 0:
        return None
    ua = str(rec.get("user_agent") or "")
    uname = str(rec.get("username") or rec.get("gl_username") or "").lower()
    if is_ci_user_agent(ua) or uname in ("gitlab-ci", "gitlab-runner"):
        return None
    remote_ip = str(rec.get("remote_ip") or rec.get("remote_addr") or "")
    host = str(rec.get("host") or "")
    if is_intranet_request(remote_ip, host, intranet_hosts, public_host):
        return None
    corr = str(rec.get("correlation_id") or rec.get("correlationId") or "").strip()
    if not corr:
        corr = f"anon:{path}:{rec.get('time') or ''}:{written}"
    ns = path.split("/", 1)[0]
    return {
        "project_path": path,
        "bytes": written,
        "idempotency_key": "ssh:" + corr,
        "from_ci": False,
        "is_intranet": False,
        "gitlab_username": ns,
    }


def should_advance_offset(http_status: int) -> bool:
    if http_status == 0:
        return False
    if 200 <= http_status < 300:
        return True
    if 400 <= http_status < 500:
        return True
    return False


def _direct_opener() -> urllib.request.OpenerDirector:
    return urllib.request.build_opener(urllib.request.ProxyHandler({}))


def post_meter(base: str, secret: str, region: str, rec: dict[str, Any]) -> int:
    url = base.rstrip("/") + "/api/internal/taskbill/charge-gitlab-traffic/"
    body = {
        "project_path": rec["project_path"],
        "bytes": rec["bytes"],
        "region": region,
        "idempotency_key": rec["idempotency_key"],
        "from_ci": rec.get("from_ci", False),
        "is_intranet": rec.get("is_intranet", False),
        "gitlab_username": rec.get("gitlab_username") or "",
    }
    data = json.dumps(body).encode("utf-8")
    req = urllib.request.Request(url, data=data, method="POST")
    req.add_header("Content-Type", "application/json")
    if secret:
        req.add_header("X-TaskBill-Internal-Secret", secret)
    opener = _direct_opener()
    try:
        with opener.open(req, timeout=10) as resp:
            return int(getattr(resp, "status", 200) or 200)
    except urllib.error.HTTPError as e:
        return int(e.code or 0)
    except (urllib.error.URLError, TimeoutError, OSError):
        return 0


def load_state(path: str) -> dict[str, Any]:
    try:
        with open(path, encoding="utf-8") as f:
            obj = json.load(f)
        if isinstance(obj, dict):
            return obj
    except (OSError, json.JSONDecodeError):
        pass
    return {}


def save_state(path: str, inode: int, offset: int) -> None:
    tmp = path + ".tmp"
    payload = json.dumps({"inode": inode, "offset": offset})
    with open(tmp, "w", encoding="utf-8") as f:
        f.write(payload)
    os.replace(tmp, path)


def follow_and_ship(
    log_path: str,
    state_path: str,
    base: str,
    secret: str,
    region: str,
    intranet_hosts: set[str],
    public_host: str,
    parse_line: Callable[
        [str, set[str], str], dict[str, Any] | None
    ] = parse_workhorse_line,
) -> None:
    state = load_state(state_path)
    first_seen = "inode" not in state
    while True:
        try:
            st = os.stat(log_path)
        except FileNotFoundError:
            time.sleep(1)
            continue
        inode = int(st.st_ino)
        with open(log_path, "r", encoding="utf-8", errors="replace") as fh:
            if state.get("inode") == inode:
                fh.seek(int(state.get("offset") or 0))
            elif first_seen:
                fh.seek(0, os.SEEK_END)
                first_seen = False
                save_state(state_path, inode, fh.tell())
                state = {"inode": inode, "offset": fh.tell()}
            else:
                fh.seek(0)
            while True:
                line = fh.readline()
                if line == "":
                    try:
                        st2 = os.stat(log_path)
                    except FileNotFoundError:
                        break
                    if int(st2.st_ino) != inode:
                        state = {"inode": inode, "offset": fh.tell()}
                        break
                    time.sleep(0.25)
                    continue
                rec = parse_line(line, intranet_hosts, public_host)
                if rec is None:
                    save_state(state_path, inode, fh.tell())
                    state = {"inode": inode, "offset": fh.tell()}
                    continue
                while True:
                    status = post_meter(base, secret, region, rec)
                    if should_advance_offset(status):
                        break
                    print(
                        json.dumps(
                            {
                                "msg": "gitlab_traffic_ship_retry",
                                "status": status,
                                "project_path": rec["project_path"],
                            }
                        ),
                        file=sys.stderr,
                        flush=True,
                    )
                    time.sleep(2)
                save_state(state_path, inode, fh.tell())
                state = {"inode": inode, "offset": fh.tell()}
                print(
                    json.dumps(
                        {
                            "msg": "gitlab_traffic_shipped",
                            "project_path": rec["project_path"],
                            "bytes": rec["bytes"],
                            "region": region,
                            "http_status": status,
                        }
                    ),
                    flush=True,
                )


def _csv_hosts(raw: str) -> set[str]:
    out = set()
    for part in (raw or "").split(","):
        h = normalize_host(part)
        if h:
            out.add(h)
    return out


def main() -> int:
    _unset_env_proxy()
    log_path = os.environ.get(
        "GITLAB_WORKHORSE_LOG", "/var/log/gitlab/gitlab-workhorse/current"
    )
    shell_log = os.environ.get(
        "GITLAB_SHELL_LOG", "/var/log/gitlab/gitlab-shell/gitlab-shell.log"
    )
    state_path = os.environ.get(
        "SHIPPER_STATE_FILE", "/state/gitlab_traffic_shipper.offset.json"
    )
    shell_state = os.environ.get(
        "SHIPPER_SHELL_STATE_FILE",
        "/state/gitlab_shell_traffic_shipper.offset.json",
    )
    base = (os.environ.get("TRAE_TASKBILL_BASE") or "").strip()
    secret = os.environ.get("TRAE_TASKBILL_INTERNAL_SECRET") or ""
    region = (os.environ.get("TRAE_GITLAB_REGION") or "").strip()
    public_host = os.environ.get("TRAE_GITLAB_PUBLIC_HOST") or ""
    intranet_hosts = _csv_hosts(os.environ.get("TRAE_GITLAB_INTRANET_HOSTS") or "")
    if not base:
        print("TRAE_TASKBILL_BASE required", file=sys.stderr)
        return 2
    if not region:
        print("TRAE_GITLAB_REGION required", file=sys.stderr)
        return 2
    print(
        json.dumps(
            {
                "msg": "gitlab_traffic_shipper_start",
                "log": log_path,
                "shell_log": shell_log,
                "region": region,
                "taskbill": base,
            }
        ),
        flush=True,
    )
    kwargs = {
        "base": base,
        "secret": secret,
        "region": region,
        "intranet_hosts": intranet_hosts,
        "public_host": public_host,
    }
    workers = [
        threading.Thread(
            target=follow_and_ship,
            kwargs={
                **kwargs,
                "log_path": log_path,
                "state_path": state_path,
                "parse_line": parse_workhorse_line,
            },
            daemon=True,
            name="workhorse-traffic",
        ),
        threading.Thread(
            target=follow_and_ship,
            kwargs={
                **kwargs,
                "log_path": shell_log,
                "state_path": shell_state,
                "parse_line": parse_gitlab_shell_line,
            },
            daemon=True,
            name="shell-traffic",
        ),
    ]
    for t in workers:
        t.start()
    for t in workers:
        t.join()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
