# NFR 澄清: SLO 代码审查修复

> 输入:
> - 设计文档: `docs/superpowers/specs/2025-06-25-slo-review-fixes-design.md`
> - 价值流文档: `docs/superpowers/plans/2025-06-25-slo-review-fixes-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 安全性 | L2 | post_logout_redirect_uri 白名单验证（scheme + host 匹配），拒绝外部 URL |
| 可观测性 | L2 | 异常日志记录（logger.warning + exc_info），非静默吞噬 |
| 数据一致性 | L1 | 保持现有 best-effort SLO（cookie 清除 + 令牌销毁，无分布式事务） |
| 容错机制 | L1 | 保持现有优雅降级（taskAuth 不可达时跳过令牌销毁，继续 Django 登出） |
| 可维护性 | L1 | 删除死代码（SessionTermination, CustomTokenRepository），函数去重 |
| 性能 | L0 | 无性能影响（改动仅在登出路径，不涉及热路径） |
| 可用性 | L0 | 无变更 |
| 可伸缩性 | L0 | 无变更 |
| 合规与隐私 | L0 | 不适用 |

## 逐增量 NFR 分析

### Increment 1: EndSession 安全加固

#### NFR 类别: 安全性
- **等级**: L2 - 标准
- **量化目标**: 100% 非白名单 post_logout_redirect_uri 被拒绝（返回 302 至默认登录页）
- **设计约束**: 白名单来源为 cfg.GatewayPublicBase + cfg.GitServicePublicBase，scheme 限制 http/https，host 完全匹配

### Increment 2: Cookie 安全属性 + 异常日志

#### NFR 类别: 可观测性
- **等级**: L2 - 标准
- **量化目标**: 100% forward_to_taskauth 异常被记录（含 exc_info），运维可通过日志定位失败原因

#### NFR 类别: 安全性
- **等级**: L2 - 标准
- **量化目标**: Set-Cookie 包含 SameSite=Lax，浏览器在跨站请求中正确携带/清除 cookie

### Increment 3: 代码质量
- **可维护性**: L1 - 基础（死代码删除 + 函数去重，无量化目标）

### Increment 4: 测试修复
- **已由现有测试框架覆盖**，不新增 NFR

## 质量场景

### QS-01: 防止开放重定向
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L2 |
| 刺激源 | 恶意用户 |
| 刺激 | 构造 GET /api/oidc/endsession?post_logout_redirect_uri=https://evil.com/ |
| 制品 | handleOidcEndSession |
| 环境 | 正常 |
| 响应 | 302 重定向至默认登录页（cfg.GatewayPublicBase + "/auth/login/"），不重定向至 evil.com |
| 响应度量 | Go 单元测试断言 Location header 不包含 evil.com；手动安全测试验证 |

### QS-02: 异常日志可见
| 要素 | 内容 |
|------|------|
| 类别 | 可观测性 |
| 等级 | L2 |
| 刺激源 | taskAuth 服务不可达 |
| 刺激 | Django logout 视图中 forward_to_taskauth 抛出 ConnectionError |
| 制品 | UserViewSet.logout |
| 环境 | 降级 |
| 响应 | logger.warning("taskAuth unreachable during logout", exc_info=True) 写入日志 |
| 响应度量 | 日志文件/聚合器中可见 WARNING 级别日志，包含异常堆栈 |

### QS-03: Cookie 正确清除
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L2 |
| 刺激源 | 正常登出用户 |
| 刺激 | 点击主应用退出按钮 |
| 制品 | handleOidcEndSession + UserViewSet.logout |
| 环境 | 正常 |
| 响应 | userId、sessionid、_gitlab_session cookies 全部清除（SameSite=Lax, Path=/） |
| 响应度量 | Playwright E2E (SLO-003) 验证 cookie 值为空或不存在 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 安全性 L2: redirect_uri 白名单 | 不引入新领域类型（白名单验证在基础设施层实现，不改变领域模型） | DDD 步骤无需新增实体/值对象 |
| 可观测性 L2: 异常日志 | 不影响领域模型 | 无 |
| 可维护性 L1: 死代码删除 | 删除 SessionTermination 值对象（已被 EndSession 处理器弃用） | 从 domain/ 目录移除该文件 |
| 一致性 L1: best-effort SLO | 保持不变（cookie 清除 + 令牌销毁，无分布式事务） | 无 |

**本次修复不引入新领域概念。** DDD 步骤主要为：确认 SessionTermination 移除不破坏其他限界上下文。

## 权衡与边界

### 取舍
- **GitLab 会话清除**: 选择 cookie-only（L1 一致性）而非服务端销毁（L3 一致性）。GitLab CE GET /sign_out 500 修复成本高，cookie 清除满足同域部署的 90%+ 场景。

### 明确不做什么
- 不修复 GitLab CE 19.0 的 GET /sign_out 500 问题（修改 GitLab 源码影响升级）
- 不引入 OAuth client 注册表依赖（白名单基于已有配置项）
- 不将 L2 安全提升至 L3（不引入 OAuth 2.0 Pushed Authorization Requests）

### 升级触发条件
- **当 GitLab 部署在不同域名时**: cookie 清除失效 → 需升级为服务端 API 调用销毁 GitLab session → 一致性从 L1 升至 L3
- **当需要支持多 OIDC RP 时**: 白名单仅支持 2 个 origin → 需升级为 OAuth Client 注册表验证 → 安全性从 L2 升至 L3

## 跳过声明
- **性能**: L0。改动仅在登出路径（低频操作），不涉及热路径。
- **可用性**: L0。无变更，保持现有可用性基线。
- **可伸缩性**: L0。无变更。
- **合规与隐私**: L0。不适用（无新数据处理需求）。
