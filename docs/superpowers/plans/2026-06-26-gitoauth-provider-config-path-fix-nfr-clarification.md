# NFR 澄清: gitOauth Provider 配置加载路径修复

> 输入:
> - 设计文档: `.claude/skills/1-brainstorming-设计文档/design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-26-gitoauth-provider-config-path-fix-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L0 | 不适用 — 配置加载仅在启动时执行一次 |
| 可伸缩性 | L0 | 不适用 — 无新数据流或用户路径 |
| 可用性 | L0 | 不适用 — 本修复恢复可用性，不引入新可用性需求 |
| 安全性 | L0 | 不适用 — OAuth 安全机制不变 |
| 数据一致性 | L0 | 不适用 — 配置加载为只读操作 |
| 容错机制 | L2 | 路径回退：主路径失败时自动尝试旧路径 |
| 可观测性 | L2 | DEBUG 模式下 OAuth 启动错误携带真实原因 |
| 合规与隐私 | L0 | 不适用 |
| 可维护性 | L2 | `port_config.py` 与 `conf_loader.py` 路径对齐，消除分歧 |

## 逐增量 NFR 分析

### Increment 1: 修正提供者配置加载路径（Thin Slice）

#### NFR 类别: 可维护性
- **等级**: L2 - 标准
- **量化目标**: `port_config.py` 提供者加载路径与 `runAll/scripts/conf_loader.py` 一致，使用相同的 `conf/auth/git-oauth/providers/` 目录
- **质量场景**: QS-01

### Increment 2: 路径回退兼容（Essential Support）

#### NFR 类别: 容错机制
- **等级**: L2 - 标准
- **量化目标**: 主路径不可用时自动回退到旧路径，不抛出异常、不返回空配置
- **质量场景**: QS-02

### Increment 3: 可观测性增强（Enhancement）

#### NFR 类别: 可观测性
- **等级**: L2 - 标准
- **量化目标**: DEBUG=True 时，OAuth start 失败响应附带内部错误原因，排查时间从「查看日志」缩短到「直接看响应体」
- **质量场景**: QS-03

## 质量场景

### QS-01: 提供者配置从正确路径加载
| 要素 | 内容 |
|------|------|
| 类别 | 可维护性 |
| 等级 | L2 |
| 刺激源 | gitOauth 服务启动 (Django settings 初始化) |
| 刺激 | `load_merged()` 被调用 |
| 制品 | `port_config.py:load_merged()` |
| 环境 | 正常启动 |
| 响应 | 从 `conf/auth/git-oauth/providers/*.yaml` 加载所有提供者配置，`out["gitOauth"]` 非空 |
| 响应度量 | `len(GITOAUTH_PROVIDER_CONFIGS) >= 1`（至少一个 gitlab provider），单测可验证 |

### QS-02: 路径回退兼容
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L2 |
| 刺激源 | gitOauth 服务启动 |
| 刺激 | 主路径 `conf/auth/git-oauth/providers/` 不存在（如目录被移动） |
| 制品 | `port_config.py:load_merged()` |
| 环境 | 异常部署（主路径缺失，旧路径存在） |
| 响应 | 自动回退到 `conf/git-oauth/providers/`（旧路径），成功加载配置 |
| 响应度量 | 不抛出异常，`out["gitOauth"]` 非空，日志中记录回退事件 |

> **注意：** QS-02 为可选增强（Increment 2），薄切片 (Increment 1) 不要求此场景。

### QS-03: DEBUG 模式错误透传
| 要素 | 内容 |
|------|------|
| 类别 | 可观测性 |
| 等级 | L2 |
| 刺激源 | 前端 CreateProject 页面触发 OAuth 授权 |
| 刺激 | `GET /api/accounts/gitlab/oauth/start-from-gateway/` 但提供者配置加载失败 |
| 制品 | `GitlabOAuthStartFromGatewayView.get()` |
| 环境 | DEBUG=True |
| 响应 | 返回 503，body 中 `detail` 字段包含 `[debug: missing_allowed_host]` 后缀 |
| 响应度量 | 响应体中可读到具体错误原因，无需查看服务端日志 |

## 领域模型影响

本修复为纯基础设施配置路径修正，**不引入新领域概念、不改变现有领域模型**。

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| — | 无影响 | 无需 DDD 建模 |

## 权衡与边界

### 取舍
- 选择简单路径修复（一行改动）而非重构整个配置加载机制 — 优先恢复功能，不引入不必要的抽象层
- 选择在 DEBUG 模式暴露错误原因而非生产环境 — 安全优先

### 明确不做什么
- 不重构 `load_merged()` 的整体架构（如统一 YAML 加载器）
- 不将 `conf_loader.py`（runAll）与 `port_config.py`（gitOauth）合并为共享模块
- 不在生产环境暴露 OAuth 配置内部状态
- 不做配置热加载 — 配置变更需重启 gitOauth

### 升级触发条件
- 如果将来出现第三次「路径不对」的同类 bug → 重构为统一配置加载入口，消除路径重复
- 如果提供者 YAML 文件数量超过 20 个 → 考虑目录监控 + 热加载
- 如果需要支持配置变更免重启 → 引入配置中心（如 etcd/Consul）

## 跳过声明

| 跳过的 NFR | 理由 |
|-----------|------|
| 性能 | 配置仅在启动时加载一次，不影响请求路径延迟 |
| 可伸缩性 | 无新数据流或用户路径 |
| 可用性 | 本修复恢复可用性（修复空配置导致的 503），不引入新的可用性需求 |
| 安全性 | OAuth 安全机制（state/csrf/token 交换）不变，路径修复不触及安全边界 |
| 数据一致性 | 配置加载为只读操作，无状态写入 |
| 合规与隐私 | 不涉及用户数据处理 |

## 自检

- [x] 每个相关 NFR 类别都有明确的支撑等级（L0-L4）
- [x] 每个 L1-L4 的 NFR 类别至少有一个量化目标
- [x] 每个 L2-L4 的 NFR 类别至少有一个质量场景（QS）
- [x] 每个质量场景的响应度量可验证
- [x] 影响领域模型的 NFR 决策已标注（本次无影响）
- [x] 权衡和边界已明确
- [x] 跳过的 NFR 类别有理由说明
- [x] 文档位置正确：`docs/superpowers/plans/2026-06-26-gitoauth-provider-config-path-fix-nfr-clarification.md`
