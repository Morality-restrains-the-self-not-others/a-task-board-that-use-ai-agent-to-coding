# NFR 澄清: gitOauth DisallowedHost 网关 Host 头校验修复

> 输入:
> - 设计文档: `docs/designs/disallowed-host-gateway-fix.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-26-disallowed-host-gateway-fix-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 安全性 | L2 | ALLOWED_HOSTS 白名单不包含通配符，仅放行网关明确配置的公网地址 |
| 可维护性 | L2 | 网关 IP 变更时只需修改 `conf/gateway/task-gateway/config.yaml` 的 `publicBase`，gitOauth 重启后自动生效 |
| 性能 | - | 跳过 — 无新增 API，无数据流变化，Host 校验是 Django request 生命周期中已有的固定开销 |
| 可用性 | - | 跳过 — 修复的是可用性 bug（400 错误），修复后即恢复可用，无新增可用性保证 |
| 数据一致性 | - | 跳过 — 纯配置变更，不引入状态变更或数据操作 |
| 容错机制 | - | 跳过 — 无新增外部依赖调用 |
| 可观测性 | - | 跳过 — 无新增日志/指标/追踪需求 |
| 合规与隐私 | - | 跳过 — 不涉及 |

## 逐增量 NFR 分析

### Increment 1: 网关公网 Host 加入 ALLOWED_HOSTS

#### NFR 类别: 安全性
- **等级**: L2 - 标准
- **量化目标**: 
  - ALLOWED_HOSTS 始终包含网关 `publicBase` 的 hostname
  - 不使用通配符 `*`
  - 不因 Host 头注入而路由到非预期后端
- **设计约束**: 仅从受信任的本地配置文件 (`conf/gateway/task-gateway/config.yaml`) 读取，不接受请求头中的动态 Host

#### NFR 类别: 可维护性
- **等级**: L2 - 标准
- **量化目标**:
  - 网关地址变更：修改 `publicBase` → 重启 gitOauth → 生效（单点配置，无硬编码）
  - 不影响主 Django 项目的独立 ALLOWED_HOSTS 策略

## 质量场景

### QS-01: 网关公网 Host 通过 ALLOWED_HOSTS 校验

| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L2 |
| 刺激源 | 浏览器用户通过 CreateProject 页面点击 OAuth 授权 |
| 刺激 | 发送 GET 请求到 `http://183.250.1.132:18081/api/accounts/gitlab/oauth/start-from-gateway/...`，Host 头为 `183.250.1.132:18081` |
| 制品 | gitOauth Django 的 CommonMiddleware + ALLOWED_HOSTS 校验 |
| 环境 | 正常开发环境 |
| 响应 | 请求通过 Host 校验，返回 200 JSON（包含 `authorize_url`） |
| 响应度量 | HTTP 响应码 200，非 400 DisallowedHost |

### QS-02: publicBase 配置变更后重启生效

| 要素 | 内容 |
|------|------|
| 类别 | 可维护性 |
| 等级 | L2 |
| 刺激源 | 运维人员修改网关 IP |
| 刺激 | 修改 `conf/gateway/task-gateway/config.yaml` 中 `publicBase` 为新 IP → 重启 gitOauth |
| 制品 | gitOauth 启动时的 `port_config.load_gateway_public_host()` |
| 环境 | 正常 |
| 响应 | gitOauth ALLOWED_HOSTS 包含新 IP hostname |
| 响应度量 | `load_gateway_public_host()` 返回值等于新 publicBase hostname；ALLOWED_HOSTS 列表包含该值 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 无 | 本次为纯基础设施配置修正，不引入新领域概念或改变现有领域模型 | DDD 步骤可跳过或产出空模型声明 |

## 权衡与边界

### 取舍
- 选择从本地配置文件读取网关地址（耦合到 `conf/gateway/task-gateway/config.yaml`），而非通过环境变量注入。理由：与主 Django 项目的 `allowedExtendHosts` 模式一致，且网关配置已存在、格式稳定

### 明确不做什么
- 不在 V1 引入多网关地址支持（当前仅一个 publicBase）
- 不使用通配符 `*`（与主 Django 的通配符策略不同，gitOauth 保持显式白名单）
- 不修改 APISIX 路由或引入 `proxy-rewrite` 插件

### 升级触发条件
- 当需要支持多个公网网关地址时 → 扩展 `load_gateway_public_host()` 返回列表，或引入 `allowedExtendHosts` 模式
- 当部署到生产环境需要严格安全审查时 → 安全性从 L2 升级到 L3，增加启动时 ALLOWED_HOSTS 校验测试

## 跳过声明

- **性能**: 跳过。Host 校验是 Django request 生命周期中的固定步骤，不新增开销。
- **可用性**: 跳过。本修复本身解决的就是可用性问题（DisallowedHost 400 错误）。
- **数据一致性**: 跳过。纯配置变更，无状态操作。
- **可伸缩性**: 跳过。不引入新负载。
- **容错机制**: 跳过。无新增外部依赖。
- **可观测性**: 跳过。无新增需要监控的指标。DisallowedHost 错误在 Django 错误日志中已有记录。
- **合规与隐私**: 跳过。不涉及。
