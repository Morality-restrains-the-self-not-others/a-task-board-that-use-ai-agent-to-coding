#!/usr/bin/env python3
"""Playwright E2E: Verify trace-log-journey var-status=4xx returns data.

Root cause: Grafana custom variable URL param matches option *text* field.
  Old: text="4xx (客户端错误)" → URL var-status=4xx didn't match → fallback literal "4xx"
  New: text="4xx" → URL var-status=4xx matches → resolves to value "4[0-9]{2}" ✅

This script:
  1. Pushes test logs (200, 401, 403, 404, 500) to Loki
  2. Opens the Grafana dashboard with var-status=4xx
  3. Verifies panels show data (not "No data")
  4. Verifies var-status=404 still works (regression)
  5. Verifies API returns clean labels (no Chinese in option text)
"""

import sys, time, json, requests, os

# Bypass unstable SOCKS5 proxy before importing playwright.
# See [[httpclient-socks-proxy-bypass]] and scripts/playwright_utils.py.
for _key in ('http_proxy', 'https_proxy', 'all_proxy',
             'HTTP_PROXY', 'HTTPS_PROXY', 'ALL_PROXY'):
    os.environ.pop(_key, None)

from playwright.sync_api import sync_playwright, TimeoutError as PwTimeout

GRAFANA = "http://localhost:3000"
LOKI = "http://localhost:3100"
USER = "admin"
PASS = "admin"
DASHBOARD_UID = "trace-log-journey"
TRACE_ID = "e2e-verify-4xx-fix-v2"


def push_fresh_logs():
    """Push test logs to Loki so dashboard has data."""
    ts_ns = str(int(time.time() * 1e9))
    entries = [
        {"service": "vfy", "level": "info",  "status": "200", "msg": "OK",            "trace_id": TRACE_ID},
        {"service": "vfy", "level": "warn",  "status": "401", "msg": "Unauthorized",   "trace_id": TRACE_ID},
        {"service": "vfy", "level": "warn",  "status": "403", "msg": "Forbidden",      "trace_id": TRACE_ID},
        {"service": "vfy", "level": "error", "status": "404", "msg": "Not Found",      "trace_id": TRACE_ID},
        {"service": "vfy", "level": "error", "status": "500", "msg": "Internal Error", "trace_id": TRACE_ID},
    ]
    payload = {"streams": [
        {"stream": {"job": "vfy"}, "values": [[ts_ns, json.dumps(e)] for e in entries]}
    ]}
    r = requests.post(f"{LOKI}/loki/api/v1/push", json=payload, timeout=10)
    assert r.status_code == 204, f"Loki push failed: {r.status_code}"
    print(f"  📤 pushed {len(entries)} test logs")
    time.sleep(3)


def login_via_api(context):
    """Get Grafana session cookie via API, inject into browser."""
    s = requests.Session()
    r = s.post(f"{GRAFANA}/login", json={"user": USER, "password": PASS})
    assert r.status_code in (200, 302), f"Login failed: {r.status_code}"
    cookies = s.cookies.get_dict()
    assert "grafana_session" in cookies, f"No session: {list(cookies.keys())}"
    context.add_cookies([{
        "name": "grafana_session", "value": cookies["grafana_session"],
        "domain": "localhost", "path": "/", "httpOnly": True,
        "secure": False, "sameSite": "Lax",
    }])
    print("  [auth] cookie injected")


def test_status_filter(page, status_value, label):
    """Navigate to dashboard with status filter; return panel-data summary."""
    push_fresh_logs()
    url = (f"{GRAFANA}/d/{DASHBOARD_UID}/{DASHBOARD_UID}"
           f"?orgId=1&from=now-2m&to=now&timezone=browser"
           f"&var-service=$__all&var-level=$__all"
           f"&var-trace_id={TRACE_ID}"
           f"&var-status={status_value}")
    page.goto(url, wait_until="networkidle", timeout=30000)
    page.wait_for_timeout(10000)

    state = page.evaluate("""() => {
        const panels = document.querySelectorAll('[class*="panel-container"]');
        const result = [];
        panels.forEach(p => {
            const header = p.querySelector('[class*="panel-title"]');
            const title = header ? header.textContent.trim() : '?';
            const hasNoData = p.querySelector('[class*="panel-empty"], [class*="no-data"]') !== null;
            result.push({title, hasNoData});
        });
        return result;
    }""")
    panels_ok = [p for p in state if not p["hasNoData"]]
    print(f"\n🧪 {label} (var-status={status_value})")
    print(f"  Final URL: {page.url[:120]}...")
    for p in state:
        icon = "✅" if not p["hasNoData"] else "❌"
        print(f"    {icon} {p['title']}")
    page.screenshot(path=f"/tmp/verify-{status_value}.png")
    return {"total": len(state), "with_data": len(panels_ok), "url": page.url}


def verify_labels_via_api():
    """Check dashboard API: status options have clean short-form texts."""
    s = requests.Session()
    s.auth = (USER, PASS)
    d = s.get(f"{GRAFANA}/api/dashboards/uid/{DASHBOARD_UID}").json()
    sv = [v for v in d["dashboard"]["templating"]["list"] if v["name"] == "status"][0]
    texts = [o["text"] for o in sv["options"]]
    xx_texts = [t for t in texts if "xx" in t.lower() and t != "All"]
    has_old = any(c in t for t in texts for c in ["客户端错误", "成功", "重定向", "服务端错误"])
    all_short = all(t in ("2xx", "3xx", "4xx", "5xx") for t in xx_texts)
    ok = all_short and not has_old
    print(f"\n🧪 API label check: texts={texts}")
    print(f"    Short xx: {all_short}, Old Chinese: {has_old} → {'✅' if ok else '❌'}")
    return ok


def main():
    results = {}
    with sync_playwright() as pw:
        browser = pw.chromium.launch(headless=True)
        ctx = browser.new_context(viewport={"width": 1920, "height": 1080})
        page = ctx.new_page()
        try:
            print("\n🔐 Authenticate…")
            login_via_api(ctx)

            # ── Test A: 4xx (the fixed case) ──
            a = test_status_filter(page, "4xx", "Test A: 4xx")
            results["4xx_has_data"] = a["with_data"] > 0
            results["4xx_url_resolved"] = "4%5B0-9%5D%7B2%7D" in a["url"]

            # ── Label check via API ──
            results["labels_clean"] = verify_labels_via_api()

            # ── Test B: 404 (regression) ──
            b = test_status_filter(page, "404", "Test B: 404")
            results["404_has_data"] = b["with_data"] > 0
        finally:
            ctx.close()
            browser.close()

    # ── Report ──
    print("\n" + "=" * 60)
    print("📊 ACCEPTANCE REPORT")
    print("=" * 60)
    all_pass = True
    for name, passed in results.items():
        icon = "✅" if passed else "❌"
        if not passed:
            all_pass = False
        print(f"  {icon} {name}: {'PASS' if passed else 'FAIL'}")
    print("=" * 60)
    print("✅ ALL CHECKS PASSED" if all_pass else "❌ SOME CHECKS FAILED")
    print("=" * 60)
    return 0 if all_pass else 1


if __name__ == "__main__":
    sys.exit(main())
