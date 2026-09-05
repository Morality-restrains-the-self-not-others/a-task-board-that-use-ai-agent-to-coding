# Plan — RBAC 逻辑资源组 v72 P1

- **Date:** 2026-08-11
- **Status:** executing via /goal

## Goal SMART

1. `dataMigrate/taskAuth/032_*.sql` 三表 + 控制台 page/region 种子 + API members（people.access）
2. PDP 注入 `region:*`/`page:*`；tenant_admin 全系统 RG；**不**从 RG 展开粗码
3. `shareLib/authz`：`RequireRegion`/`HasRegion`/`HasPage` + 单测
4. API：目录树 + 角色 RG 读写；写路径 Enforce Region 或 member:manage
5. FE：访问管理 Region 树保存；侧栏 `page:*` 优先 + 粗码回退
6. 相关单测绿；登记精准重启 taskAuth/taskFE

## Tasks

- [x] T1 DDL + seed
- [x] T2 shareLib region helpers + tests (Red→Green)
- [x] T3 PDP expand + tests
- [x] T4 HTTP handlers + wire
- [x] T5 FE tree + save + nav
- [x] T6 Review + ship notes + OPT

## Event tasks

- [x] P1：依赖 membership_rev / 角色更新失效缓存（`publishRoleChanged`）
- [ ] OPT：显式 `RoleResourceGroupsChanged` Kafka（非阻断）
