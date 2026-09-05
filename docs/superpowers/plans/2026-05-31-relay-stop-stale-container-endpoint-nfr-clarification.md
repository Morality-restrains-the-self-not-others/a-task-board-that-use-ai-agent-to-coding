# NFR 澄清: 容器停止 stale 端点作废

> 输入: design + value-stream (Increment 1 为主)

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 数据一致性 | L2 | stop 成功后当前行 reachability 与 SSE 在 1s 内一致为未注册 |
| 容错机制 | L2 | go_relay stop 失败时不误清 DB |
| 可观测性 | L2 | stop_reason + audit 事件可追踪 |
| 性能 | L1 | 同步 DB+SSE，无额外 RPC |
| 安全性 | L3 | 不清 refresh token；History/audit 不含明文 access token |

## 质量场景

### QS-01: relay stop 后层图不再转发 stale URL

| 要素 | 内容 |
|------|------|
| 刺激源 | 用户点 relay 停止 |
| 刺激 | go_relay `/v1/stop` 200 |
| 制品 | `relay_to_trae_stop` + `CloudServerConfig` |
| 响应 | `server_url==""`；SSE `container_endpoint_registered=false` |
| 响应度量 | pytest + Playwright mock |

### QS-02: stop 失败保留端点

| 要素 | 内容 |
|------|------|
| 刺激 | go_relay stop 5xx |
| 响应 | 不修改 `server_url` |

## 领域模型影响

| NFR | 模型影响 |
|-----|---------|
| L2 一致性 | `TaskServerRuntimeSession` 聚合；close 与 clear 同一事务边界 |
| L3 安全 | token 保留在 Config 当前行，History 仅 URL 快照 |

## 权衡

- Increment 1 不做 History migration（Increment 2）
- 不做 P99 优化
