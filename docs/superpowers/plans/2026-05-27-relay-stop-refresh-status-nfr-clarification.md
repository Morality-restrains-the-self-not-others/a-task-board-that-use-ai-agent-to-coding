# NFR 澄清: relayToTrae 停止后刷新状态

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-05-27-relay-stop-refresh-status-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-05-27-relay-stop-refresh-status-value-stream.md`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | 刷新状态 UI 更新 ≤ 3s（含 health+status 串行） |
| 可用性 | L2 | stop 后 refresh 不因瞬态 health 失败误判 offline |
| 容错机制 | L2 | 并发 refresh 后发结果覆盖先发 stale 失败 |
| 可维护性 | L2 | 单文件前端修复，Playwright 回归 |
| 数据一致性 | L0 | 无服务端状态变更 |
| 安全性 | L0 | 无鉴权变更 |
| 可伸缩性 | L0 | 不适用 |

## 质量场景

### QS-01: stop 后 refresh 保持 relay 在线
| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L2 |
| 刺激源 | 用户点击「停止」后点击「刷新状态」 |
| 刺激 | health 偶发超时，status 返回 200 `{running:false}` |
| 制品 | `fetchRelayToTraeServiceStatus` |
| 环境 | 本机 relay 运行、onlineService 已停 |
| 响应 | `relayToTrae 服务：在线`，`onlineServiceJS：未启动` |
| 响应度量 | Playwright 断言通过 |

### QS-02: 并发 refresh 不被 stale 覆盖
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L2 |
| 刺激源 | stop 自动 refresh + 用户手动 refresh 重叠 |
| 刺激 | 先发请求 health 失败、后发请求成功 |
| 制品 | generation token 丢弃过期写入 |
| 环境 | 正常 |
| 响应 | 最终 UI 为 relay 在线 |
| 响应度量 | 逻辑单测或 Playwright 通过 |

## 领域模型影响

纯前端状态机修复，**不新增后端 domain 实体**。前端 VO：

- `RelayReachability` — health/status HTTP 可达性
- `OnlineServiceRuntime` — running / orphan / ui_url

## 权衡与边界

- 仅当 health 与 status HTTP 均失败才显示「未连接」
- 不改动 go_relay / Django stop 异步语义
- relay 真正未启动时仍正确显示 offline

## 跳过声明

- 后端 DDD 增量跳过：无持久化或 API 契约变更
