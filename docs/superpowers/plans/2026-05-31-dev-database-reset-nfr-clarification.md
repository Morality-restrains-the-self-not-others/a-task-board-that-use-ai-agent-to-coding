# NFR 澄清: 开发环境一键重置全部数据库

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-05-31-dev-database-reset-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-05-31-dev-database-reset-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L1 | 全量重置 < 5 分钟（本地，含 Django migrate） |
| 可伸缩性 | L0 | 不适用（单机 dev） |
| 可用性 | L0 | 重置本身不要求高可用；操作后服务故意停机 |
| 安全性 | L3 | 仅 localhost 或 `RUNALL_ALLOW_DEV_DB_RESET=1`；须 `confirm=RESET_ALL` |
| 数据一致性 | L2 | 停服后删库；migrate+init 顺序固定；fail-fast |
| 容错机制 | L2 | 分步 `partial` 回报；Redis/Kafka 失败则中止 migrate |
| 可观测性 | L1 | API JSON 分步状态；无专项指标 |
| 合规与隐私 | L0 | 仅本地 dev |
| 可维护性 | L2 | registry 驱动脚本路径，与 observability reset 同模式 |

## 逐增量 NFR 分析

### Increment 1–4（合并：本地 dev 工具）

#### 安全性 — L3
- **量化:** 无确认 → 400；非本地且无 env → 403；并发重置 → 409
- **质量场景:** QS-01

#### 数据一致性 — L2
- **量化:** `order` 升序；单库 migrate 失败不执行该库 init 及后续库
- **质量场景:** QS-02

#### 容错机制 — L2
- **量化:** 任一步失败 → `status: partial` + 明细数组
- **质量场景:** QS-03

## 质量场景

### QS-01: 未授权调用被拒绝
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | 远程客户端 |
| 刺激 | `POST /api/dev/reset-databases` 无 confirm 或 Host 非 localhost |
| 制品 | runAll API |
| 环境 | 未设置 `RUNALL_ALLOW_DEV_DB_RESET` |
| 响应 | 400 或 403，不删任何文件 |
| 响应度量 | 集成测试断言 HTTP 状态与 `db/saas/*.sqlite3` mtime 不变 |

### QS-02: 有序 migrate 后 init
| 要素 | 内容 |
|------|------|
| 类别 | 数据一致性 |
| 等级 | L2 |
| 刺激源 | 开发者确认重置 |
| 刺激 | saas migrate 成功、task-auth migrate 失败 |
| 制品 | DatabasePlatformResetService |
| 环境 | 正常 |
| 响应 | task-auth init 不执行；`migrations` 含 failed |
| 响应度量 | 单测 mock 脚本退出码 |

### QS-03: Redis 失败中止后续
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L2 |
| 刺激源 | docker-redis 未启动 |
| 刺激 | `redis-flush.sh` 非零退出 |
| 制品 | 重置流水线 |
| 环境 | 降级 |
| 响应 | `redis_reset: failed`，不执行任何 migrate_script |
| 响应度量 | 单测 + 手工：无新 schema 表出现 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| L3 安全 | 重置为显式命令，须 ConfirmToken 值对象 | `ConfirmToken` VO；API 层校验 |
| L2 fail-fast | 流水线状态机，非大聚合 | `DatabasePlatformReset` 编排服务 + 分步 `StepResult` |
| L2 顺序 | RegisteredDatabase 按 order 排序 | 实体列表排序后依次执行 |
| 不自动启服 | 无 StartServices 领域行为 | 编排服务不包含启动端口 |

## 权衡与边界

### 取舍
- 安全 L3 换取误操作风险趋零（仅 dev 可用）
- Redis/Kafka 失败即中止 migrate，避免「空库 + 脏缓存」不一致

### 明确不做什么
- 不自动 `start_command`
- 不跑 `init-tenant`
- 不清 Grafana volumes

### 升级触发条件
- 若需 CI 夹具 → 新增独立 stream + 临时目录策略（L2 安全降级为专用 token）

## 跳过声明
- 可伸缩性、可用性、合规：本地单机 dev 工具，跳过。
