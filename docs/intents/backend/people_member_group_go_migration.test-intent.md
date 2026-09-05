# 测试意图 — 人员页成员/邀请/分组迁 Go

| ID | 场景 | 期望 |
|----|------|------|
| T1 | Admin invite link | 201 + invite_token |
| T2 | validate-invite 有效/过期 | valid true/false |
| T3 | join 成功 | MEMBER 创建；邀请 accepted |
| T4 | 非 admin company_members | has_permission=false |
| T5 | 创建者不可 DELETE/toggle | 400 |
| T6 | 分组 create + add/remove member | 201/200 |
| T7 | 网关路径可达 Go | 非 Django 默认回落 |
| T8 | Django 公网旧 members/groups | 404/410 |
