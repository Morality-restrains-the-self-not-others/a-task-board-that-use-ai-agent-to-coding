# Design: 邀请加入后公司切换 & 域事件 & 跳转优化

**Date:** 2026-06-28
**Status:** approved
**Related Issue:** bowek64234@divahd.com 接受邀请后看不到 contact@daydaymoney.com 的工作空间

---

## 问题回顾

用户 A 邀请用户 B 加入公司 C，后端 `join` API 正确创建了 `CompanyMember` 和 `WorkspaceAccess`，但：

1. **P0**: 前端缺少公司/租户切换器 — 用户 B 加入公司 C 后，Navbar/Sidebar 仅显示当前路由租户，无法切换到公司 C
2. **P1**: `join` 成功静默 — 无 Kafka 域事件发布，下游系统（通知、审计、分析）无法感知新成员加入
3. **P2**: 加入后跳转目标不合理 — `PeopleJoin.vue` 跳转到人员管理页，用户无法立即看到对方的工作空间

---

## Value Stream Impact

### Affected Streams
| Stream | Impact |
|--------|--------|
| `company-management` / `member-crud` | join 方法新增域事件发布 |
| `company-management` / `invitation-email` | 新增 `invitation-accepted` 消费者（可选） |
| `project-workspace` / `workspace-crud` | 新增 `workspace-member-added` 消费者（可选） |

### New Steps Needed
- `member-joined-event` — join 成功后发布 Kafka 事件
- `frontend-company-switcher` — Navbar 公司切换下拉

### Fields Impact
- `saas-backend.accounts_companymember.id` — 事件携带 member_id
- `saas-backend.projects_workspaceaccess.id` — 事件携带 workspace 访问信息

---

## Domain Concept Inventory

| Concept | Type | Context |
|---------|------|---------|
| CompanyMember | Entity | 组织与成员 |
| Company | Entity | 组织与成员 |
| WorkspaceAccess | Entity | 项目与工作空间 |
| Invitation | Entity (invariant: is_accepted → true after join) | 组织与成员 |
| MemberJoined | Domain Event | 组织与成员 → 通知/审计 |
| CompanySwitcher | UI Component | 前端导航 |

---

## 修复 1 (P0): Navbar 公司切换器

### 目标
在 Navbar 右侧用户区添加"当前公司"下拉菜单，允许用户在已加入的公司间切换。

### 设计
```
Navbar.ui.vue — 在"工作面板"链接旁新增公司切换下拉

┌─────────────────────────────────────────────────┐
│  AI项目推进平台  │ 价格 │ 代码仓库 │ [公司▾] [头像] │
│                                   ├ 我的公司      │
│                                   ├ example-user  │ ← 用户加入的其他公司
│                                   └ ...           │
└─────────────────────────────────────────────────┘
```

### 实现步骤
1. **Navbar.ui.vue**: 添加公司切换下拉 `<select>` 或自定义 dropdown
   - Props: `companies` (来自 user/me API 的 `companies` 数组)
   - Emits: `company-switched(companyId)`
2. **Navbar.logic.vue**: 
   - 从 `applyMePayload()` 的 `userInfo.companies` 读取所有公司
   - 监听 `company-switched` → `window.location.href = '/tenant/' + companyId + '/'`
   - 与 `Sidebar.vue` 已有的 `currentTenant` 逻辑保持一致
3. **UserSerializer.get_companies()**: 已有的 `companies` 字段足以支撑，无需修改后端

### 关键数据流
```
user/me API → companies[] → Navbar.logic.vue → Navbar.ui.vue 渲染下拉
                                                         ↓
                                             用户点击切换公司
                                                         ↓
                                         window.location = /tenant/{newId}/
```

---

## 修复 2 (P1): join API 发布 Kafka 域事件

### 目标
`join` 成功后发布 `MEMBER_JOINED` 域事件，使下游系统可感知新成员加入。

### 事件定义
```yaml
event_type: MEMBER_JOINED
data:
  member_id: "858949970..."       # CompanyMember.id
  user_id: "858949970248626176"   # 加入的用户 ID
  company_id: "850256677331562496" # 被加入的公司 ID
  company_name: "example-user"
  workspace_id: "857903329669984256" # 关联的工作空间 ID
  role: "member"                   # member | admin
  invitation_id: "858973598373654528" # 邀请记录 ID
  joined_at: "2026-06-28T07:34:54+00:00"
```

### 实现步骤
1. **member_views.py join 方法 (line ~268)**:
   - `WorkspaceAccess` 创建成功后，调用 `send_event('MEMBER_JOINED', {...})`
   - 放在 try/except 中，失败仅记录日志不阻断加入流程
2. **taskEvents 消费者 (可选，后续阶段)**:
   - 在 `taskEvents/cmd/` 下新增 `member_joined/1_send_welcome_notification/main.go`
   - 注册到 `config/event_registry.go` 的 `AllEvents` 列表

### 事件注册
- `core/kafka/handler_loader.py` 和 `conf/events/domain-events/` 需新增 `member_joined` 事件配置

---

## 修复 3 (P2): 加入后跳转到工作面板

### 目标
用户接受邀请后立即看到新公司的工作面板，而非人员管理页。

### 设计
```javascript
// PeopleJoin.vue joinTeam() line 218
// Before: router.push(`/tenant/${tenantId}/people/manage/`)
// After:  router.push(`/tenant/${tenantId}/work-panel/`)
```

工作面板 (`WorkPanel.vue`) 已有 WorkspaceSwitcher，用户可以立即看到并切换到邀请方的工作空间。这是更自然的入口。

### 额外优化
- 在 `PeopleJoin.vue` 成功提示中加入工作空间名称: `成功加入 ${teamName} 团队！正在跳转到工作面板...`

---

## 测试影响

| 测试 | 变更 |
|------|------|
| `accounts/view_test/CompanyMemberViewSet_test.py` | 新增 `test_join_publishes_member_joined_event` |
| `front_project/app/src/tests/domain/auth/auth_domain_model.test.js` | 新增公司切换逻辑测试 |
| `PeopleJoin.vue` | E2E 验证跳转到 work-panel |

---

## 实施顺序

```
P0 (公司切换器) → P2 (跳转优化) → P1 (Kafka 事件)
```

P0 和 P2 可并行（前端独立），P1 涉及后端和事件基础设施，需在前两项完成后进行。
