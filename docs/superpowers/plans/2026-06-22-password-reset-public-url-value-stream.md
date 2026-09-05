# Value Stream: 密码重置邮件域名可配置化

> Derived from design: `docs/superpowers/specs/2026-06-22-password-reset-public-url-design.md`

## Value Summary

密码重置邮件中的链接使用外部可访问域名（而非 127.0.0.1），用户可直接点击链接完成密码重置。

## Related Value Streams

- **`user-auth`**: modification — 密码重置邮件链接的域名从硬编码 `host:port` 改为可配置的 `publicBaseUrl`

## End-to-End Flow

[用户点击密码重置] → [Django 构造重置 URL] → [_get_frontend_domain() 优先 publicBaseUrl_] → [Kafka EMAIL_SENT] → [用户收到含外部域名的邮件]

## Value Increments

### Increment 1: publicBaseUrl 配置与优先 (Thin Slice — the whole fix)

**Value to user:** 密码重置邮件中的链接使用外部可访问域名，点击即可完成密码重置。

**Scope:**
1. `conf/frontend/vue/config.yaml` — 新增 `publicBaseUrl: http://183.250.1.132:4000`
2. `core/config/settings_manager.py` — `get_frontend_domain()` 优先使用 `publicBaseUrl`，缺失时回退至 `host:port`

**Depends on:** nothing (独立配置修复)
