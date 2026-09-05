# NFR 澄清 — 部署机 9999 源码编译重启

- **Date:** 2026-09-02

## 路径分片键审视

| 路径 | 分片 ID | 适配性 | 级别 | 动作 |
|------|---------|--------|------|------|
| `POST /api/precise-restart` | 无 tenantId | 本机单实例 ops UI | **L0** | 不补键。升级触发：9999 变成多租户/多部署根 SaaS 控制面时再设计 `deploy_id` |
| `GET /api/precise-restart/progress` SSE | 无 | 同上，按 `run_id` | **L0** | `run_id` 是操作关联键，非库表分片键 |
| `POST /api/build-all` | 无 | 同上 | **L0** | 同 precise-restart |
| `GET /api/build-all/progress` SSE | 无 | 同上 | **L0** | 同左 |
| 子进程 `precise-compile.sh` | 无 | 本机 `$SOURCE_ROOT` | **L0** | 禁止 SSH 远程编译（后续迭代） |

全部路径未带租户分片 ID；本增量明确 **无可伸缩性需求（L0）**：单一部署机、单一 `SOURCE_ROOT`。资金/配额路径不涉及。

## 幂等性审视

| 路径 | 副作用 | 重复触发 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|----------|--------------|--------|----------|
| `POST /api/precise-restart` | 编译、rsync `conf-local`、install ELF、重启进程 | 双击按钮；SSE 重试 | **一次 bulk 运行**（本批登记集合） | 前端 clickGuard + 服务端 `TryBeginPreciseRestart` / bulk 锁；`run_id` | 进行中 409；失败不安装、登记保留；成功 last-writer-wins 产物 |
| `POST /api/build-all` | `--all` 编译 + rsync + install；不重启 | 双击 | **一次全量编译运行** | 既有 build-all bulk 锁 | 失败不清登记、不覆盖产物 |
| `rsync --delete conf-local` | 覆盖机密树 | 编译成功后必跑 | 与源码树 `conf-local/` 对齐 | 挂在编译成功事务之后 | 编译失败 **不** rsync（T8b） |
| `install-local-artifacts` | 替换 ELF | 同上 | 单文件 size+mtime 跳过 | 文件内容/mtime | 未变不 `mv`；变化 copy-then-mv |
| Kafka / Webhook / timer | 无 | — | — | — | **L0 无副作用** |

资金/配额/云资源：不涉及（默认 ≥L3 不适用）。无新事件消费者，禁止用 `tenant_id` 当幂等键（本路径无租户键）。

## 其它

- **可用性：** 编译失败保留 last-good；不重启 runAll 编排器。
- **安全：** 日志不打印 `conf-local` 文件内容、token、PEM。
- **可观测性：** 复用既有 SSE；失败响应带既有 `data-traceId`。
