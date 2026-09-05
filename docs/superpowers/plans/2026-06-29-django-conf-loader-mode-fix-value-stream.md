# Value Stream: Django conf_loader mode 模板展开修复

> Derived from design: `.claude/skills/1-brainstorming-设计文档/design.md`

## Value Summary

Django 侧 base.yaml 解析与 runAll 侧行为一致：mode 字段 `${DEPLOY_MODE:-local}` 正确展开，非法 mode 立即抛错而非静默落入默认模式。

## Related Value Streams

- **deploy-mode-addressing** (`2026-06-29`): **extension** — 修补 Django `conf_loader.py` 中 `_load_domain_map()` 的 mode 解析，补齐 deploy-mode-addressing 在 Django 侧的缺失

## End-to-End Flow

[Django 启动] → [conf_loader._load_domain_map()] → [展开 mode `${DEPLOY_MODE:-local}` → `local`] → [解析 local 段地址] → [allowedHost 正确为 183.250.1.132:8001] → [ALLOWED_HOSTS 包含 183.250.1.132]

## Value Increments

### Increment 1: mode 模板展开 + fail-fast (单增量)
**Value to user:** 无 `DEPLOY_MODE` 环境变量时 Django 也能正确解析为 local 模式；非法 mode 值启动即失败
**Scope:**
- `conf_loader.py:112` — 展开 mode 字段 `${DEPLOY_MODE:-local}` 模板
- `conf_loader.py:115+` — 非法 mode 抛 `ImproperlyConfigured`
**Depends on:** deploy-mode-addressing stream（base.yaml 已有的 mode/gateway 字段）
