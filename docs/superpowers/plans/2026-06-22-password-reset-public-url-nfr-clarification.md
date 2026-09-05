# NFR 澄清: 密码重置邮件域名可配置化

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-06-22-password-reset-public-url-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-22-password-reset-public-url-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 说明 |
|------|------|------|
| 全部 | L0 | 不适用 — 纯配置变更，无新数据流、无外部依赖、无性能/安全/一致性影响 |

## 跳过声明

此为纯配置修复——2 个文件，约 5 行代码：
- `conf/frontend/vue/config.yaml` — 新增 `publicBaseUrl` 字段
- `core/config/settings_manager.py` — `get_frontend_domain()` 新增回退逻辑

无新数据流、无新外部依赖、无新服务间通信。所有 NFR 类别均跳过。
