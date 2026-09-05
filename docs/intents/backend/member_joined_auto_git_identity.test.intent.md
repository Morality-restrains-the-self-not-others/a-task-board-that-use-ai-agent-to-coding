# 测试意图：公司成员多 Git 身份与加入自动创建

> 配对：`member_joined_auto_git_identity.intent.md`

## 单元 / 集成

| ID | 场景 | 断言 | 落点 |
|----|------|------|------|
| T1 | `BuildSystemGitEmail(member,tenant)` | 稳定、长度、后缀 `@daydaymoney.com` | taskTaskService |
| T2 | ensure-default 首次创建 | 201 + label=system-auto + is_default | taskTaskService |
| T3 | ensure-default 重复 | 200 同 id，行数不增 | taskTaskService |
| T4 | MEMBER_JOINED handler 调 ensure | DispatchSuccess | taskEvents |
| T5 | 缺 member_id 永久失败 | DispatchPermanent | taskEvents |
| T6 | tenant manage API 无权限 403 | 非 self 且无 perm | taskTaskService |
| T7 | invite join 发布含 member_name | publishEvent payload | taskTenantService |

## 手工 / E2E（可选）

1. 邀请加入 → Kafka `member-joined` → DB 出现 system-auto 身份
2. PeopleManage 「Git 身份」弹窗可增删改
