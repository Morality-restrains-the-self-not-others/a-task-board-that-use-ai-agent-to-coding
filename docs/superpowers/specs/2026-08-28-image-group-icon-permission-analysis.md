# 镜像组图标 — 角色权限分析

- **日期**: 2026-08-28
- **设计**: `docs/superpowers/specs/2026-08-28-image-group-icon-design.md`

## 角色

| 角色 | 说明 |
|------|------|
| 已获准厂商 vendor | `requireVendor`，仅操作本 vendor 的组与 file_key |
| 匿名/主站用户 | 只读公开图标流与 catalog 中的 `icon_url` |
| 平台 staff | 无单独写图标接口；审核流不改 |

## 端点权限

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| POST icon-upload-url / complete / local-put | vendor | Vendor 资源 | write | requireVendor + OwnsFileKey(userID, file_key) | ✅ | kind 仅 `image_group_icon` |
| POST/PUT image-groups | vendor | Vendor 资源 | write | requireVendor + 组 vendor_id 匹配 | ✅ | 校验 icon_file_key 归属本 vendor |
| GET public .../icon | 匿名 | 公开读 | read | 组存在即流式；无 key → 404 | ✅ | 不列目录；禁止 `..`；不打 file_key 全量到日志 |
| GET catalog / vendor list | 既有 | 读 | read | 既有 | ✅ | 仅多返回 icon_url |

## IDOR

- file_key 必须 `OwnsFileKey(vendor.ID, key)`，禁止绑定他厂商对象。
- 公开 GET 只按 group id 取该组自己的 key，不接受客户端传入 file_key。

## 不引入新 RBAC 角色
