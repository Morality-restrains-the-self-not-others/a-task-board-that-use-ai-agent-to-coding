# 权限分析：厂商镜像版本行管理

- **Date:** 2026-08-29
- **Design:** `docs/superpowers/specs/2026-08-29-vendor-version-row-management-design.md`

## 角色

| 角色 | 范围 |
|------|------|
| 已认证厂商 (vendor JWT) | 仅自己的 `ContainerImage`（`img.VendorID == v.ID`） |
| 平台审核 staff | 既有 admin `unpublish`，本次不改授权面 |
| 匿名 / 他厂商 | 404 not found（与现网一致，避免 IDOR  enumeration 差异） |

## 端点

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| POST `/api/vendor/container-images/{id}/withdraw/` | vendor | Resource（本厂商镜像） | 撤回审核 / 下架 | requireVendor + owner 404 | ⚠️ approved 未实现 | VendorWithdraw；激活则同时 is_active=0 |
| DELETE `/api/vendor/container-images/{id}/` | vendor | Resource | 删除 | requireVendor + owner | ⚠️ 未调 CanDelete | DeleteGuard：禁止激活版本；禁止 pending_review |
| GET 列表 | vendor | Resource | 读 | requireVendor | ✅ | — |
| Admin unpublish | staff | Resource | 下架 | requireStaff | ✅ 本次顺带写 is_active=0 | UpdateContainerImageStatus 持久化 is_active |

## IDOR

路径 ID 先 `GetContainerImage` 再比对 `VendorID`。他厂商 ID → 404。不按 status 泄露 403。

## 条件

- 激活版本不可删（公开目录生效点）。
- 审核中不可删（须先撤回，避免绕过审核记录）。
- 下架已上架：厂商 actor_id 写入 reviewer_id，note 默认「厂商自行下架」。
