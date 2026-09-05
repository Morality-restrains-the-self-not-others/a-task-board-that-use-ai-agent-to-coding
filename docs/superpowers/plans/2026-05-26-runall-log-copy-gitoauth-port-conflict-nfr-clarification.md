# NFR 澄清: runAll 日志复制与 git-oauth 端口冲突恢复启动

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-05-25-runall-stability-first-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-05-26-runall-log-copy-gitoauth-port-conflict-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L1 | 日志复制 UI 反馈 ≤ 300ms；端口 preflight 清理 P95 ≤ 3s |
| 可用性 | L2 | UI 手动启动 `git-oauth` 在端口被外部进程占用时，一次点击内恢复成功率 ≥ 95% |
| 容错机制 | L2 | 端口冲突在 Launch 前完成识别与清理；清理失败 3s 内返回结构化错误 |
| 可观测性 | L2 | 复制动作与端口冲突均可在 UI/日志中给出可执行提示（含 PID/端口） |
| 安全性 | L2 | 仅清理「非 runAll 受管」监听进程；不扩大 kill 范围 |
| 一致性 | L2 | UI 手动启动与批量启动共用同一 preflight 端口治理语义 |
| 可维护性 | L2 | 新增 UI 行为与 preflight 路径均有自动化测试 |
| 可伸缩性 | L0 | 不适用（本地单用户编排 UI） |
| 数据一致性 | L0 | 不适用（无持久化数据模型变更） |
| 合规与隐私 | L0 | 不适用（日志复制仅在本地浏览器剪贴板） |

## 逐增量 NFR 分析

### Increment 1: 日志复制最小闭环 (Thin Slice)

#### NFR 类别: 性能
- **等级**: L1 - 基础
- **量化目标**: 点击「复制日志」到 UI 给出成功/失败反馈 ≤ 300ms（不含用户粘贴操作）
- **约束**: 仅复制当前面板已渲染文本（最近 200 行），不额外请求后端

#### NFR 类别: 可观测性
- **等级**: L1 - 基础
- **量化目标**: 复制失败时 100% 展示可读原因（权限拒绝 / 空日志 / API 异常）
- **约束**: 不引入新的后端审计接口

#### NFR 类别: 安全性
- **等级**: L1 - 基础
- **量化目标**: 复制内容仅限当前服务日志缓冲区，不读取磁盘文件
- **约束**: 不自动复制跨服务日志

#### NFR 类别: 可维护性
- **等级**: L2 - 标准
- **量化目标**: `status.html` 片段测试覆盖复制按钮 hook 与 clipboard 失败回退路径
- **约束**: 不改变 `/api/logs` 契约

---

### Increment 2: git-oauth 端口冲突可诊断化

#### NFR 类别: 容错机制
- **等级**: L2 - 标准
- **量化目标**:
  - UI 手动启动路径在 Launch 前执行端口 preflight（与批量启动一致）
  - 冲突检测 P95 ≤ 1s；失败返回 `PRECHECK_PORT_CONFLICT` + 端口 + PID 列表
- **约束**: 检测失败时不进入 `python3` WSGI 绑定，避免仅 stderr traceback 可见

#### NFR 类别: 可观测性
- **等级**: L2 - 标准
- **量化目标**:
  - 状态页 `error` / `hint` 字段包含端口与 PID
  - 日志面板可复制完整诊断文本（依赖 Increment 1）
- **约束**: 错误文案必须可执行（例如提示「关闭占用进程后重试」或「将自动清理并重试」）

#### NFR 类别: 一致性
- **等级**: L2 - 标准
- **量化目标**: UI `POST /api/start` 与 orchestrator 批量启动对 `git-oauth:8002` 使用同一 preflight 规则
- **约束**: 禁止「批量启动有 preflight、手动启动无 preflight」的双轨行为

**根因说明（实现约束）**: 当前 `startService()` 直接调用 `startAndCheck()`，未走 `runPreflight()`，导致 UI 点击启动时出现 `OSError: [Errno 48] Address already in use`。Increment 2 必须闭合此路径。

---

### Increment 3: git-oauth 端口冲突自动恢复

#### NFR 类别: 可用性
- **等级**: L2 - 标准
- **量化目标**:
  - 在「端口被外部进程占用且非 runAll owned PID」场景下，单次 UI 启动成功率 ≥ 95%
  - 自动恢复后 `git-oauth` 健康检查 `http://127.0.0.1:8002/api/health/` 通过
- **约束**: 恢复失败时必须保留 Increment 2 的结构化诊断，不得静默失败

#### NFR 类别: 容错机制
- **等级**: L2 - 标准
- **量化目标**:
  - 清理流程：SIGTERM → 250ms → SIGKILL（与现有 `preflight.go` 一致）
  - 全流程（检测 + 清理 + 重扫 + Launch）P95 ≤ 5s
- **约束**: 清理后端口仍被占用则 fail-fast，不无限重试

#### NFR 类别: 安全性
- **等级**: L2 - 标准
- **量化目标**:
  - 仅终止 `filterForeignPIDs()` 判定为外部的监听进程
  - 不终止 runAll 当前 session 已登记 owned PID
- **约束**:
  - 不对「同端口但不同服务名」做全局 kill-all
  - 清理动作写入服务日志（含 PID），便于 Increment 1 复制审计

#### NFR 类别: 一致性
- **等级**: L2 - 标准
- **量化目标**: 自动清理语义与 `2026-05-25-runall-stability-first` 中 preflight 外来进程清理一致
- **约束**: 本增量是「补齐手动启动路径」，不是引入新的 takeover 模式；显式 `takeover` 命令语义保持不变

---

### Increment 4: 交互反馈增强

#### NFR 类别: 可观测性
- **等级**: L2 - 标准
- **量化目标**:
  - 复制成功/失败在日志面板 meta 区 2s 内可见
  - 端口冲突自动恢复后在服务行或 hint 区展示「已清理 PID xxx 并重试」或「需手工处理」
- **约束**: 反馈为纯前端文案，不新增后端 API

#### NFR 类别: 性能
- **等级**: L1 - 基础
- **量化目标**: 反馈动画/文案切换不阻塞 2s 状态轮询
- **约束**: 复用现有 `pulseClickFeedback` 模式，不引入 loading 禁用整页

#### NFR 类别: 可维护性
- **等级**: L2 - 标准
- **量化目标**: UI 测试覆盖复制反馈与冲突 hint 渲染分支
- **约束**: 不改变 `/api/status` 字段契约（可消费已有 `hint` / `error`）

## 质量场景 (Quality Attribute Scenarios)

### QS-01: 日志一键复制成功
| 要素 | 内容 |
|------|------|
| 类别 | 性能 / 可观测性 |
| 等级 | L1 |
| 刺激源 | 开发者在 runAll 状态页打开某服务日志面板 |
| 刺激 | 点击「复制日志」，面板已有 ≥ 1 行日志 |
| 制品 | `status.html` 日志面板 + Clipboard API |
| 环境 | 本地 Chrome/Safari，HTTPS 或 localhost |
| 响应 | 当前可见日志全文写入系统剪贴板，面板 meta 显示「已复制」 |
| 响应度量 | 点击到反馈文案出现 ≤ 300ms；粘贴内容与面板文本一致（逐行） |

### QS-02: 日志复制失败可感知
| 要素 | 内容 |
|------|------|
| 类别 | 可观测性 |
| 等级 | L1 |
| 刺激源 | 浏览器拒绝剪贴板权限或面板为空 |
| 刺激 | 点击「复制日志」 |
| 制品 | 日志面板复制 handler |
| 环境 | 权限受限或空日志 |
| 响应 | 不抛未捕获异常；展示「复制失败：原因」 |
| 响应度量 | 100% 有可见错误提示；页面其他按钮仍可用 |

### QS-03: UI 手动启动 git-oauth 端口冲突可诊断
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 / 一致性 |
| 等级 | L2 |
| 刺激源 | 开发者通过 UI 启动已 stopped 的 `git-oauth` |
| 刺激 | 8002 端口被外部 python 进程占用（非 runAll owned） |
| 制品 | `POST /api/start` → `startService` → preflight |
| 环境 | 本地开发，runAll 状态页 `:9999` |
| 响应 | 不输出仅 stderr traceback；状态页返回 `PRECHECK_PORT_CONFLICT` + port=8002 + pid 列表 + hint |
| 响应度量 | 从点击启动到 failed 状态可见 ≤ 3s；日志可复制完整诊断（QS-01） |

### QS-04: UI 手动启动 git-oauth 自动清理并重试成功
| 要素 | 内容 |
|------|------|
| 类别 | 可用性 / 容错机制 |
| 等级 | L2 |
| 刺激源 | 开发者点击 UI「启动」git-oauth |
| 刺激 | 8002 被可终止的外部监听进程占用 |
| 制品 | preflight 端口清理 + `gitOauth/run.sh` Launch + health check |
| 环境 | 本地开发，单次操作 |
| 响应 | 自动终止外部 PID → 重新绑定 8002 → 健康检查通过 → 状态 `healthy` |
| 响应度量 | 单次点击内成功率 ≥ 95%（可重复测试 20 次）；全流程 P95 ≤ 5s |

### QS-05: 端口清理安全边界
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 / 一致性 |
| 等级 | L2 |
| 刺激源 | runAll 已受管进程占用目标端口 |
| 刺激 | UI 启动同名或其他服务导致端口检测 |
| 制品 | `filterForeignPIDs` + ownership 集合 |
| 环境 | runAll session 内已有 owned PID |
| 响应 | 不 kill owned PID；若仍冲突则 fail-fast 并提示 owner/session |
| 响应度量 | owned PID 误杀率 0%；错误信息含 session 提示 |

### QS-06: 自动恢复失败时的可操作反馈
| 要素 | 内容 |
|------|------|
| 类别 | 可观测性 / 可用性 |
| 等级 | L2 |
| 刺激源 | 外部进程无法被 SIGTERM/SIGKILL 终止（权限不足） |
| 刺激 | UI 启动 git-oauth |
| 制品 | preflight 清理 + 状态页 hint |
| 环境 | 本地 macOS，占用进程为 root 或系统服务 |
| 响应 | 保持 `failed`；hint 给出「需手工 sudo kill <pid>」或等价指引 |
| 响应度量 | 60s 内用户可凭 hint + 复制日志完成定位；无 hung `starting` 状态 |

## 领域模型影响

| NFR 决策 | 领域模型影响 | 对应 DDD 动作 |
|----------|-------------|-------------|
| 一致性 L2（手动/批量 preflight 统一） | 启动命令必须经同一 Preflight 领域服务，而非 UI 专用捷径 | 抽取 `ServicePreflightService`，`startService` 与批量启动均调用 |
| 容错 L2（端口冲突结构化） | 失败需建模为带端口/PID 的值对象，而非自由文本 | 扩展 `PortConflictSnapshot`（port, foreign_pids, cleanup_result） |
| 安全 L2（仅清理 foreign PID） | 清理决策依赖 ownership 聚合状态 | `ServiceOwnership` 参与 preflight 决策；清理为领域服务方法 `ResolvePortConflict()` |
| 可观测性 L2（可复制诊断） | UI 操作与后端失败事件需共享同一 message 格式 | 日志行与 `FailureHint` 使用统一模板渲染 |
| 可用性 L2（自动恢复） | 启动会话需记录「清理尝试」事件 | 新增 `PortConflictCleanupAttempted` / `PortConflictCleanupSucceeded` 领域事件（内存态即可） |
| 性能 L1（复制纯前端） | 无后端聚合变更 | DDD 范围限定在 orchestration-lifecycle + service-runtime-observability 上下文 |

## 权衡与边界

### 取舍
- **可用性 vs 安全**：允许对「非 owned 外部监听进程」自动 SIGTERM/SIGKILL，换取本地开发一键恢复；不做无差别端口扫描。
- **一致性 vs 改动面**：优先复用现有 `preflight.go` 逻辑，而非为 `git-oauth` 单独写特殊分支。
- **可观测性 vs 复杂度**：复制功能纯前端实现，不新增 `/api/logs/export`，降低后端面。

### 明确不做什么
- 不实现跨服务批量复制日志、不导出到文件。
- 不修改 `gitOauth/run.sh` 的 WSGI 实现（除非 preflight 仍不足）。
- 不改变显式 `takeover` CLI 语义；UI 自动清理仅覆盖「foreign listener」子集。
- 不追求 L3+ 多用户并发编排可用性（仍为本地单开发者场景）。
- 不为剪贴板失败实现服务端 fallback 下载接口（V1 仅 UI 提示）。

### 升级触发条件
- 若自动清理误杀非目标进程 ≥ 1 次/周 → 安全从 L2 升到 L3，增加进程 cmdline 校验与白名单。
- 若 UI 启动成功率 < 90% → 可用性升级，增加「清理后二次 preflight + 显式重试按钮」。
- 若复制失败率 > 5%（非 Safari 权限场景）→ 可维护性升级，增加 `document.execCommand('copy')` 或 textarea 回退。
- 若 preflight 全流程 P95 > 5s → 性能升级到 L2，优化 `lsof` 调用频率或缓存端口快照。

## 跳过声明

- **可伸缩性 (L0)**：本地 runAll UI，单实例单用户，无水平扩展需求。
- **数据一致性 (L0)**：日志与冲突状态均为内存/runtime 视图，无跨服务事务。
- **合规与隐私 (L0)**：剪贴板操作不离开用户机器；不新增 PII 存储。
- **性能 L3/L4**：非核心卖点；复制与 preflight 保持 L1-L2 即可。
- **可用性 L3/L4**：全平台编排 SLA 已在 `runall-stability-first` 单独澄清；本增量仅聚焦 UI 手动启动 git-oauth 路径。

## 自检 (Hard Gate)

- [x] 每个相关 NFR 类别都有明确支撑等级（L0-L4）
- [x] 每个 L1-L4 类别具备量化目标
- [x] 每个 L2-L4 类别至少一个质量场景（QS-03~06 覆盖 L2）
- [x] 每个场景有可验证响应度量
- [x] 影响领域模型的 NFR 决策已标注且对应 DDD 动作
- [x] 权衡与边界明确（包含不做事项与升级触发）
- [x] 跳过类别给出理由
- [x] 文档路径正确（`docs/superpowers/plans/2026-05-26-runall-log-copy-gitoauth-port-conflict-nfr-clarification.md`）

---

NFR 澄清完成。关键 NFR 决策如下：

1. **一致性 L2**：UI 手动启动必须与批量启动共用 preflight，闭合「Errno 48 仅出现在 stderr」的路径缺口。
2. **容错/可用性 L2**：对外部占用 8002 的进程允许受控自动清理并重试，单次点击成功率目标 ≥ 95%。
3. **安全 L2**：仅清理 `foreign` 监听 PID，owned 进程零误杀；清理动作可日志化并可复制。
4. **可观测性 L1-L2**：日志复制与冲突诊断均需在 UI 给出可执行反馈，支撑排障闭环。

这些决策将在 `/5-ddd-领域设计驱动` 中驱动 `ServicePreflightService`、`PortConflictSnapshot` 与启动路径统一建模。
