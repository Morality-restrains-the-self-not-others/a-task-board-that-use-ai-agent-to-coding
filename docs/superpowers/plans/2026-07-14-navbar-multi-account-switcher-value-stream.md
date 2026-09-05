# 价值流：导航栏多账号切换

**日期**: 2026-07-14  
**设计**: `docs/superpowers/specs/2026-07-14-navbar-multi-account-switcher-design.md`

## 影响域

`user-auth`（`conf/value-stream.yaml`）

## 增量切片（按价值）

### Increment 1 — 账号槽存储（MVP 基础）

- 步骤: upsert / list / remove / max 5
- 字段: 无新 DB 字段；客户端 `savedAccounts`
- 测试: T1, T2

### Increment 2 — activate-session API

- 步骤: Token → enrich → 返回 user
- 字段: `task-auth.accounts_customtoken.key`, `task-auth.accounts_user.id`
- 测试: T3, T4, T5

### Increment 3 — Navbar 下拉 + 切换

- 步骤: 展开/切换/刷新
- 测试: T6, T7

### Increment 4 — 添加账号 / 退出当前

- 步骤: add_account 登录、logout 当前并切下一槽
- 测试: T8, T9

## YAML 片段（待同步 conf/value-stream.yaml）

```yaml
- name: multi-account-activate-session
  status: planned
  test_file: taskAuth/src/auth_activate_session_test.go
  fields:
    - name: task-auth.accounts_customtoken.key
      description: 切换目标账号 Token
    - name: task-auth.accounts_user.id
      description: Token 解析出的用户 ID
- name: multi-account-navbar-switch
  status: planned
  test_file: front_project/app/src/tests/domain/auth/saved_accounts_store.test.js
  fields:
    - name: task-auth.accounts_user.id
      description: 激活账号 userId 写入 cookie
```

## 最小可交付

Increment 1+2+3 即可满足核心目标；Increment 4 同迭代交付。
