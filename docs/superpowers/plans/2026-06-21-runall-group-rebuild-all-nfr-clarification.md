# NFR 澄清: runAll 分组批量重新编译

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-06-21-runall-group-rebuild-all-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-21-runall-group-rebuild-all-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L1 | 无专项 — 编译操作本身耗时数十秒至数分钟，API 仅触发并立即返回 |
| 可伸缩性 | L0 | 不适用 — 本地开发工具，单用户 |
| 可用性 | L0 | 不适用 — 本地开发工具，非生产服务 |
| 安全性 | L0 | 不适用 — 无认证体系，同源访问 |
| 数据一致性 | L1 | 基础 — 复用已有 BuildService 的 CAS 状态转换 |
| 容错机制 | L2 | 标准 — best-effort 继续策略；单个失败不阻塞后续编译 |
| 可观测性 | L1 | 基础 — 复用已有 log.Printf + runner 日志体系 |
| 合规与隐私 | L0 | 不适用 — 纯工具软件 |
| 可维护性 | L1 | 基础 — 复用已有 BuildService，不引入新抽象 |

## 逐增量 NFR 分析

### Increment 1: 分组编译按钮 + API + 结果反馈 (Thin Slice)

#### NFR 类别: 容错机制
- **等级**: L2 - 标准
- **量化目标**: 组内任一服务编译失败时，其余独立服务继续进行；失败不产生副作用（状态回滚到编译前）
- **质量场景**: QS-01

#### NFR 类别: 性能
- **等级**: L1 - 基础
- **量化目标**: `/api/build-group` 请求响应时间 P95 < 100ms（仅指 API 响应时间，不含编译执行）。编译本身以同步轮询方式执行，时长因服务而异。
- **质量场景**: QS-02

#### NFR 类别: 可观测性
- **等级**: L1 - 基础
- **量化目标**: 每次 build-group 操作记录一条汇总日志，每个子 build 记录独立日志行
- **质量场景**: QS-03

## 质量场景

### QS-01: 分组编译部分失败时继续执行
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L2 |
| 刺激源 | 开发者在 runAll UI 点击 platform 组「全部重新编译」 |
| 刺激 | 组内包含 8 个可编译服务，其中 1 个编译失败（如依赖缺失） |
| 制品 | POST /api/build-group + Runner.BuildGroup |
| 环境 | 正常 — 部分服务 healthy，部分 failed |
| 响应 | 跳过失败服务，继续编译其余 7 个；API 返回 `{status: "partial", built: 7, failed: ["svc-x"]}` |
| 响应度量 | failed 列表非空时 status 为 "partial"；built + |failed| = total；每个 attempted 服务的编译日志独立可查 |

### QS-02: API 响应时间
| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L1 |
| 刺激源 | runAll UI JavaScript fetch 调用 |
| 刺激 | POST /api/build-group `{"group": "platform"}` |
| 制品 | `/api/build-group` 路由处理器 |
| 环境 | 正常 — runAll 进程本地运行 |
| 响应 | 返回 JSON `{status, total, built, failed, skipped, no_build}` |
| 响应度量 | 无编译任务触发时 P95 < 50ms；有编译任务时首次响应 P95 < 200ms（含状态 CAS + 首服务 build 启动） |

### QS-03: 编译操作可追踪
| 要素 | 内容 |
|------|------|
| 类别 | 可观测性 |
| 等级 | L1 |
| 刺激源 | 后台 log 查看 |
| 刺激 | build-group 执行完成（成功/部分/全失败） |
| 制品 | runAll stdout / 日志文件 |
| 环境 | 正常 |
| 响应 | 日志包含 `[build-group]` 标记行，列出组名、总数、成功数、各失败项 |
| 响应度量 | 从日志能还原每次 build-group 操作的完整结果 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 容错 L2（best-effort 继续） | BuildGroupResult 是独立值对象，不耦合到 Service 聚合生命周期 | DDD 步骤将 BuildGroupResult 建模为值对象，BuildGroup 作为领域服务 |
| 复用 BuildService | 无新聚合，无新一致性边界 | 不引入新实体或聚合，仅扩展现有 ServiceLifecycle 领域服务 |

## 权衡与边界

### 取舍
- 选择**串行编译**（按依赖深度排序）而非并行编译，牺牲速度换取安全（避免 go build 并发写 `$GOPATH/pkg` 冲突）

### 明确不做什么
- 不做编译进度实时推送（WebSocket/SSE），仅返回最终汇总
- 不做编译超时终止（由各子 build 自身的 context 超时控制）
- 不做跨组编译（一次只操作一个组）

### 升级触发条件
- 当 runAll 支持并行安全编译（如隔离的 build cache）时，可升级为并行编译（性能 L1 → L2）
- 当 UI 需要实时编译进度时，增加 SSE 推送（可观测性 L1 → L2）

## 跳过声明

- **可伸缩性**: 跳过。本地开发工具，单用户实例，无水平扩展需求。
- **可用性**: 跳过。非生产服务，无 uptime 保证需求。
- **安全性**: 跳过。本地同源工具，无认证体系，无敏感数据。
- **合规与隐私**: 跳过。纯开发者工具，不处理用户数据。
- **数据一致性**: 不单独设场景。完全复用已有 BuildService 的 CAS 状态转换，不引入新的一致性需求。
- **可维护性**: 不单独设场景。变更量小（~100 行代码），无新模块或新依赖引入。
