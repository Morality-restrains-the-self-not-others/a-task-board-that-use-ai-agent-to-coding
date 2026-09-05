# shareLib/authz — Companion

本目录实现租户/平台权限 Enforce 客户端（解析网关注入头 → O(1) 集合查找）。

## 必读元规则

改本包或新增租户 API 鉴权前，**必须**遵循：

- `.ai/01_project_constraints/45_tenant_logical_rbac_resource_groups.md`（逻辑资源组 v72 + 授予效果 v73）
- ADR-0003：`docs/adr/0003-logical-resource-group-page-region-rbac.md`
- ADR-0004：`docs/adr/0004-resource-group-grant-effect-view-operate.md`

## API 选用

| 场景 | API |
|------|-----|
| 旧粗码过渡 | `RequirePerm` / `HasPerm` |
| **租户页内区域 · 任意访问** | `RequireRegion` / `HasRegion`（= view∨operate∨legacy） |
| **租户页内区域 · 只读** | `RequireRegionView` / `HasRegionView` |
| **租户页内区域 · 编辑执行（新写 API）** | `RequireRegionOperate` / `HasRegionOperate` |
| **访问管理写** | `HasPeopleAccessWrite`（subject_list/region_matrix operate ∨ 存量 save_actions ∨ member:manage） |
| 侧栏整页 | `HasPage`（含 `page:key:view|operate` 与遗留 `page:key`） |
| 组资源数据范围 | `HasGroupResourceAccess`（`group-res:*`，≠ 逻辑资源组） |

## 禁止

- 在业务 handler 自造平行 ACL，绕过 PDP 注入的 `X-Tenant-Perms`
- 把 `group-res:` 与 `region:` 语义混用
- 写路径仅用 `HasRegionView` / 仅 FE 隐藏当作授权完成
