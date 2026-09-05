# NFR 澄清: OAuth 回调失败 Toast

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-05-27-oauth-callback-error-toast-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-05-27-oauth-callback-error-toast-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L1 | 路由 afterEach 回调处理 < 5ms（同步路径） |
| 可伸缩性 | L0 | 不适用（纯前端、无服务端状态） |
| 可用性 | L1 | Toast 展示不阻塞页面渲染与 API 请求 |
| 安全性 | L2 | 回调后 1 次导航内清除 URL 中的 OAuth 错误码；Toast 仅展示预置文案 |
| 数据一致性 | L0 | 不适用（不写入/不读取持久化绑定状态） |
| 容错机制 | L1 | 未知错误码有兜底文案；guard 异常不导致白屏 |
| 可观测性 | L1 | 无新增后端指标；可选前端手工验收 |
| 合规与隐私 | L0 | 不适用（不新增 PII 采集） |
| 可维护性 | L2 | 文案单点 `gitSiteOAuthCallbackUtils`；设置页 skip 列表可配置扩展 |

**跳过说明：** 本特性为前端展示层增强，不引入新 API、新表或分布式事务；性能/安全/可维护性为相关维度，其余标注 L0。

---

## 逐增量 NFR 分析

### Increment 1: 回调失败 Toast 薄切片

#### 性能 — L1 基础

- **量化目标：** `applyOAuthCallbackFromRoute` 在单次 `afterEach` 内完成（无 await 远程 IO）；用户感知 Toast 出现延迟 < 100ms（首屏绘制后）。
- **质量场景：** QS-01

#### 可用性 — L1 基础

- **量化目标：** OAuth 失败回跳后，用户在 **5 秒内** 必须能看到错误 Toast（duration=5000ms）；页面主内容（项目详情加载）不因 guard 失败而中断。
- **质量场景：** QS-02

#### 安全性 — L2 标准

- **量化目标：** 处理完成后 URL 不得仍含 `gitlab`/`github` 回调参数；Toast 文案仅来自 `GITHUB_CALLBACK_HINTS` / `GITLAB_CALLBACK_HINTS`，不拼接后端原始 exception。
- **质量场景：** QS-03

#### 可维护性 — L2 标准

- **量化目标：** 新增 OAuth 错误码时只需改 1 个 hints 对象 + 1 条 Vitest；设置页与业务页文案零分叉。
- **质量场景：** QS-04

### Increment 2: return_key 入口对齐

#### 可维护性 — L2 标准

- 与 Increment 1 相同；额外要求 CreateTaskModal 与 ProjectDetail 使用同一 `return_key` 契约（参数名、`setGithubAppReturnTarget` TTL）。

### Increment 3: 设置页 skip 回归

#### 安全性 / 可用性 — L2 / L1

- **量化目标：** `shouldSkipOAuthCallbackToast` 对三条 `git-site-oauth` 路径 **100%** 不触发 Toast；内联 `errorMessage` 仍为唯一错误面。
- **质量场景：** QS-05

### Increment 4: 多入口回归护栏

#### 可维护性 — L2 标准

- view_test 文档覆盖 ≥4 个 return_key 入口；无自动化 SLA（L1 可观测性）。

---

## 质量场景

### QS-01: 路由守卫同步开销

| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L1 |
| 刺激源 | Vue Router 完成导航 |
| 刺激 | 目标 route 含 `?gitlab=profile_failed` |
| 制品 | `setupOAuthCallbackToastGuard` → `applyOAuthCallbackFromRoute` |
| 环境 | 正常负载、本地 dev |
| 响应 | 同步解析 query、触发 Toast、`router.replace` 清 query |
| 响应度量 | Vitest 单测 < 50ms；Chrome Performance 无 >5ms 长任务（手工抽检） |

### QS-02: 失败回跳用户可见性

| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L1 |
| 刺激源 | 用户完成 GitLab OAuth 授权（失败：profile_failed） |
| 刺激 | 回跳至项目详情 URL |
| 制品 | 全局 Toast + `ProjectDetail` 页面 |
| 环境 | 正常 |
| 响应 | 红色 Toast 显示「授权失败：无法读取 GitLab 用户资料」；项目详情仍可加载 |
| 响应度量 | 手工 E2E：回跳后 1s 内可见 Toast；`fetchProjectDetail` 仍发起（网络面板） |

### QS-03: URL 错误码不残留

| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L2 |
| 刺激源 | OAuth 回调合并 query |
| 刺激 | `gitlab=exchange_failed` 出现在业务页 URL |
| 制品 | `applyOAuthCallbackFromRoute` |
| 环境 | 正常 |
| 响应 | `router.replace` 后 `location.search` 不含 `gitlab=` / `github=` |
| 响应度量 | Vitest 断言 `replace` 的 query；手工刷新页面不重复弹 Toast |

### QS-04: 文案单点维护

| 要素 | 内容 |
|------|------|
| 类别 | 可维护性 |
| 等级 | L2 |
| 刺激源 | 开发者新增错误码 `no_gitlab_id` |
| 刺激 | 仅修改 `GITLAB_CALLBACK_HINTS` |
| 制品 | `gitSiteOAuthCallbackUtils.js` |
| 环境 | 开发 |
| 响应 | Toast 与设置页内联错误显示相同字符串 |
| 响应度量 | `UserGitSiteOAuthSettings` import 同一模块；Vitest `resolveOAuthCallbackMessage` 覆盖新 key |

### QS-05: 设置页不重复 Toast

| 要素 | 内容 |
|------|------|
| 类别 | 安全性 / 可用性 |
| 等级 | L2 / L1 |
| 刺激源 | 用户在 Git 站点 OAuth 设置页完成失败回调 |
| 刺激 | `?gitlab=bad_state` |
| 制品 | `shouldSkipOAuthCallbackToast` + 设置页内联 UI |
| 环境 | 正常 |
| 响应 | 无全局 Toast；内联 `errorMessage` 展示 |
| 响应度量 | view_test `oauth-callback-error-toast-settings-skip-regression.md` 手工清单 |

---

## 领域模型影响

| NFR 决策 | 领域模型影响 | 具体动作 |
|----------|-------------|---------|
| 无持久化写入 (L0 一致性) | 前端限界上下文内建模即可 | DDD 步骤：`OAuthCallbackCode` 值对象 + `OAuthCallbackNotifier` 领域服务（接口），无新聚合根 |
| 预置文案 (L2 安全) | 错误码 → 文案映射为值对象工厂 | `resolveOAuthCallbackMessage(provider, code)` 纯函数，禁止拼接外部输入 |
| skip 路径 (L2 可维护) | 路由策略为领域服务策略 | `shouldSkipOAuthCallbackToast(path)` 可扩展为策略列表，不入实体状态 |

**说明：** 本特性不改变 `git-oauth.api_githubappusercredential` 写入契约；与 `bind_error` 的关联仅在展示层对齐文案，无跨上下文事务。

---

## 权衡与边界

### 做了什么取舍

- **Toast 仅失败、不成功：** 降低打扰（可用性 L1），成功态依赖设置页内联或静默。
- **全局 guard 而非每页组件：** 可维护性 L2，接受每次路由导航的微量开销（性能 L1）。
- **Vitest 为主、valueStream pytest 为 planned：** 前端测试不在 `Saas_project` working_dir，价值流环节用 view_test 手工契约。

### 明确不做什么

- 不修复 `profile_failed` 根因（GitLab API / token / scope）—— 属 gitOauth 后端增量，非本 NFR 范围。
- 不做 i18n 多语言 Toast（L0）。
- 不做 Toast 队列/stacking（L0）；沿用现有单例 `toastService`。
- 不在 URL 长期保留错误码供分享/书签（安全 L2 边界）。

### 触发升级条件

| 条件 | 升级项 |
|------|--------|
| 日活 OAuth 回跳 >1 万次/日且 guard 测得长任务 | 性能 L1 → L2（懒处理、仅匹配 query 存在时执行） |
| 需将错误码上报 APM/Sentry | 可观测性 L1 → L2（guard 内结构化 log） |
| 监管要求 OAuth 失败审计留痕 | 安全性 L2 → L3（前端事件 + 后端 audit 关联） |

---

## 自检

- [x] 每个 QS 有刺激、响应、可度量验证方式
- [x] L0 类别已声明不适用理由
- [x] 领域影响仅限前端 BC，未过度建模后端聚合
- [x] 增量 2–4 继承增量 1 的 NFR 基线，无重复堆砌
