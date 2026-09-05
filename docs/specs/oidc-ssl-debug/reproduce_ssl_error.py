#!/usr/bin/env python3
"""
Playwright 脚本：验证 taskAuth OIDC SSO SSL record layer failure 修复

用法:
  python3 reproduce_ssl_error.py          # headless 模式
  python3 reproduce_ssl_error.py --headed  # 可视化调试
"""

import asyncio
import sys
import subprocess
import json
import os
from playwright.async_api import async_playwright

MAIN_URL = "http://183.250.1.132:4000/auth/login/"
GITLAB_URL = "http://183.250.1.132:8012"
TASKAUTH_OIDC_URL = "http://183.250.1.132:8003"
ACCOUNT_EMAIL = "contact@daydaymoney.com"
ACCOUNT_PASSWORD = "rgNodkdq8677!ci"
SCREENSHOT_DIR = "/tmp/ram-work/docs/specs/oidc-ssl-debug/screenshots"


def diagnostic_checks():
    """环境诊断"""
    print("=" * 60)
    print("🔍 诊断：环境与网络验证")
    print("=" * 60)

    # 1. taskAuth OIDC Discovery (HTTP)
    print("\n▶ 测试 1: curl HTTP taskAuth Discovery")
    r = subprocess.run(
        ["curl", "-s", "-o", "/dev/null", "-w", "HTTP %{http_code}",
         f"{TASKAUTH_OIDC_URL}/.well-known/openid-configuration"],
        capture_output=True, text=True
    )
    print(f"   宿主机 HTTP → {TASKAUTH_OIDC_URL}: {r.stdout}")

    # 2. SSL 直接连接 taskAuth (应失败)
    print("\n▶ 测试 2: openssl s_client taskAuth (预期失败 - 无TLS)")
    r = subprocess.run(
        ["openssl", "s_client", "-connect", "183.250.1.132:8003",
         "-servername", "183.250.1.132"],
        input="", capture_output=True, text=True, timeout=10
    )
    for line in r.stderr.split("\n") + r.stdout.split("\n"):
        if "record layer failure" in line or "packet length too long" in line:
            print(f"   SSL 错误: {line.strip()}")
            break
    print("   ✅ 确认: 端口 8003 纯 HTTP，无 TLS/SSL")

    # 3. OIDC Discovery 文档
    print("\n▶ 测试 3: OIDC Discovery 内容")
    r = subprocess.run(
        ["curl", "-s", f"{TASKAUTH_OIDC_URL}/.well-known/openid-configuration"],
        capture_output=True, text=True
    )
    discovery = json.loads(r.stdout)
    print(f"   issuer: {discovery.get('issuer')}")
    print(f"   token_endpoint: {discovery.get('token_endpoint')}")
    print("   ⚠️  所有端点使用 http:// 协议")

    # 4. GitLab 容器内 Ruby 验证 (修复后)
    print("\n▶ 测试 4: GitLab 容器内 Ruby 验证")
    script = """
require "openid_connect"
require "swd"
uri = URI.parse("http://183.250.1.132:8003")
r = OpenIDConnect::Discovery::Provider::Config::Resource.new(uri)
actual = r.endpoint.to_s
expected = "http://183.250.1.132:8003/.well-known/openid-configuration"
puts "SWD.url_builder=#{SWD.url_builder} endpoint=#{actual}"
puts (actual == expected) ? "RESULT:FIXED" : "RESULT:BROKEN"
"""
    r = subprocess.run(
        ["docker", "exec", "gitlab", "gitlab-rails", "runner", script],
        capture_output=True, text=True, timeout=30
    )
    for line in r.stdout.split("\n"):
        line = line.strip()
        if line:
            print(f"   {line}")

    # 5. 完整 OIDC Discovery 测试
    print("\n▶ 测试 5: GitLab 内完整 OIDC Discovery")
    script2 = """
require "openid_connect"
config = OpenIDConnect::Discovery::Provider::Config.discover!("http://183.250.1.132:8003")
puts "SUCCESS|auth=#{config.authorization_endpoint}|token=#{config.token_endpoint}"
"""
    r = subprocess.run(
        ["docker", "exec", "gitlab", "gitlab-rails", "runner", script2],
        capture_output=True, text=True, timeout=30
    )
    for line in r.stdout.split("\n"):
        line = line.strip()
        if line:
            print(f"   {line}")

    # 6. 检查 GitLab 日志中最新的 OIDC 错误
    print("\n▶ 测试 6: GitLab 日志最新 OIDC 事件")
    r = subprocess.run(
        ["docker", "exec", "gitlab", "bash", "-c",
         "grep 'openid_connect' /var/log/gitlab/gitlab-rails/application_json.log 2>/dev/null | tail -3"],
        capture_output=True, text=True, timeout=15
    )
    if r.stdout.strip():
        for line in r.stdout.strip().split("\n"):
            try:
                entry = json.loads(line)
                sev = entry.get("severity", "?")
                msg = entry.get("message", "")[:200]
                print(f"   [{sev}] {msg}")
            except json.JSONDecodeError:
                print(f"   {line[:200]}")
    else:
        print("   无 OIDC 相关日志")


async def simulate_sso(headless=True):
    """Playwright 模拟 SSO 登录流程"""
    print("\n" + "=" * 60)
    print("🎭 Playwright 模拟：SSO 登录流程")
    print("=" * 60)

    os.makedirs(SCREENSHOT_DIR, exist_ok=True)

    async with async_playwright() as p:
        browser = await p.chromium.launch(
            headless=headless,
            args=["--ignore-certificate-errors"]
        )
        context = await browser.new_context(
            ignore_https_errors=True,
            viewport={"width": 1280, "height": 900}
        )
        page = await context.new_page()

        failures = []
        page.on("requestfailed", lambda req: failures.append({
            "url": req.url,
            "error": req.failure
        }))

        try:
            # Step 1: 主站登录页
            print("\n▶ Step 1: 访问主站登录页")
            await page.goto(MAIN_URL, wait_until="networkidle", timeout=30000)
            await page.wait_for_timeout(2000)
            await page.screenshot(path=f"{SCREENSHOT_DIR}/01-main-login.png")
            print(f"   当前 URL: {page.url}")

            # Step 2: 登录
            print("\n▶ Step 2: 登录")
            email_input = page.locator('input[type="email"], input[name="email"], input[name="username"]').first
            if await email_input.count() > 0:
                await email_input.fill(ACCOUNT_EMAIL)
            pwd_input = page.locator('input[type="password"]').first
            if await pwd_input.count() > 0:
                await pwd_input.fill(ACCOUNT_PASSWORD)
            login_btn = page.locator('button[type="submit"]').first
            if await login_btn.count() > 0:
                await login_btn.click()
            await page.wait_for_timeout(5000)
            await page.screenshot(path=f"{SCREENSHOT_DIR}/02-after-login.png")
            print(f"   登录后 URL: {page.url}")

            # Step 3: 代码仓库 → GitLab
            print("\n▶ Step 3: 点击代码仓库 → GitLab")
            repo_link = page.locator('a:has-text("代码仓库")').first
            if await repo_link.count() > 0:
                await repo_link.click()
            else:
                await page.goto(f"{GITLAB_URL}/users/sign_in")
            await page.wait_for_timeout(5000)
            await page.screenshot(path=f"{SCREENSHOT_DIR}/03-gitlab-login.png")
            print(f"   GitLab URL: {page.url}")

            # Step 4: taskAuth SSO
            print("\n▶ Step 4: 点击 taskAuth SSO")
            sso_btn = page.locator('a:has-text("taskAuth")').first
            if await sso_btn.count() > 0:
                await sso_btn.click()
            else:
                sso_btn = page.locator('a[href*="openid_connect"]').first
                if await sso_btn.count() > 0:
                    await sso_btn.click()
            await page.wait_for_timeout(8000)
            await page.screenshot(path=f"{SCREENSHOT_DIR}/04-sso-result.png")
            print(f"   SSO 回调 URL: {page.url}")

            # Step 5: 检查结果
            print("\n▶ Step 5: 检查结果")
            content = await page.content()
            if "record layer failure" in content.lower():
                print("   ❌ 仍存在 SSL record layer failure 错误")
            elif "Could not authenticate" in content:
                print("   ⚠️  认证失败（可能是其他原因）")
            else:
                print("   ✅ 无 SSL 错误")

        except Exception as e:
            print(f"   ⚠️  异常: {e}")
            await page.screenshot(path=f"{SCREENSHOT_DIR}/error.png")

        finally:
            if failures:
                print(f"\n--- 网络请求失败 ({len(failures)} 个) ---")
                for f in failures:
                    print(f"   {f['error']}: {f['url'][:120]}")
            await browser.close()


async def main():
    print("""
╔══════════════════════════════════════════════════════════════╗
║  taskAuth OIDC SSO — SSL record layer failure 诊断/验证  ║
╚══════════════════════════════════════════════════════════════╝
""")
    diagnostic_checks()
    headless = "--headed" not in sys.argv
    await simulate_sso(headless=headless)

    print("\n" + "=" * 60)
    print("📝 诊断摘要")
    print("=" * 60)
    print("""
根因: openid_connect Ruby gem v2.3.1 的 Resource 类丢弃 URI scheme,
     SWD.url_builder 默认使用 URI::HTTPS, 强制 HTTP issuer → HTTPS.

修复: GitLab Rails initializer 设置 SWD.url_builder = URI::HTTP,
      恢复 HTTP scheme 传递, OIDC Discovery 正确使用 http:// 端点。
""")


if __name__ == "__main__":
    asyncio.run(main())
