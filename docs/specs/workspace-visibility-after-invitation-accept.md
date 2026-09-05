# 设计文档：用户注册后公司/工作空间缺失问题

> **状态:** 部分已修复（2026-08-10）
> **日期:** 2026-06-28
> **类型:** Bug 修复
> **优先级:** 🔴 Critical

### 2026-08-10 修复补记（手机注册 + 工作面板公司下拉）

**根因（生产复现 user `874599872605483008`）**: 手机/微信注册发布的 `USER_CREATED` 故意不带 `username`（避免手机号/微信昵称成为公司名）。`taskEvents` `0_create_company` handler 却对空 username 返回 `DispatchPermanent`（`missing username`）→ 进 DLT → **自有公司从未创建**。用户仅保留受邀公司成员资格 → Navbar `userCompanies.length <= 1` → 「工作面板」渲染为普通链接而非公司切换下拉。

**修复**:
1. handler 空 username 回退「我的公司」（与 `CreateCompanyForUser` / OPT-20260806-035 对齐）
2. `scripts/repair_users_without_company.py` 改为按 `companies/by-creator` 判断（受邀成员不再误跳过）
3. 存量用户：`python3 scripts/repair_users_without_company.py --user-id <id>`

下文 §2 的 Django post-register 静默吞错等历史分析仍保留作对照。

---

## 1. 问题描述

### 现象

`hedicip541@divahd.com` 接受 `contact@daydaymoney.com` 的邀请后，访问
`/tenant/850256677331562496/settings/task-panel/` 只看到**一个默认工作空间**。

### 系统设计（用户确认）

- 每个用户注册时，通过事件链自动创建**自己的公司** + **默认工作空间**：
  ```
  USER_CREATED → [port 18025] COMPANY_CREATED → [port 18029] WORKSPACE_CREATED
  ```
- 用户接受他人邀请后，成为对方公司的成员，可看到对方公司的所有工作空间

### 核心疑点

`hedicip541@divahd.com` **可能没有自己的公司和工作空间**——注册时的事件链在某处断裂了。

---

## 2. 🔴 根因：注册链路错误被静默吞掉

### 2.1 注册流程

```
用户注册 (email/password)
  │
  ▼
taskAuth Go: createUserWithEmailLogin()  ← 在 taskAuth DB 创建用户身份
  │
  ▼
taskAuth Go: djangoPostRegister()        ← HTTP POST 到 Django
  │                                         /api/internal/taskauth/post-register/
  │
  ▼
Django: post_register()                  ← 创建 User, 发送激活邮件,
  │                                         发送 USER_CREATED Kafka 事件
  │
  ▼
Kafka: USER_CREATED 事件
  │
  ├── [port 18025] 0_create_company     ← 创建 Company + CompanyMember
  │     └── 发布 COMPANY_CREATED 事件
  │           ├── [port 18027] 设置默认交付物体系
  │           ├── [port 18028] 设置默认进度体系
  │           └── [port 18029] 创建默认工作空间
  │                 └── 发布 WORKSPACE_CREATED 事件
  │                       └── [port 18030] 设置工作空间管理员+默认配置
  │
  └── [port 18038] 2_sync_user_profile  ← 同步 UserProfile.username
```

### 2.2 🔴 关键 Bug：`auth_register.go` L62-64

```go
// taskAuth/src/auth_register.go
func handleEmailRegister(w http.ResponseWriter, r *http.Request) {
    // ... 验证 email/password ...
    // ... createUserWithEmailLogin() ...

    // ⬇️⬇️⬇️ BUG IS HERE ⬇️⬇️⬇️
    if err := djangoPostRegister(r.Context(), userID, email, username, body, activationURL); err != nil {
        log.Printf("[taskAuth] post-register side effects: %v", err)
        // ← 错误仅被记录日志，不返回给调用方！
    }
    // ⬆️⬆️⬆️ BUG IS HERE ⬆️⬆️⬆️

    // 无论 djangoPostRegister 是否成功，都返回 201 "注册成功"
    writeJSON(w, http.StatusCreated, map[string]interface{}{
        "message":       "注册成功，激活邮件已发送，请查收邮件并激活账号",
        "user_existed":  false,
    })
}
```

**失败场景**:
1. `djangoPostRegister()` 内部调用 `POST /api/internal/taskauth/post-register/`
2. 如果 Django 端 **任何异常**（Kafka 不可用、DB 异常、网络超时等）导致返回 >= 400
3. `djangoPostRegister` 返回 error
4. Go 端 **只打日志，不报错**，返回 `201 Created` 给用户
5. 用户看到「注册成功」，但：
   - ❌ `USER_CREATED` 事件**未发送**
   - ❌ 消费链**未触发**
   - ❌ 公司**未创建**
   - ❌ 默认工作空间**未创建**

### 2.3 同样的 Bug 也存在于手机注册

```go
// taskAuth/src/auth_phone_register.go L67-80
enriched, enrichStatus, err := djangoPostPhoneRegister(r.Context(), userID, phone, email, body, validated)
if err != nil || enrichStatus >= 400 {
    // 静默回退为 minimal user JSON，不包含 company/workspace 信息
    userPayload, uerr := minimalUserJSON(userID)
    writeJSON(w, http.StatusCreated, map[string]interface{}{
        "token":   token,
        "user":    userPayload,     // ← 无 company/workspace
        "message": "注册成功",       // ← 用户仍看到"注册成功"
    })
    return
}
```

### 2.4 激活链路也有类似问题

```go
// taskAuth/src/handlers.go L138
_ = djangoPostActivate(r.Context(), lm.ObjectID, lm.MethodType, lm.Identifier)
// ← 错误被完全忽略（_ =）
```

### 2.5 Django 端：Kafka 异常会导致 post_register 500

```python
# task2app/Saas_project/accounts/taskauth_internal_views.py L178
send_event('USER_CREATED', {...})  # ← 无 try-except

# task2app/Saas_project/core/kafka/producer.py L47-49
except Exception as e:
    kafka_logger.error(f"Error sending message to Kafka: {e}")
    raise  # ← 异常向上传播 → Django 返回 500
```

**恶性循环**: Kafka 异常 → Django 500 → Go 收到 >=400 → 静默吞掉 → 用户无公司/工作空间

---

## 3. 为什么不影响「看到默认工作空间」

`hedicip541@divahd.com` 虽然可能没有自己的公司，但：

1. **接受了邀请** → `member_views.py` `join` action 创建了 `CompanyMember` (关联到邀请方的 company)
2. **访问邀请方的租户** → `WorkspaceViewSet.get_queryset()` 按 `company_id` 返回邀请方公司的工作空间
3. **看到默认工作空间** → 邀请方公司下至少有一个默认工作空间（邀请者注册时事件链成功创建）

**但看不到其他工作空间的原因** (二选一)：
- **A**: 邀请方公司下**确实只有一个**默认工作空间（无自定义工作空间）
- **B**: 其他工作空间存在但 `company_id` 不匹配

---

## 4. 验证方案

### 4.1 确认 hedicip541@divahd.com 是否有自己的公司

```sql
-- Step 1: 找到 hedicip541@divahd.com 的 user_id
SELECT u.id AS django_user_id, lm.identifier, lm.method_type
FROM accounts_user u
JOIN accounts_login_method lm ON lm.user_id = u.id
WHERE lm.identifier = 'hedicip541@divahd.com';

-- Step 2: 检查是否有自己的公司（作为 creator）
SELECT id, name, creator_id, created_at
FROM accounts_company
WHERE creator_id = '<user_id_from_step1>';

-- Step 3: 检查 CompanyMember 记录
SELECT cm.id, cm.company_id, cm.workspace_id, cm.is_admin, c.name AS company_name
FROM accounts_company_member cm
JOIN accounts_company c ON c.id = cm.company_id
WHERE cm.user_id = '<user_id_from_step1>';

-- Step 4: 如果有公司，检查其工作空间
SELECT w.id, w.name, w.is_default, w.company_id
FROM projects_workspace w
WHERE w.company_id IN (
    SELECT company_id FROM accounts_company_member
    WHERE user_id = '<user_id_from_step1>'
);
```

### 4.2 确认 taskAuth 日志中的错误

```bash
# 搜索 hedicip541@divahd.com 注册时的 post-register 错误
grep -i "post-register side effects\|hedicip541" logs/taskAuth*.log | tail -30
```

### 4.3 确认邀请方公司下的工作空间

```sql
-- 公司 850256677331562496 下的所有工作空间
SELECT id, name, is_default, company_id, created_at
FROM projects_workspace
WHERE company_id = 850256677331562496
ORDER BY created_at;
```

---

## 5. 解决方案

### 🔴 修复 1（必须）：注册失败时回滚用户创建

**文件**: `taskAuth/src/auth_register.go` L54-69

```go
// 修改前：
userID, activationToken, err := createUserWithEmailLogin(email, password)
if err != nil {
    log.Printf("[taskAuth] register error: %v", err)
    writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "register failed"})
    return
}

activationURL := fmt.Sprintf("/auth/activate/%s", activationToken)
if err := djangoPostRegister(r.Context(), userID, email, username, body, activationURL); err != nil {
    log.Printf("[taskAuth] post-register side effects: %v", err)
    // ← BUG：继续返回成功
}

writeJSON(w, http.StatusCreated, ...) // ← 应该只在 djangoPostRegister 成功时返回

// 修改后：
userID, activationToken, err := createUserWithEmailLogin(email, password)
if err != nil {
    log.Printf("[taskAuth] register error: %v", err)
    writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "register failed"})
    return
}

activationURL := fmt.Sprintf("/auth/activate/%s", activationToken)
if err := djangoPostRegister(r.Context(), userID, email, username, body, activationURL); err != nil {
    log.Printf("[taskAuth] post-register failed, cleaning up user %s: %v", userID, err)
    // 回滚：删除已创建的用户身份
    if delErr := deleteUser(userID); delErr != nil {
        log.Printf("[taskAuth] cleanup user %s failed: %v", userID, delErr)
    }
    writeJSON(w, http.StatusInternalServerError, map[string]string{
        "detail": "注册失败，请稍后重试",
    })
    return
}

writeJSON(w, http.StatusCreated, ...)
```

**需要新增 `deleteUser()` 函数**（或使用 taskAuth 现有的用户删除能力）。

### 🟡 修复 2（推荐）：增加自愈机制 — 登录时补建公司

**文件**: `task2app/Saas_project/accounts/taskauth_internal_views.py` `enrich_login` (L92-143)

在用户登录时检查是否缺少公司，若缺少则补发 `USER_CREATED` 事件：

```python
def enrich_login(request):
    # ... 现有逻辑 ...
    
    # 登录后检查：用户是否缺少自己的公司
    user_id = str(principal.id)
    has_own_company = Company.objects.filter(creator_id=user_id).exists()
    is_any_member = CompanyMember.objects.filter(user_id=user_id).exists()
    
    if not has_own_company and not is_any_member:
        # 用户因注册时 post-register 失败而没有公司
        # 补发 USER_CREATED 事件（幂等，消费者会跳过已存在的 company）
        logger.warning(f"[SELF-HEAL] User {user_id} has no company, re-firing USER_CREATED")
        try:
            send_event('USER_CREATED', {
                'user_id': user_id,
                'username': _login_method_username(user_id) or '',
                'email': principal.email or '',
                'phone': None,
                'is_active': principal.is_active,
            }, key=user_id)
        except Exception as e:
            logger.error(f"[SELF-HEAL] Failed to re-fire USER_CREATED for {user_id}: {e}")
```

### 🟡 修复 3（推荐）：Kafka 发送失败时异步重试

**文件**: `task2app/Saas_project/core/kafka/producer.py`

当前 Kafka 发送异常直接 raise，导致 `post_register` 整体 500。改为异步重试或写入死信队列（DLQ）。

### 🟢 修复 4（防御性）：增加管理命令修复存量用户

**新文件**: `task2app/Saas_project/accounts/management/commands/heal_missing_companies.py`

```python
"""修复没有公司的存量用户 — 补发 USER_CREATED 事件"""
class Command(BaseCommand):
    help = '为缺少公司的用户补发 USER_CREATED 事件'

    def handle(self, *args, **options):
        # 找到所有 account_user 中不在 accounts_company 和 accounts_company_member 中的用户
        orphan_users = User.objects.raw('''
            SELECT u.id FROM accounts_user u
            WHERE u.id NOT IN (SELECT creator_id FROM accounts_company)
            AND u.id NOT IN (SELECT user_id FROM accounts_company_member)
        ''')
        for user in orphan_users:
            send_event('USER_CREATED', {...}, key=str(user.id))
            self.stdout.write(f'Healed: {user.id}')
```

---

## 6. 价值流影响

| 价值流 | 步骤 | 影响 |
|--------|------|------|
| `company-management` | `member-crud` | join 流程不变，但受邀用户可能缺少自己的公司 |
| `project-workspace` | `workspace-crud` | 工作空间列表 API 无 bug |
| `domain-events-consumer-split` | `increment1-registration-chain-go` | **USER_CREATED 消费链可能未被触发** |
| (新) `user-auth` | `user-registration` | **需要新增注册事务性保证** |

---

## 7. 域概念清单

| Bounded Context | 实体 | 聚合根 | 领域事件 |
|----------------|------|--------|---------|
| 用户与认证 (Auth) | User, LoginMethod | User | USER_CREATED, USER_ACTIVATED |
| 组织与成员 (Organization) | Company, CompanyMember, Invitation | Company | COMPANY_CREATED |
| 工作空间 (Workspace) | Workspace, WorkspaceAccess | Workspace | WORKSPACE_CREATED |

**跨上下文事件链**: `USER_CREATED(Auth)` → `COMPANY_CREATED(Org)` → `WORKSPACE_CREATED(Workspace)`

---

## 8. 测试计划

| 测试类型 | 文件 | 场景 |
|---------|------|------|
| 单元测试 | `taskAuth/src/auth_register_test.go` | `djangoPostRegister` 失败时用户被清理，返回 500 |
| 单元测试 | `taskAuth/src/auth_register_test.go` | `djangoPostRegister` 成功时返回 201 |
| 集成测试 | `task2app/accounts/view_test/` | enrich_login 补发 USER_CREATED 事件 |
| E2E | `playwright/front_project/tests/` | 注册→登录→验证公司/工作空间存在 |

---

## 总结清单

- **根因定位**: 方案1（DB 查询确认 hedicip541@divahd.com 是否有自己的公司）、方案2（taskAuth 日志确认 post-register 是否报错）
- **修复方向**: 修复1（注册失败回滚用户, 🔴必须）、修复2（登录时自愈补建公司, 🟡推荐）、修复3（Kafka 异步重试）、修复4（管理命令修复存量用户）
- **影响范围**: taskAuth Go (`auth_register.go`, `auth_phone_register.go`, `handlers.go`) + Django (`taskauth_internal_views.py`, `kafka/producer.py`)
- **紧急程度**: 🔴 Critical — 所有在 `djangoPostRegister` 失败时注册的用户都受影响，且无感知
