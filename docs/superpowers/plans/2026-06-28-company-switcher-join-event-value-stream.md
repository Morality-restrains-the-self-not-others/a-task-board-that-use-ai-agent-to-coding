# Value Stream: Company Switcher & Member Join Event

> Derived from design: `docs/superpowers/specs/2026-06-28-company-switcher-join-event-design.md`

## Value Summary

用户接受邀请加入其他公司后，能够在 Navbar 中切换公司查看对应工作空间，且下游系统能通过域事件感知新成员加入。

## Related Value Streams

Greenfield — 无直接相关的现有价值流。与 `company-management` 流（`conf/value-stream.yaml` line 213）相邻，本流为扩展而非修改。

## End-to-End Flow

```
[邀请方发送邀请] → [被邀请方点击链接] → [验证token] → [POST join/]
                                                              ↓
                                          ┌─────────── MEMBER_JOINED event → Kafka
                                          │
                                          [PeopleJoin 跳转 /work-panel/]
                                                              ↓
                                          [Navbar 公司切换器] → 用户切换公司查看工作空间
```

## Value Increments

### Increment 1: Company Switcher in Navbar (Thin Slice — P0)
**Value to user:** 用户加入多个公司后可在 Navbar 下拉菜单中切换公司，无需手动修改 URL。
**Scope:**
- `Navbar.ui.vue`: 添加公司切换下拉 UI
- `Navbar.logic.vue`: 从 user/me API 读取 companies，处理切换事件
**Depends on:** nothing (现有 user/me API 已返回 companies 数组)
**Test:** 手动验证 + 前端单元测试

### Increment 2: Post-Join Redirect to Work Panel (P2)
**Value to user:** 接受邀请后立即看到新公司的工作面板和工作空间，而非需要额外导航的人员管理页。
**Scope:**
- `PeopleJoin.vue`: 修改 `joinTeam()` 跳转目标从 `/people/manage/` → `/work-panel/`
**Depends on:** Increment 1（用户需要公司切换器才能回到自己公司）
**Test:** E2E 验证邀请链接点击→加入→到达工作面板

### Increment 3: MEMBER_JOINED Kafka Event (P1)
**Value to user:** 下游系统（通知、审计、分析）可实时感知新成员加入，为后续功能（欢迎通知、审计日志）奠定基础。
**Scope:**
- `member_views.py`: join 方法新增 `send_event('MEMBER_JOINED', {...})`
- `core/kafka/`: 事件类型注册
- `conf/events/domain-events/member_joined/`: 事件配置
**Depends on:** nothing（独立后端改动）
**Test:** 单元测试验证 join 成功后 Kafka 消息已发送
