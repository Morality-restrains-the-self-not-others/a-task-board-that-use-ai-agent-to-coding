# 公司成员多 Git 身份 + MEMBER_JOINED 自动建身份 — 实施计划

设计：`docs/superpowers/specs/2026-08-12-member-joined-auto-git-identity-design.md`  
架构：v77 application-integration（20260812-1535-cursor）  
状态：实现已完成（goal-mode）；公网验收见 OPT-20260812-027

## Slice 1 — taskTenant 发布 MEMBER_JOINED
- [x] 确认 topic `member-joined` / EventTopic 映射
- [x] invite join 成功路径 publish（payload 契约）
- [x] 公开建公司创建者成员路径 publish
- [x] internal upsert 新建成员时 publish
- [x] 单测：publish helper smoke；新建门闩（internal upsert isNew）

## Slice 2 — taskTaskService ensure-default + 管理 API（TDD）
- [x] `POST /api/internal/git-identities/ensure-default/`（邮箱哈希、label=`system-auto`、幂等）
- [x] `GET/POST .../tenant/{tid}/member/{mid}/`
- [x] `PATCH/DELETE .../tenant/{tid}/identity/{iid}/`
- [x] 鉴权：self / `member:manage` / `group-members:manage`
- [x] 单元测试 Red→Green（ensure + auth）

## Slice 3 — taskEvents consumer
- [x] Intent `member_joined/1_create_default_git_identity`（port 18060）
- [x] 调 ensure-default；4xx permanent / 5xx retryable
- [x] runAll / IntentTopic / EventTopic 注册
- [x] 消费单测

## Slice 4 — taskFE PeopleManage
- [x] MemberList 操作列「Git 身份」弹窗（列表/新增/设默认/删除）
- [x] UserGitIdentities 保持不变
- [x] 控行数；失败展示 traceId

## Slice 5 — 意图 / 架构收尾 / 精准重启
- [x] intents 与测试意图对齐验收
- [x] 登记 precise restart：taskTenantService、taskTaskService、taskEvents、taskFE
- [ ] `/10-ship` 时将 v77 VERSION_HISTORY 标 shipped（待精准重启 + 公网验收后）
