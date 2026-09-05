# NFR — 逻辑资源组 v72

- **Date:** 2026-08-11
- **Default level:** L2；授权域 **L3**

| 类别 | 等级 | 说明 |
|------|------|------|
| Security | L3 | API 无 region → 403；禁止仅 FE 隐藏 |
| Consistency | L3 | FE/BE 同源 registry；B2 不展开粗码 |
| Performance | L2 | PDP 批量 SQL；header ≤7KB |
| Observability | L2 | `event=rbac_resource_group_*` 结构化日志 |
| Compatibility | L2 | 侧栏 `page:*` 优先，无 page 码时回退粗码（过渡） |
