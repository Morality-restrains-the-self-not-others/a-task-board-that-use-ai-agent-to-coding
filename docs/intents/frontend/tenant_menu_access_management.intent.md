# 意图：租户控制台访问管理（菜单级 RBAC）

> **Status:** superseded  
> **Superseded by:** `tenant_logical_resource_group_access.intent.md`  
> **Design:** `docs/superpowers/specs/2026-08-11-rbac-page-resource-group-v72-design.md`（ADR-0003）

## 用户故事

（历史）作为公司管理员，我希望在「人员管理 → 访问管理」中为成员或小组勾选可访问的左侧菜单……

本意图已被 **逻辑资源组（page ⊃ ui_region）** 方案取代：页面组仅为载体；资源组为 UI 组件区域；前后端用 `region:*`/`page:*` 同源 Enforce（A1/B2）。

## 迁移说明

- 入口「访问管理」保留。
- 勿再按「菜单 ↔ 粗码」实现保存/验收。
- 详见新意图与 v72 设计文档。
