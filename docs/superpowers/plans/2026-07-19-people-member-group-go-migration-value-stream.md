# 价值流 — 人员页成员/邀请/分组迁 Go

日期：2026-07-19  
设计：`2026-07-19-people-member-group-go-migration-design.md`

## 最小价值增量（MVP）

1. **邀请链路**：生成邀请 →（可选邮件）→ validate → join → MEMBER_JOINED → 工作区访问
2. **管理人员**：列表 / 改角色名 / 启停 / 移除 / 待处理邀请撤销与重发
3. **管理分组**：分组 CRUD + 分组成员加减

## 端到端流（测试点）

| ID | 步骤 | 期望 | 对应用例 |
|----|------|------|----------|
| VS1 | Admin POST invite(link) | 201 + invite_token；DB 有 invitation | Go TestInviteLink |
| VS2 | GET validate-invite | valid=true | Go TestValidate |
| VS3 | 登录用户 POST join | 201 member；invitation accepted；可选 workspace access | Go TestJoin |
| VS4 | 非 admin GET company_members | 200 members=[] meta.has_permission=false | Go TestMembersDenied |
| VS5 | Admin PATCH update_role | 200 | Go TestUpdateRole |
| VS6 | Admin POST groups | 201；GET 列表含 memberCount | Go TestCreateGroup |
| VS7 | add_member / remove_member | 201/200 | Go TestGroupMembers |
| VS8 | 网关切流后前端三页可用 | 无 404/5xx | 手工/E2E |
| VS9 | Django 公网旧路径 | 404/410 | 路由扫描 |

## 与 value-stream-test-integration

本增量映射到组织与成员价值流；实现后在 `docs/flows/value-stream-test-integration.wsd` 补测试点 VS1–VS9（若图存在则追加）。
