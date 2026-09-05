#!/usr/bin/env python3
"""Predicates for ensure-edge-tunnels.sh: real autossh vs half-open probe.

The SH reverse-forward can keep sshd LISTEN on 127.0.0.1:<port> after the
channel dies. nginx then hangs until proxy timeout (provider.daydaymoney.com
「无法打开」). pgrep must not treat agent bash wrappers as a live tunnel.
"""

from __future__ import annotations

import argparse
import re
import sys


def autossh_pgrep_regex(bind: str, port: str) -> str:
    """Regex matched against /proc cmdline. bind is '' or '172.17.0.1:' / '0.0.0.0:'."""
    return rf"/usr/lib/autossh/autossh .*-R {re.escape(bind)}{re.escape(port)}:127\.0\.0\.1:{re.escape(port)}"


def cmdline_is_real_autossh(cmdline: str, bind: str, port: str) -> bool:
    return re.search(autossh_pgrep_regex(bind, port), cmdline) is not None


def remote_probe_ok(*, curl_exit: int, http_code: str) -> bool:
    """True if SH 127.0.0.1:<port> accepted a curl (tunnel channel alive).

    curl 22 = HTTP >=400 (service up). curl 52 = empty reply (GitLab :8012).
    curl 7 = refused, 28 = timeout (half-open / dead).
    """
    if http_code.isdigit() and 100 <= int(http_code) <= 599:
        return True
    return curl_exit in (22, 52)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser()
    sub = parser.add_subparsers(dest="cmd", required=True)
    pgrep = sub.add_parser("pgrep-regex")
    pgrep.add_argument("bind")
    pgrep.add_argument("port")
    match = sub.add_parser("match-cmdline")
    match.add_argument("bind")
    match.add_argument("port")
    match.add_argument("cmdline")
    probe = sub.add_parser("probe-ok")
    probe.add_argument("curl_exit", type=int)
    probe.add_argument("http_code")
    args = parser.parse_args(argv)
    if args.cmd == "pgrep-regex":
        bind = "" if args.bind in ("-", "") else args.bind
        print(autossh_pgrep_regex(bind, args.port))
        return 0
    if args.cmd == "match-cmdline":
        bind = "" if args.bind in ("-", "") else args.bind
        return 0 if cmdline_is_real_autossh(args.cmdline, bind, args.port) else 1
    bind_ok = remote_probe_ok(curl_exit=args.curl_exit, http_code=args.http_code)
    return 0 if bind_ok else 1


if __name__ == "__main__":
    sys.exit(main())
