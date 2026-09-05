#!/usr/bin/env python3
"""回归：APISIX 须在 Docker DNAT 后从 X-Forwarded-For 还原真实客户端 IP。

缺陷：APISIX 以 ports: 18081:9080 发布时，容器 $remote_addr 是
taskgateway_default 网关 172.25.0.1；taskFE :4000 同理是 172.26.0.1。
若只信 RemoteAddr / XFF 首跳，登录历史会写入 Docker 网桥地址。

禁止用 http_configuration_snippet 再写 real_ip_header：APISIX ngx_tpl 已输出
该指令，重复会导致 nginx: [emerg] duplicate 且网关起不来。
"""
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parent.parent
CONFIG_YAML = ROOT / "apisix" / "config.yaml"


def _http() -> dict:
    doc = yaml.safe_load(CONFIG_YAML.read_text(encoding="utf-8"))
    http = (doc.get("nginx_config") or {}).get("http") or {}
    assert isinstance(http, dict) and http, (
        "apisix/config.yaml 缺少 nginx_config.http.real_ip_*；"
        "Docker 发布端口会把客户端 IP 写成 172.25.0.1"
    )
    return http


def test_real_ip_header_is_xff():
    assert _http().get("real_ip_header") in ("X-Forwarded-For", "http_x_forwarded_for")


def test_real_ip_recursive_on():
    val = str(_http().get("real_ip_recursive", "")).lower()
    assert val in ("on", "true", "1")


def test_trusts_docker_rfc1918_and_loopback():
    sources = [str(x) for x in (_http().get("real_ip_from") or [])]
    for cidr in ("127.0.0.0/8", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"):
        assert cidr in sources, f"missing {cidr} in {sources}"


def test_no_duplicate_real_ip_header_snippet():
    doc = yaml.safe_load(CONFIG_YAML.read_text(encoding="utf-8"))
    snippet = (doc.get("nginx_config") or {}).get("http_configuration_snippet") or ""
    assert "real_ip_header" not in snippet, (
        "http_configuration_snippet 不得再写 real_ip_header（与 ngx_tpl 重复）"
    )


if __name__ == "__main__":
    import traceback

    failed = 0
    for name, fn in sorted(globals().items()):
        if name.startswith("test_") and callable(fn):
            try:
                fn()
                print(f"PASS {name}")
            except AssertionError as e:
                failed += 1
                print(f"FAIL {name}: {e}")
                traceback.print_exc()
    raise SystemExit(1 if failed else 0)
