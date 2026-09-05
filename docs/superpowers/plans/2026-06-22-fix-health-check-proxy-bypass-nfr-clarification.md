# NFR 澄清: Health Check Proxy Bypass Fix

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-06-22-fix-health-check-proxy-bypass-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-22-fix-health-check-proxy-bypass-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 容错机制 | L2 | Health check 在代理环境变量存在时正常工作 |
| 性能 | L0 — 不适用 | 无性能变化（仅替换 HTTP client 实现） |
| 可伸缩性 | L0 — 不适用 | 无伸缩性变化 |
| 可用性 | L0 — 不适用 | 无可用性变化 |
| 安全性 | L0 — 不适用 | 无安全模型变更 |
| 数据一致性 | L0 — 不适用 | 无数据流变更 |
| 可观测性 | L0 — 不适用 | 无观测性变化 |
| 合规与隐私 | L0 — 不适用 | 无合规影响 |
| 可维护性 | L0 — 不适用 | 无 API/接口变更 |

## 逐增量 NFR 分析

### Increment 1: Proxy-Free Health HTTP Client (唯一增量)

#### NFR 类别: 容错机制
- **等级**: L2 - 标准
- **量化目标**: runAll health check 在所有常见代理环境变量配置下正常完成（`ALL_PROXY`, `HTTP_PROXY`, `HTTPS_PROXY` 设置时连接 127.0.0.1 / 0.0.0.0 / LAN 地址均直连）
- **质量场景**: QS-01

## 质量场景

### QS-01: Health check bypasses proxy
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L2 |
| 刺激源 | 操作系统环境变量 |
| 刺激 | `ALL_PROXY=socks5h://127.0.0.1:1234` 已设置 |
| 制品 | runAll `checkHealth()` 函数 |
| 环境 | runAll 启动 task-auth 后的 readiness 探测阶段 |
| 响应 | HTTP GET 直连 `http://127.0.0.1:8003/api/health/`，不经过 SOCKS 代理 |
| 响应度量 | HTTP 请求在 5s 内返回 2xx/3xx；`netstat` / 抓包确认无 socks 连接至 127.0.0.1:1234 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 无 | 此修复仅改基础设施层 HTTP 客户端，不引入新领域概念 | DDD 步骤中无需新增实体、值对象或聚合 |

## 权衡与边界

### 取舍
- 无取舍。修复是严格改进：health check HTTP client 从代理感知变为代理无关。

### 明确不做什么
- 不修改全局 `http.DefaultTransport` 的代理行为（避免影响其他需要代理的出站请求）
- 不修改 `NO_PROXY` 环境变量（环境变量属于用户配置，不应被程序隐式修改）
- 不修改 `conf/auth/task-auth/config.yaml` 中的 `host: 0.0.0.0`（该字段语义为 bind address，修改会破坏服务监听行为）

### 升级触发条件
- 如果未来 runAll 需要管理外部（公网）服务且 health check 需走代理 → 需将 healthHTTPClient 改为可配置（按服务按需启用代理），当前无此需求

## 跳过声明
- **性能**: 跳过。`http.DefaultClient` 与自定义 `http.Client{Proxy: nil}` 性能等价。
- **可伸缩性**: 跳过。无伸缩性变化。
- **可用性**: 跳过。此修复提升可靠性（修复 bug），但本身不引入可用性 SLO。
- **安全性**: 跳过。不改变认证/授权/加密模型。
- **数据一致性**: 跳过。不操作业务数据。
- **可观测性**: 跳过。不改变日志/指标/追踪。
- **合规与隐私**: 跳过。无合规影响。
- **可维护性**: 跳过。不改变 API 或配置格式。
