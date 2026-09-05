# 实现计划：用户注册后公司/工作空间缺失修复

> 基于设计文档: `docs/specs/workspace-visibility-after-invitation-accept.md`
> 日期: 2026-06-28

---

## Task 1: 修复 taskAuth 邮箱注册 — djangoPostRegister 失败时回滚

**文件**: `taskAuth/src/auth_register.go` (L54-69)
**优先级**: 🔴 Critical
**预估**: ~15 行改动

- [ ] 1.1 新增 `deleteUser(userID string) error` 函数（或复用已有清理能力）
- [ ] 1.2 修改 `handleEmailRegister`: `djangoPostRegister` 失败时调用 `deleteUser(userID)` 回滚
- [ ] 1.3 失败时返回 HTTP 500 `{"detail": "注册失败，请稍后重试"}` 而非 201

## Task 2: 修复 taskAuth 手机注册 — 同样问题

**文件**: `taskAuth/src/auth_phone_register.go` (L67-80)
**优先级**: 🔴 Critical
**预估**: ~5 行改动

- [ ] 2.1 `djangoPostPhoneRegister` 失败时同样回滚用户（调用 `deleteUser`）
- [ ] 2.2 返回 HTTP 500 而非 201

## Task 3: 修复 taskAuth 激活链路 — djangoPostActivate 错误被忽略

**文件**: `taskAuth/src/handlers.go` (L138)
**优先级**: 🟡 Important
**预估**: ~3 行改动

- [ ] 3.1 `djangoPostActivate` 错误至少 warn 级别日志（而非 `_ =`）

## Task 4: Django enrich_login 自愈机制

**文件**: `task2app/Saas_project/accounts/taskauth_internal_views.py` (L92-143)
**优先级**: 🟡 Recommended
**预估**: ~25 行改动

- [ ] 4.1 在 `enrich_login` 末尾增加检查：用户是否有自己的公司或任何 CompanyMember 记录
- [ ] 4.2 若无，补发 `USER_CREATED` 事件（消费者幂等处理）
- [ ] 4.3 包装 try-except，不阻塞登录流程

## Task 5: Django 管理命令修复存量用户

**新文件**: `task2app/Saas_project/accounts/management/commands/heal_missing_companies.py`
**优先级**: 🟢 Nice-to-have
**预估**: ~40 行

- [ ] 5.1 查询所有无公司、无 CompanyMember 的 User
- [ ] 5.2 逐个补发 `USER_CREATED` 事件
- [ ] 5.3 输出修复统计

## Task 6: 后端测试

**文件**: `task2app/Saas_project/accounts/view_test/`
**优先级**: 🔴 Required
**预估**: ~60 行

- [ ] 6.1 测试 `enrich_login` 自愈：模拟无公司用户登录，验证 USER_CREATED 事件被发送
- [ ] 6.2 测试管理命令：创建孤儿用户，运行命令，验证事件发送

## Task 7: Go 端测试

**文件**: `taskAuth/src/auth_register_test.go` (新建或追加)
**优先级**: 🟡 Important
**预估**: ~40 行

- [ ] 7.1 测试 `djangoPostRegister` 失败时用户被删除
- [ ] 7.2 测试 `djangoPostRegister` 成功时用户保留

## Task 8: 端到端验证

**任务**: 手动验证 + 确认修复
**优先级**: 🟢 Nice-to-have

- [ ] 8.1 模拟 djangoPostRegister 失败场景，验证回滚行为
- [ ] 8.2 验证自愈机制在存量用户下次登录时生效
- [ ] 8.3 运行管理命令确认存量修复

---

## 执行顺序

```
Task 1 ──→ Task 2 ──→ Task 3 (Go 端修复，可并行)
Task 4 ──→ Task 5 ──→ Task 6 ──→ Task 7 (Python 端修复 + 测试)
Task 8 (验证)
```

## 涉及文件清单

| 文件 | 操作 | 任务 |
|------|------|------|
| `taskAuth/src/auth_register.go` | 修改 | 1 |
| `taskAuth/src/auth_phone_register.go` | 修改 | 2 |
| `taskAuth/src/handlers.go` | 修改 | 3 |
| `task2app/Saas_project/accounts/taskauth_internal_views.py` | 修改 | 4 |
| `task2app/Saas_project/accounts/management/commands/heal_missing_companies.py` | 新建 | 5 |
| `task2app/Saas_project/accounts/view_test/test_enrich_login_heal.py` | 新建 | 6 |
| `taskAuth/src/auth_register_test.go` | 新建 | 7 |
