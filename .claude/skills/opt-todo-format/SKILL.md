---
name: opt-todo-format
description: Standard format for recording optimization suggestions into `.learnings/OPTIMIZATION_TODOS.md`. Use when writing or editing OPT entries during /goal execution, self-check iterations, or any time an actionable improvement is identified. Also use when completing or migrating OPT entries.
---

# OPT Todo Format — 优化建议录入规范

## Quick Reference

每个 OPT 条目使用以下精确模板（**按字段顺序，不可省略必填字段**）：

```markdown
### OPT-YYYYMMDD-NNN — 一句话标题（祈使句/陈述句，描述要做什么或发现了什么）

- **Status**: pending
- **Created**: YYYY-MM-DD
- **Context**: [1-3 句，说明发现背景：在做什么时发现的、影响了什么、当前行为是什么]
- **Action**: [具体可执行步骤，用 (1) (2) (3) 编号列出；若是单步操作直接写一句话]
- **Why**: [1-2 句，解释为什么这个优化是必要的——不改会怎样、改了收益是什么]
- **How to apply**: [实施指引：涉及的文件路径、函数名、配置 key、依赖的上下游服务等]
```

**Completed 条目额外字段：**

```markdown
- **Completed**: YYYY-MM-DD
```

**Cancelled 条目额外字段及说明：**

```markdown
- **Completed**: YYYY-MM-DD
- **Reason**: [取消原因：为什么不再需要做、被哪个其他 OPT 取代]
```

## Field Specifications

### 标题 (`OPT-YYYYMMDD-NNN — 标题`)

- **编号**: 通过 `python3 .learnings/_move_opt.py --next-id` 获取，**禁止手工自增**
- **日期**: 创建当天的 ISO 日期（如 `20260729`）
- **序号**: 三位数字，当日递增，不跨日复用
- **标题**: 祈使句或陈述句，一句话说清是什么。长度控制在 60 字以内

### Status

| 值 | 含义 | 何时使用 |
|---|---|---|
| `pending` | 待执行 | 初始状态 |
| `completed` | 已完成 | 实施完毕且验证通过后 → 填 `Completed` 日期 → **立即迁移到 COMPLETED 文件** |
| `cancelled` | 已取消 | 不再需要或已被其他方案取代 → 填 `Completed` 日期 + `Reason` → 迁移 |

### Context

说明"在什么场景下发现的"。包含：

- 当时在做什么操作
- 影响范围（哪个服务/模块/用户路径）
- 当前行为（为什么它是问题）

**反例**（太模糊）: "代码需要优化"  
**正例**: "执行 `run.sh` 启动 taskAuth 时发现 Kafka broker ping 失败导致 panic，影响了 taskAuth + taskEvents + taskGateway + taskCredentialService 4 个服务的启动"

### Action

具体、可执行的步骤。多步骤用 `(1) (2) (3)` 编号；单步骤直接写一句话。每条 Action 应包含：

- 操作类型（添加/修改/删除/创建）
- 目标文件路径
- 关键变更点

**反例**: "修复这个 bug"  
**正例**: "(1) 修改 `taskAuth/src/verification_code.go:154` 删除 `db.Exec(\"DELETE ...\")` 行；(2) 改为 `log.Error(...) + return err` 保留 DB 记录"

### Why

解释**必要性**——不改的后果或改的收益。避免空洞表述如"更好"、"应该"，要具体：

**反例**: "这样更好"  
**正例**: "Kafka 短暂不可用时验证码被误删，用户即使稍后重试也需重新生成。保留 DB 记录允许 Kafka 恢复后通过重试机制补发。"

### How to apply

实施指引，包含：

- 涉及的文件完整路径
- 函数名 / 配置 key / 常量名
- 依赖的前置条件（需先合并哪个 PR、需先部署哪个服务）
- 验证方式（运行什么测试、检查什么日志）

如果 Action 已经足够详细，此字段可简化或与 Action 合并。

## Lifecycle: Creation → Completion → Migration → Archive

```
创建 (pending)
  │  python3 .learnings/_move_opt.py --next-id 获取编号
  │  写入 OPTIMIZATION_TODOS.md
  │
  ▼
执行中
  │  修改代码/配置/文档
  │
  ▼
完成 (completed)
  │  Status → completed, 填 Completed 日期
  │  ⚠️ 立即迁移到 OPTIMIZATION_TODOS_COMPLETED.md
  │  ⚠️ 禁止只改 Status 不迁移
  │
  ▼
迁移后
  │  python3 .learnings/_archive_completed_opts.py
  │  非当日条目自动归档到 archive/completed/
```

### 迁移硬规则

1. `pending` → `completed` 后，**必须立即**将条目从 `OPTIMIZATION_TODOS.md` 移到 `OPTIMIZATION_TODOS_COMPLETED.md`
2. **禁止**在 pending 清单中留下 completed 条目——这是审计不兼容的
3. 迁移后运行 `python3 .learnings/_archive_completed_opts.py` 触发归档检查

## Examples

### Example 1: 典型 Bug 修复（pending 状态）

```markdown
### OPT-20260729-002 — sendEmailVerificationCode Kafka 失败时不应删除 DB 验证码

- **Status**: pending
- **Created**: 2026-07-29
- **Context**: `taskAuth/src/verification_code.go:154-161` 在 Kafka publish 失败时删除已写入 DB 的验证码，然后返回错误。这比优雅降级更差——既没发出邮件，又把有效的验证码删了。
- **Action**: 将 `sendEmailVerificationCode` 对齐 `handleCreateEmailInvitation` 模式：Kafka 失败时保留 DB 中的验证码、记录日志、返回友好错误，不删除已存储的验证码。
- **Why**: 当前行为导致 Kafka 短暂不可用时验证码丢失，用户即使稍后重试也需要重新生成。保留验证码允许 Kafka 恢复后通过其他路径补发。
- **How to apply**: 修改 `verification_code.go` — 删除 `db.Exec("DELETE ...")` 行，改为 log + return error。
```

### Example 2: 基础设施/监控类（pending，需外部依赖）

```markdown
### OPT-20260727-011 — taskBill SQLITE_BUSY 修复部署后 Grafana 监控验证

- **Status**: pending (需 Grafana Loki 外部基础设施)
- **Created**: 2026-07-27
- **Context**: taskBill `db.go` 将 `SetMaxOpenConns(4)` → `SetMaxOpenConns(1)` 以消除 SQLITE_BUSY。部署后需确认修复生效。
- **Action**: (1) 观察 Grafana Loki 日志 `{job="task-bill"} |= "SQLITE_BUSY"` 是否清零；(2) 对比 `consume-task-post-quota` 端点 P50/P95/P99 延迟变化
- **Why**: `SetMaxOpenConns(1)` 是正确的理论修复，但生产验证是不可或缺的最后一步。
- **How to apply**: 部署后通过 Grafana 仪表盘验证，无需代码变更。
```

### Example 3: Completed 条目

```markdown
### OPT-20260728-008 — 数据库清空后用户无公司修复：Kafka 事件重放方案

- **Status**: completed
- **Created**: 2026-07-28
- **Completed**: 2026-07-28
- **Context**: 清空数据库重新初始化后用户出现 companies: [], current_workspace: null。根因：公司/工作空间创建完全依赖 Kafka 事件链，初始化流程不重放事件。
- **Action**: (1) _ensure_user_has_company 回退为纯 send_event；(2) 03_03_repair_users_without_company.py 重写为 Kafka 事件重放；(3) db/saas/init.sh 更新。
- **Why**: 保持架构一致性，所有公司创建统一走 Kafka → Go consumer。
```

## Anti-Patterns

### 禁止

1. **手工自增编号** — 必须用 `_move_opt.py --next-id`，因为可能跨会话有其他人同时分配
2. **Status=completed 但留在 pending 清单** — 完成即迁移
3. **缺字段** — Context/Action/Why 为必填三要素；缺一则 OPT 不可执行
4. **模糊的 Action** — "优化性能"、"修复 bug" 这类不可执行；必须说清楚改哪个文件的什么
5. **在 Final Overview 正文中直接写优化建议而不落盘** — Overview 只引用 OPT 编号，详细内容必须在 `OPTIMIZATION_TODOS.md` 中
6. **超出范围的条目** — 需产品决策的写入 `PRODUCT_DECISIONS.md`

### 建议

- **标题体现可查找性** — 好的标题让你一周后仍能看懂是什么。包含函数名/文件名/服务名
- **Why 说后果不说情绪** — "不改会导致数据不一致" 优于 "这样不好"
- **Related 字段** — 如果与其他 OPT 或 memory 相关，添加 `- **Related**: [[memory-name]]` 或 `OPT-YYYYMMDD-NNN`
