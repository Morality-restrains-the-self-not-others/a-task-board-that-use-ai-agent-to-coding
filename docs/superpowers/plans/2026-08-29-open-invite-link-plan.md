# 开放式邀请链接 — 实施计划

- **日期**: 2026-08-29
- **设计**: `docs/superpowers/specs/2026-08-29-open-invite-link-design.md`

## Tasks

- [ ] **T1** dataMigrate `012_open_invite_link.sql`（utf8mb4、tenant_ 前缀、guarded ADD COLUMN + redemption 表）
- [ ] **T2** 红灯测试：默认 invite 仍 single；open 两人 join；max_uses 耗尽；email+open 400；非 admin 403
- [ ] **T3** `handleInvite`/`validate`/`join`/`pending` 实现 + 事务 + 日志（无完整 token）
- [ ] **T4** 事件 payload：INVITATION_CREATED `link_kind`/`max_uses`；MEMBER_JOINED `invitation_exhausted`
- [ ] **T5** 测试库 cleanup 含 `tenant_invitation_redemption`
- [ ] **T6** FE `InviteLinkMethodPanel` 单次/开放 + 测例
- [ ] **T7** `PeopleInvite.vue` 传 `link_kind`/`max_uses`；开放不强制成员名
- [ ] **T8** `PendingInvitations.vue` 已用/上限
- [ ] **T9** `PeopleJoin.vue` 展示 remaining_uses（若有）
- [ ] **T10** 跑 `invite_handlers_test.go` 与相关 FE 单测；`gofmt`/`py_compile` 不适用则 gofmt + node 测

不新增 Python API。不新增 Kafka topic。
