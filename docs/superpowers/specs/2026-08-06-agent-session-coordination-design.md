# 多智能体会话互知与编辑冲突防护设计 — Agent Session Coordination

> **迭代版本**: v14 (enterprise-landscape) / v66 (application-integration) 🎯 target
> **设计日期**: 2026-08-06 00:39
> **作者**: claude
> **需求**: 多个智能体会话同时执行时能互相知晓，避免相互间的编辑互相冲突
> **关联**: v13 git-hooks-version-control（分发机制复用）、v65 nightly-test-sweep（claude-agent 无头修复接入）、`claude-agent/` Go CLI

---

## 1. 背景与问题

本项目以 `/tmp/ram-work` 为主工作区，40+ 子模块仓库（16 Go 服务 + taskFE + 基建仓）。智能体工作形态已经多样化且**可以同时存在**：

| 会话形态 | 载体 | 现状 |
|---------|------|------|
| 交互式 Claude Code 会话 | 开发者终端 `claude` | 无任何会话感知 |
| 无头 `claude-agent run` | Go CLI → `claude -p`（夜间自愈等） | 无会话感知 |
| 夜间巡检 | `nightly-test-sweep.sh`（00:00–08:00 cron） | 仅 `flock` 防自身重叠 |
| CI / pre-commit 门禁 | `.githooks/pre-commit` 等 | 仅提交时门禁 |

**问题**：任意两个会话可能同时编辑同一个子仓（如交互会话改 `db`，夜间自愈的 claude-agent 也在改 `db`）。现有防护互不相通：

1. `flock` 只保护「巡检进程」自身，不保护「巡检 vs 交互会话」
2. pre-commit/pre-push 门禁在**提交时**才拦截 — 冲突早已产生
3. worktree 隔离依赖人工自觉，无强制、无知晓
4. 会话之间**零感知**：A 会话不知道 B 会话正在改哪个仓

## 2. 现状勘察（2026-08-06 实测）

| 机制 | 位置 | 保护范围 | 缺口 |
|------|------|---------|------|
| `flock` 互斥 | `scripts/nightly-test-sweep.sh:27-32` | 单巡检进程 | 不跨会话类型 |
| 子仓优先提交门禁 | `.githooks/pre-commit`、`pre-push`（v13，HOOK_VERSION 1.0.0） | 提交/推送顺序 | 不识别「谁在改」 |
| pre-commit 框架随机单测 | 同 pre-commit | 提交质量 | 同上 |
| worktree 并行规范 | `.claude/skills/git-workflow-and-versioning` | 手动流程 | 无强制 |
| claude-agent Go CLI | `claude-agent/src/`（run/interactive/show-config/tools） | 单次任务封装 | 无 session/lock 子命令 |
| 轨迹记录 | `claude-agent/src/trajectory/recorder.go` | 会话内审计 | 无跨会话注册表 |

**结论**：无「会话注册表 + 锁/租约 + 心跳 + 跨会话互知」这一层基础设施；本设计补齐该层，且与 v13 分发机制、v65 巡检编排天然衔接。

## 3. 根因分析

- 冲突的本质是**共享工作区并发写**：多会话共享同一 meta root 的 40+ 子仓，无协调原语
- 现有门禁全部集中在**提交点**（git 层），而写冲突发生在**编辑点**（文件系统层），时间差就是冲突窗口
- 无会话身份：无法回答「现在谁在改这个仓」「锁是谁持有的、还活着吗」

## 4. 目标与非目标

### 目标

- **互知**：任何会话可查询当前活跃会话、各自占用哪些仓库、心跳状态
- **仓库级锁**：每子仓一把租约锁，多仓按规范顺序获取（防死锁），带 TTL + 心跳（防活锁/僵尸锁）
- **分层策略**：交互会话冲突→警告（人决策）；无头/巡检→强制（等待→死锁检测→暂停对方→shadow edit）
- **暂停/恢复**：必要时暂停其他会话（SIGSTOP 进程树），完成后恢复
- **Shadow Edit**：必要时把文件拷贝到临时目录编辑完再原子拷回（带校验和 + 冲突报告）
- **全类型覆盖**：交互式 / 无头 claude-agent / 夜间巡检 / CI·pre-commit
- **复用既有**：v13 分发器（38 仓 hooks）、v65 flock、claude-agent Go 二进制

### 非目标

- 跨主机/分布式协调（范围：同主机单 meta root）
- 文件级锁（粒度：仓库级；文件冲突交由 git/人工）
- 自动合并冲突解决（git 层面职责）
- 新增任何服务端 HTTP 接口（本机制纯本地开发工具链）

## 5. 方案设计

### 5.1 会话注册表（Session Hub）

统一目录 `logs/sessions/`（相对 meta root；meta root 由 `.gitmodules` 标记向上发现，`SESSION_HUB_DIR` 可覆盖）：

```
logs/sessions/
├── registry/<session_id>.json     # 每会话一条记录
├── locks/<repo>.json              # 每子仓一把锁（repo = 相对 meta root 路径）
├── shadow/<session_id>/...        # shadow edit 快照区（见 5.5）
└── audit/<date>.log               # 加锁/窃锁/暂停/shadow 审计日志
```

会话记录字段：

```json
{
  "session_id": "sess_1770_ab12cd",
  "kind": "interactive | headless | sweep | ci",
  "pid": 12345,
  "start_cwd": "/tmp/ram-work",
  "started_at": "2026-08-06 00:00:00",
  "heartbeat_at": "2026-08-06 00:01:30",
  "status": "active | paused | borrowing | done | dead",
  "locks": ["db"],
  "holding_wait": ["taskAuth"],
  "note": "夜间自愈 fix #xxx"
}
```

### 5.2 仓库级锁（租约 + 心跳 + 窃取）

**获取** `acquire <repo>`（经 `locks/.ctl` 控制文件的 `flock` 序列化，保证原子）：

| 状态 | 判定 | 结果 |
|------|------|------|
| FREE | 无锁文件 | 原子创建 → 返回 `ACQUIRED` |
| HELD·新鲜 | 持有者心跳 ≤ 10min 前 | 返回 `HELD_BY <session_id>`（调用方走 5.3 升级路径） |
| HELD·过期 | 心跳 > 10min 且持有者 pid 不存活 | **窃取**（记录 audit，旧持有者恢复时被告知） |
| HELD·过期·存活 | 心跳 > 10min 但 pid 存活 | 不窃取，返回 `HELD_BY`（心跳守护未跑属异常，通知） |

**心跳**：后台守护进程 `claude-agent session heartbeat --pid <pid>`（SessionStart 时拉起，SessionEnd 停止），每 30s 刷新 registry 心跳 + 校验持有者 pid 存活；pid 死亡 → 标记 `dead` → 锁进入可回收状态。事件兜底：`UserPromptSubmit`/`PreCompact` hook 同步刷新。

**释放**：会话结束（SessionEnd/Stop hook、`claude-agent run` 收尾）→ `release-all`。

### 5.3 互锁与死锁

- **预防（规范加锁序）**：需要多仓锁时按 repo 路径**字典序**依次获取 → 无环等待，理论不死锁
- **检测**：`acquire` 等待超时（默认 120s）后查询持有者的 `holding_wait` 集合：若 ∩ 我方已持有 ≠ ∅ → 判定死锁，告警输出锁依赖环
- **升级路径（冲突处理策略，按会话类型）**：

```
冲突(HELD_BY)
 ├─ 交互会话 → 警告 holder 信息 → 用户决策: [等待 / 暂停对方 / shadow edit / 放弃]
 ├─ 无头 claude-agent → 等待 ≤3min → 死锁检测 → (配置允许)暂停对方 → shadow edit → 或失败重试/跳过
 ├─ 夜间巡检 → 跳过该仓（记录原因），其他仓继续
 └─ CI/pre-commit → 阻断提交，提示 holder
```

### 5.4 暂停 / 恢复（必要情况下暂停其他会话）

- `pause <session_id>`：注册表标记 `paused` + 对持有者 **pid 发送 SIGSTOP**（默认单进程；`PauseSessionGroup` 提供进程组变体）；恢复 `resume <session_id>` 发 SIGCONT
- **实现注**（2026-08-06 实施时修正）：默认单进程 SIGSTOP 而非整个进程组 — 交互会话与终端共享进程组，组信号会波及 shell；单进程已足以冻结 agent 主循环
- 暂停期间**锁保持持有**（防止他方窃取，保证恢复后一致性）
- **借编辑**：持有者冻结后，借用方可直接在仓库编辑，注册表记 `borrowing` 标记，完成后 `resume`
- 边界与风险：冻结可能落在写操作中途 → 策略上**优先 shadow edit**（5.5），SIGSTOP 仅用于「对方挂起空闲/长思考」场景；交互会话在暂停前通过 heartbeat 守护检查「无进行中写操作」标志；同主机会话才可暂停（本设计范围即同主机）

### 5.5 Shadow Edit（拷贝到临时目录编辑 → 原子拷回）

用户指定的必要场景：**把文件拷贝到其他临时目录中编辑完，再拷贝回来**。

```
shadow-begin <repo> <path...>
  → 快照拷贝至 logs/sessions/shadow/<sid>/<repo>/<path>
  → 记录原文件 SHA-256 到 manifest.json
  → 返回 shadow 路径（编辑目标即 shadow 路径）

（agent 在 shadow 区完成编辑）

shadow-apply
  → 比对原文件 SHA-256：
     未变 → 原子拷回（先写 .<name>.shadow 再 mv）→ 审计 → 成功
     已变 → 报告冲突（差异清单）→ 用户决策 [强制覆盖(留 .bak) / 放弃]
shadow-abort → 丢弃快照，清理
```

- shadow-begin **无需获取目标锁**（记录 shadow 意向即可）— 解决「持锁方无法暂停（如 CI）」场景
- 与 5.4 配合：无头会话默认路径 = 等待 → 暂停对方 → 直接编辑；对方不可暂停（CI/未知 pid）→ shadow edit

### 5.6 Claude Code hooks 接线（分发复用 v13）

`.claude/settings.json`（meta 仓 + 38 子仓，随 v13 分发器扩展分发；`scripts/lib/sessionctl.sh` 为 bash 薄封装，hook 内调用 Go CLI）：

| Hook | 动作 |
|------|------|
| `SessionStart` | `sessionctl register`（注册 + 拉起 heartbeat 守护 + 输出当前活跃会话摘要——互知的直接体现） |
| `SessionEnd` / `Stop` | `sessionctl release-all` + 停止 heartbeat |
| `UserPromptSubmit` | heartbeat 刷新（事件兜底） |
| `PreCompact` | heartbeat 刷新 |
| `PreToolUse` (Write/Edit) | `sessionctl check <repo>`：他会话持锁 → 交互输出警告；无头按策略返回 |

### 5.7 CLI 与工具

- **`claude-agent session <register|list|acquire|release|heartbeat|pause|resume|shadow-*>`**（Go，逻辑 SSOT）
- `scripts/lib/sessionctl.sh` — bash 薄封装（hooks / nightly sweep / CI 调用）
- `nightly_test_sweep.py` — 接入：巡检前 `list` 活跃会话 → 有活跃会话的仓跳过（替代/增强现有 dirty-skip）
- `.githooks/pre-commit` v1.1.0 — 提交前 `check` 锁：他会话持锁 → 阻断 + 提示 holder 信息
- 与 v65 `flock` 分层：flock = 进程级互斥（单巡检）；Session Hub = 会话级互知（跨会话类型），两层叠加

### 5.8 会话类型接入矩阵

| 会话类型 | 注册 | 锁 | 冲突策略 |
|---------|------|----|---------|
| 交互式 claude | SessionStart hook | 编辑/提交时 check | 警告 + 用户决策（等待/暂停对方/shadow/放弃） |
| 无头 claude-agent run | `run` 入口内建 | 目标仓全锁（字典序） | 等 ≤3min → 死锁检测 → 暂停对方 → shadow → 失败重试/跳过 |
| 夜间巡检 | 入口注册 sweep 会话 | 抽查仓 | 有活跃会话的仓跳过（记录原因） |
| CI / pre-commit | 不注册长驻 | 提交时校验 | 持锁 → 阻断提交并提示 |

### 5.9 迁移计划（分期）

- **P0** ✅（2026-08-06 完成）：`claude-agent session` 子命令（register/list/acquire/release/heartbeat/check/precheck/pause/resume/shadow-*）+ 注册表/锁/心跳/死锁检测 + Go 单测（26 项全绿）+ CLI 冒烟通过
- **P1** ✅（meta root 完成；38 仓分发见 OPT-20260806-001）：Claude Code hooks 接线（SessionStart/End/Stop/UserPromptSubmit/PreCompact/PreToolUse）+ settings.json + 交互会话互知摘要 + precheck 编辑前警告
- **P2** ✅（完成）：nightly sweep 接入（注册 sweep 会话 + 跳过活跃会话仓）+ pre-commit v1.1.0 提交锁校验（HELD_BY_OTHER 阻断 / HELD_BY_SELF 放行验证通过）+ check-hooks 扩展
- **P3** ✅（完成）：pause/resume + shadow edit 完整链路（含 audit、校验和冲突报告、原子拷回、.bak 备份）
- **P4** ✅（完成）：设计文档 + 双会话演练 + 架构交付（v14/v66 三类伴生文件 + VERSION_HISTORY + Archi 验证）
- **待办**：38 仓 settings.json/锁校验部署（OPT-20260806-001/002）、heartbeat 守护进程（OPT-20260806-003）

## 6. 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|--------|--------|--------------|---------|
| — | — | — | — | **无对应事件**：本设计为纯本地开发工具链（会话注册表/锁/心跳/影子编辑），不改变任何系统事实（无服务状态变更）、不触发跨边界副作用、无用户可感知行为；仅影响开发者主机上的多会话协调。符合「纯前端无服务端状态变更」例外类别（与 v13 git-hooks 同例） |

## 7. 🐍 Python 新增接口清单与 Go 替代评估

**未触发**：本设计不新增任何 HTTP 接口（`claude-agent session` 为 Go CLI 子命令，hooks 为 bash 薄封装调用 Go 二进制，nightly sweep 为既有 Python 编排脚本的调用侧接入，均非服务端点）。不进入 Python 接口专项审批门。

## 8. Value Stream 影响

**无影响**：`conf/value-stream.yaml` 各 stream（用户与认证 / 组织与成员 / 云平台与资源 / 项目与工作空间 / 任务协作）均以 `<service>.<table>.<field>` 为字段维度，本设计零服务/表/字段变更，不新增 stream、不改变任何 stream 状态。按「纯开发工具链变更」跳过价值流映射。

## 9. 🏛️ 架构变更影响

- **迭代版本**: v14 (enterprise-landscape) / v66 (application-integration) 🎯 target
- **迭代名称**: 多智能体会话互知与冲突防护（agent-session-coordination）
- **作者**: claude
- **设计日期**: 2026-08-06 00:39
- **变更视图**: 两个视图均变更（enterprise-landscape base = v13 ✅ current 已交付；application-integration base = v65 🎯 target 待交付，本次在其上叠加；v66 交付时与 v65 一并切换 current）
- **新增文件**（每个视图三类伴生格式，缺一不可）:
  - 🆕 `docs/architecture/v14-enterprise-landscape-<ts>-claude.puml`
  - 🆕 `docs/architecture/v14-enterprise-landscape-<ts>-claude.archimate`（含 Plateau v13→v14 架构变迁视图 + sourceConnection 连线）
  - 🆕 `docs/architecture/v14-enterprise-landscape-<ts>-claude.mermaid.md`
  - 🆕 `docs/architecture/v66-application-integration-<ts>-claude.puml`
  - 🆕 `docs/architecture/v66-application-integration-<ts>-claude.archimate`（含 Plateau v65→v66 架构变迁视图 + sourceConnection 连线）
  - 🆕 `docs/architecture/v66-application-integration-<ts>-claude.mermaid.md`
- **已有文件（未修改）**: `v13-enterprise-landscape-*.puml`、`v65-application-integration-*.puml`（target）、`v12`/`v63`（current）及其伴生文件
- **变更明细 (enterprise-landscape v14, vs v13)**:
  - 🟢 [NEW] `Session Hub`（Technology_Artifact — logs/sessions/ 注册表 + 锁/租约 + 心跳 + audit）
  - 🟢 [NEW] `claude-agent session 子命令`（Technology_Artifact — register/list/acquire/release/heartbeat/pause/resume/shadow-*）
  - 🟢 [NEW] `Claude Code 会话钩子`（Technology_Artifact — SessionStart/End/PreToolUse 等，settings.json 分发）
  - 🟢 [NEW] `Shadow Edit 工作区`（Technology_Artifact — logs/sessions/shadow/ + manifest）
  - 🟡 [MODIFIED] claude-agent — 新增 session/lock 子命令 + 心跳守护
  - 🟡 [MODIFIED] Git Hooks 分发器 — 扩展分发 `.claude/settings.json` 会话钩子 + pre-commit v1.1.0 锁校验
- **变更明细 (application-integration v66, vs v65)**:
  - 🟢 [NEW] `Session Hub` 组件（注册表/锁/心跳，接入 40+ 子仓）
  - 🟡 [MODIFIED] claude-agent — 无头 run 内建注册 + 目标仓加锁（字典序）+ 冲突升级路径
  - 🟡 [MODIFIED] nightly-sweep — 巡检前查活跃会话，有活跃会话的仓跳过
  - 🟡 [MODIFIED] 子仓 pre-commit — 提交锁校验（他会话持锁 → 阻断）

### .archimate 架构变迁要点

| 元素类型 | 内容 |
|----------|------|
| **Plateau v13/v65** | 基线 — git hooks 入库 + 夜间巡检自愈（flock 单进程互斥） |
| **Plateau v14/v66** | Target — Session Hub 跨会话互知 + 仓库级锁/心跳 + 暂停/影子编辑 |
| **Gap** | 多会话并发编辑无互知、无锁、冲突只能事后处理 |
| **WorkPackage** | WP-session-coordination — P0 CLI/注册表/锁 → P1 hooks 分发 → P2 sweep/CI 接入 → P3 pause/shadow → P4 交付 |
| **视图** | `架构变迁 v13→v14 — 多会话互知` / `v65→v66 — 多会话互知`（可导入 Archi；含 sourceConnection 连线） |

## 10. 风险与缓解

| 风险 | 缓解 |
|------|------|
| R1: SIGSTOP 可能冻结会话于写操作中途 | 策略上优先 shadow edit（非暂停）；暂停前 heartbeat 守护检查「无进行中写操作」；恢复后 git status 双会话确认 |
| R2: 锁文件/注册表漂移、僵尸锁 | 锁有效性以持有者 pid 存活 + 心跳新鲜双重判定；孤儿锁自动回收 + audit 留痕 |
| R3: 38 仓 settings.json hooks 分发维护成本 | 复用 v13 分发器与 HOOK_VERSION 版本戳（扩展 session-hooks 版本）；幂等增量 |
| R4: 交互会话 hook 输出噪音 | 仅冲突/会话摘要时输出；正常无输出 |
| R5: 与 v65 flock 职责重叠 | 明确定层：flock=进程级（单巡检），Session Hub=会话级（跨类型）；两层叠加不互斥 |
| R6: 无头会话自动暂停他人风险 | headless 默认仅等待/跳过/失败重试；`pause` 需配置显式开启（默认关）；shadow edit 为默认升级路径 |
| R7: hook 在子仓目录启动时找不到 meta root | 向上发现 `.gitmodules` 标记（或 SESSION_HUB_DIR 覆盖）；分发脚本写入绝对路径兜底 |

## 11. 验收清单

- [ ] P0: `claude-agent session register/list/acquire/release/heartbeat` 全链路可用；Go 单测覆盖（含死锁检测、心跳过期窃取）
- [ ] P0: 两会话互斥验证 — 会话 A 持锁后，会话 B `acquire` 返回 HELD_BY 并展示 holder 信息
- [ ] P1: 交互会话 SessionStart 自动注册、SessionEnd 自动释放；SessionStart 摘要展示当前活跃会话
- [ ] P1: 38 仓 hooks 分发幂等（HOOK_VERSION 扩展条目 + `--check` 全绿）
- [ ] P2: nightly sweep 跳过有活跃会话的仓（日志记录原因）；pre-commit v1.1.0 持锁提交被阻断
- [ ] P3: pause/resume + shadow edit 端到端（含校验和冲突报告 + 原子拷回 + audit）
- [ ] P4: 架构 v14/v66 三类伴生文件 + VERSION_HISTORY 条目 + Archi 加载验证通过
