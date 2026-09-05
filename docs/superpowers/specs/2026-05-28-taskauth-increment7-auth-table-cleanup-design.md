# taskAuth Increment 7 — 旧 Auth 表清理与 API 收口

> 延续：`2026-05-28-taskauth-db-split-design.md`（Increment 6 已将读写迁至 `taskAuth/data/auth.db`）

## 1. 目标

在 Increment 6 双库分离基础上：

1. **所有用户-facing 认证写路径** 仅通过 taskAuth API（Django 桥接 delegate），不再保留 Django fallback 写 `LoginMethod` / `CustomToken`。
2. **共享库 `db.sqlite3` 删除** `accounts_login_method`、`accounts_customtoken` 物理表（保留 `accounts_user` 供业务 FK）。
3. 只读跨域查询（序列化、Kafka、账单邮箱解析）继续经 **DATABASE_ROUTERS → auth.db**，无需额外 HTTP 往返。

## 2. 成功标准（SMART）

| 标准 | 验收 |
|------|------|
| 无 default 写 | `TASKAUTH_USE_SEPARATE_DB=true` 时，grep 无 `LoginMethod.objects.create/save/update` 于 user-facing 路径（init 脚本除外） |
| 表已删除 | `db.sqlite3` 中不存在 `accounts_login_method`、`accounts_customtoken` |
| API 收口 | `send_verification_code` 已 delegate 至 taskAuth（当前唯一未 delegate 的 user-auth 写端点） |
| Fallback 移除 | UserViewSet 各 auth action 删除 delegate 后的 Django 本地实现块 |
| 测试 | user-auth 价值流 pytest 全绿；`scripts/verify_auth_tables_dropped.sh` 通过 |
| 生产 | `TASKAUTH_ENABLED=true` 为 runAll / 生产默认 |

## 3. 路径分类（现状审计）

### 3.1 已通过 taskAuth API（delegate ✅）

`UserViewSet`：`login`、`logout`、`email_register`、`phone_register`、`confirm_activation`、`resend_activation_email`、密码重置全套。

taskAuth Go 已实现对应 handler；Django 仍保留 **fallback 大块代码**（delegate 返回 `None` 时执行）——**Increment 7 删除**。

### 3.2 应新增 taskAuth API（❌ 待迁移）

| 端点 | 现状 | 方案 |
|------|------|------|
| `POST .../send_verification_code/` | 纯 Django `VerificationCodeService` | Go handler + delegate；验证码仍存 default（`VerificationCode` 非 auth 表） |

### 3.3 保留 Django ORM 只读（router → auth.db ✅ 无需 API）

经 `TaskAuthDatabaseRouter` 读 auth.db，**不删、不改**：

- `UserSerializer` / `LoginMethod.get_email_for_user`
- `billing/views/utils.py`、`accounts/views/utils.py`
- Kafka handlers（welcome、company_created）
- `sso_user_resolve.py`
- `frontend_hashed_password_backend.py`（legacy admin 登录，读 auth.db）

### 3.4 运维脚本（特殊）

| 脚本 | 策略 |
|------|------|
| `scripts/init/init_system.py`、`init_tenant.py`、`create_admin_ruandao.py` | 改为调用 taskAuth internal `bootstrap-login-method`（新增）或文档要求先启 taskAuth |
| `scripts/auxiliary/get_token.py`、`update_admin_password_hash.py` | 改读 auth.db 路径或走 taskAuth CLI |

### 3.5 应删除的重复服务层

- `accounts/services.py` 中注册/登录/重置重复逻辑（仅 fallback 使用）
- `accounts/services/password_reset_service.py`（delegate 已覆盖 ViewSet）

### 3.6 测试 fixture

pytest 中 `CustomToken.objects.create` / `LoginMethod.objects.create`：**保留**，router 写入 `:memory:` taskauth 或测试专用 auth.db fixture。

## 4. 共享库清理策略

### 4.1 前置条件

1. `migrate_from_shared_db.sh` 已执行，auth.db 含完整数据。
2. `TASKAUTH_ENABLED=true` + `TASKAUTH_USE_SEPARATE_DB=true`。
3. Increment 7 代码合并（fallback 已删、send_verification_code 已 delegate）。

### 4.2 清理步骤

```bash
# 1. 备份
cp task2app/Saas_project/db.sqlite3 task2app/Saas_project/db.sqlite3.bak

# 2. 删除 default 中 auth 表（保留 accounts_user）
taskAuth/scripts/drop_shared_auth_tables.sh

# 3. 验证
taskAuth/scripts/verify_auth_tables_dropped.sh
```

`drop_shared_auth_tables.sh` 内容：

```sql
DROP TABLE IF EXISTS accounts_customtoken;
DROP TABLE IF EXISTS accounts_login_method;
-- 不删 accounts_user、django_content_type（default 仍需要 contenttypes app）
```

### 4.3 Django migrate 行为

- `TaskAuthDatabaseRouter.allow_migrate`：`LoginMethod`/`CustomToken` **不在 default 建表**（已实现）。
- 新增 migration `accounts.00xx_drop_auth_tables_from_default`：`RunSQL` drop + `SeparateDatabaseAndState` 避免 state 漂移（可选，与脚本二选一；推荐 **脚本 + state migration**）。

### 4.4 回滚

- 恢复 `.bak`；或从 auth.db ATTACH 回灌 `accounts_login_method` / `accounts_customtoken` 至 default。

## 5. 架构（Increment 7 后）

```
前端 ──▶ Django UserViewSet ──delegate──▶ taskAuth Go (:8003)
                                              │
                                              ├─ auth.db（login_method, customtoken, auth user）
                                              └─▶ Django internal（sync-user, 邮件, Kafka）

Django default db.sqlite3
  ├─ accounts_user（业务 FK，sync-user 维护）
  └─ ❌ 无 accounts_login_method / accounts_customtoken

Django 只读跨域 ──router──▶ auth.db
Token 认证 ──CustomTokenAuthentication──▶ auth.db + User@default
```

## 6. 领域概念清单（→ Step 5 DDD）

| 概念 | 归属 |
|------|------|
| Bounded Context | **Auth**（taskAuth）vs **Tenant**（Django default User FK） |
| Aggregate | `LoginMethod` + `CustomToken` 根在 auth.db；`User` 双写（auth 最小集 + default 业务集） |
| Domain Event | USER_CREATED 仍由 Django internal 发出（不变） |
| Repository | Go `db.go` 为 auth 写；Django router 为只读 |

## 7. 价值流影响

影响 `value-stream.yaml` → **user-auth** 全部 active 步骤：

- 新增/明确 `send-verification-code` 步骤指向 task-auth
- 字段源统一为 `task-auth.*`（删除 `saas-backend.accounts_login_method.*` / `accounts_customtoken.*` 冗余字段条目）
- 新增步骤 `auth-table-cleanup`（planned → active）：验证 shared DB 无 auth 表

跨流：`frontend-auth-guard-redirect` 不变；`billing` 只读邮箱仍经 router。

## 8. 实施切片（→ Step 6 Plan）

| 切片 | 内容 | 风险 |
|------|------|------|
| **7.1** | `delegate_send_verification_code` + Go handler | 低 |
| **7.2** | 删除 UserViewSet auth fallback 代码块 | 中（需 TASKAUTH_ENABLED 默认 true） |
| **7.3** | 删除/标记废弃 `UserAuthService`、password_reset_service ViewSet 外引用 | 低 |
| **7.4** | drop 脚本 + verify 脚本 + migration state | 中（数据不可逆） |
| **7.5** | init 脚本改 internal bootstrap | 低 |
| **7.6** | value-stream + pytest 全量回归 | — |

## 9. NFR 预设（Step 4 默认 L2，auth 域 L3）

- **可用性 L3**：删除 fallback 后 taskAuth 不可用 → Django 返回 503（非静默 fallback）
- **一致性 L2**：sync-user 失败记录日志 + 告警；不阻塞注册响应
- **安全 L3**：drop 脚本仅本地/运维执行；生产需备份 gate

## 10. 非目标

- 不迁移 `VerificationCode` 表（仍 default）
- 不迁移 `accounts_user` 出 default（业务 FK 约束）
- 不在本增量实现 taskAuth 只读 HTTP API（router 已足够）

## 11. 推荐方案

**方案 A（推荐）**：删除 fallback + drop 共享表 + send_verification_code delegate。  
**方案 B**：保留 fallback 但 drop 表——**不可行**（fallback 写 default 会建表失败）。  
**方案 C**：全部改 HTTP 读 API——过度工程，拒绝。

采用 **方案 A**，分 6 个切片顺序交付；7.4 仅在 7.1–7.3 测试通过后执行。
