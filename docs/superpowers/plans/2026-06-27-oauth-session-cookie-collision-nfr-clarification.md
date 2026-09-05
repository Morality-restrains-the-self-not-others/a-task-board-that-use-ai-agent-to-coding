# NFR 澄清: OAuth Session Cookie 冲突修复

> 输入:
> - 设计文档: `docs/specs/oauth-redirect-loop-port-4000/design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-27-oauth-session-cookie-collision-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 安全性 | L2 | Session cookie 命名空间隔离，防止跨服务 session 劫持 |
| 可维护性 | L1 | 单行配置变更，零业务逻辑修改 |
| 性能 | L0 | 不适用 |
| 可伸缩性 | L0 | 不适用 |
| 可用性 | L0 | 不适用 |
| 数据一致性 | L0 | 不适用 |
| 容错机制 | L0 | 不适用 |
| 可观测性 | L1 | 建议添加 session cookie 设置日志 |
| 合规与隐私 | L0 | 不适用 |

## 逐增量 NFR 分析

### Increment 1: Session Cookie 命名隔离

#### NFR 类别: 安全性
- **等级**: L2 - 标准
- **量化目标**: gitOauth 与主站 Django 使用不同的 session cookie 名，OAuth 流程中主站 `sessionid` 不被覆盖
- **原理**: 修复前两个 Django 服务共用 `sessionid` cookie 名，同域名下后者覆盖前者，导致主站用户认证状态丢失。修复后 gitOauth 使用 `gitoauth_sessionid`，主站 session 不受影响

#### NFR 类别: 可维护性
- **等级**: L1 - 基础
- **量化目标**: 单行 `settings.py` 配置变更，无 API 变更，无数据库迁移
- **回滚方案**: 删除 `SESSION_COOKIE_NAME = 'gitoauth_sessionid'` 行即可回滚

## 质量场景

### QS-01: OAuth 流程不破坏主站 Session
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L2 |
| 刺激源 | 用户在项目详情页点击 OAuth 授权 |
| 刺激 | 完整的 OAuth 流程（start → GitLab 授权 → callback → 回跳） |
| 制品 | gitOauth session cookie + 主站 session cookie |
| 环境 | 正常负载 |
| 响应 | OAuth 完成后主站 `sessionid` 值不变，用户认证状态保持 |
| 响应度量 | 浏览器 Cookies 中 `sessionid` 值在 OAuth 前后一致；`gitoauth_sessionid` 独立存在 |

## 领域模型影响

无。本次修复是基础设施层（Django settings）的配置变更，不引入新的领域概念，不修改任何实体、值对象、聚合或领域服务。

## 权衡与边界

### 取舍
- 无架构级权衡。选择最小侵入方案（cookie 重命名）而非更复杂的跨服务 session 共享方案。

### 明确不做什么
- 不引入集中式 session 服务（如 Redis session store）—— 当前的单服务 session 方案满足需求
- 不修改 Gateway 的 cookie 处理逻辑 —— Gateway 保持透传
- 不修改主站 Django 的 session 机制

### 升级触发条件
- 当需要跨服务的统一 session 管理时（如多个 Django 服务需共享登录状态），升级到集中式 session store
- 当域名增多需要跨域 session 时，升级 SSO 方案

## 跳过声明

- **性能**: 跳过。Cookie 命名变更不影响请求处理路径，零性能影响。
- **可伸缩性**: 跳过。配置变更，不引入新的有状态组件或数据流。
- **可用性**: 跳过。不影响任何服务的可用性。
- **数据一致性**: 跳过。OAuth 流程的 session 状态管理不涉及数据库事务。
- **容错机制**: 跳过。不引入新的外部依赖或网络调用。
- **合规与隐私**: 跳过。Session cookie 内容不变（仍为 Django signed cookie），隐私特性不变。
- **可观测性**: L1 基础（建议日志），非强制。修复本身不需要额外的可观测性投入。
