# Review — 人员页成员/邀请/分组迁 Go

日期：2026-07-19  
对照计划：`2026-07-19-people-member-group-go-migration-plan.md`

## 结论

**可合并（带过渡项）**：公网 API 已落 taskTenantService；网关已切流；Django 公网 router 已卸除；单测通过。

## Checklist

| 项 | 状态 |
|----|------|
| T1–T8 Go 服务与路由 | ✅ |
| T9 迁表脚本 | ✅ `db/scripts/migrate_people_tables_to_task_tenant.sh` |
| T10 Django tenant_client + create_company / resolve | ✅（部分路径仍 ORM 回退） |
| T11 Django 公网 urls 卸除 | ✅ |
| T12 DROP saas 旧表 | ⏳ 切流验证后执行（OPT） |
| Log Audit | ✅ 结构化小写 level；拒绝路径有 warn |
| Intent→Event | ✅ INVITATION_CREATED / MEMBER_JOINED publish（日志+payload；Kafka 生产投递可加强） |

## 非阻塞跟进

- 残留 Django `CompanyMember.objects` 读路径逐步改 client
- company_members 的 llm_budget meta 接 Cloud
- Kafka 真投递替换 log stub
