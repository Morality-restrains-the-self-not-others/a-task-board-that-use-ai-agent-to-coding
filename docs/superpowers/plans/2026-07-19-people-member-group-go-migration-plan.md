# 实施计划 — 人员页成员/邀请/分组迁 Go

日期：2026-07-19

## 任务清单

- [x] T1 脚手架 taskTenantService（go.mod/run/build/conf/runAll/registry）
- [x] T2 SQLite schema + migrate + import API
- [x] T3 公网 members handlers（invite/validate/join/list/update/toggle/delete/pending/revoke/resend）+ 单测
- [x] T4 公网 groups handlers + 单测
- [x] T5 Internal members/groups API
- [x] T6 事件发布 INVITATION_CREATED / MEMBER_JOINED
- [x] T7 join → taskProjectService workspace access
- [x] T8 OpenAPI + api_route_ownership + gateway apisix/routes
- [x] T9 迁表脚本 saas → task_tenant
- [x] T10 Django：tenant_client；改 create_company / resolve_member / internal resolve
- [x] T11 Django：卸除公网 members/groups 路由（410/404）
- [x] T12 table_ownership 四表换 owner；saas 四表 DROP（本地已落地；Django DROP migration 可选后续）
- [x] T13 意图文档 + 架构 v39 三类伴生
- [x] T14 回归：go test + 关键 Django 测试修红
- [x] T15 Review / Ship（架构 v39 → current；消费方 string-safe 收口）

## 事件契约任务

- [x] E1 文档对照表写入 intents
- [x] E2 publish 实现与消费者（既有 taskEvents）联调

## 切流顺序

1. 部署 taskTenantService（空库）
2. 停 Django 公网写（或维护窗）
3. import 四表
4. 网关切路由
5. Django urls 卸除
6. 验证三页
7. DROP saas 旧表（migration）

## 收口（2026-07-19）

- [x] C1 Django 门禁统一 `is_active_member` / `company_ids_for_user`
- [x] C2 `CompanyMember.objects.create` 种子写入 Go
- [x] C3 VERSION_HISTORY / `@status` v39 → current
