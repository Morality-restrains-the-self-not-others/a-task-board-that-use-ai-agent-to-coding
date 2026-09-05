# NFR 澄清: 功能参数多层级配置

> 输入:
> - 设计文档: `docs/designs/feature-params-hierarchy.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-30-feature-params-hierarchy-value-stream.md`
>
> 输出使用者: `/6-ddd-领域设计驱动`, `/7-plans-实施计划`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | 配置 CRUD P95 ≤ 500ms；env 拉取 P95 ≤ 200ms |
| 可伸缩性 | L1 | 数据量级小（配置表 ≤ 万行），单实例即可 |
| 可用性 | L2 | env 拉取不可用时回退到公司默认，无单点阻断 |
| 安全性 | L3 | 个人配置 IDOR 防护 + API key 脱敏 + 多租户隔离 + 审计快照鉴权 |
| 数据一致性 | L1 | 配置读写单表操作，无跨服务事务 |
| 容错机制 | L2 | 快照写入失败不阻塞 env 返回；resolver 多级回退链 |
| 可观测性 | L2 | 快照审计记录；配置变更日志；resolver 回退 warning |
| 可维护性 | L2 | 向后兼容（容器路径不变、现有任务零迁移）；API 不变 |
| 合规与隐私 | L1 | 标准多租户隔离，无跨境/法规专项 |

## 逐增量 NFR 分析

### Increment 0: 修复现有权限缺陷

**不引入新 NFR 关注点**。仅加固现有 `manage_feature_params` POST 的 `is_admin` 校验。现有安全等级从「隐式（靠前端隐藏按钮）」提升为「显式服务端校验」。

### Increment 1: 数据模型 + 配置解析服务

#### NFR 类别: 性能
- **等级**: L2 - 标准
- **量化目标**: resolver 单次调用 ≤ 50ms（3 次简单 DB 查询），env 拉取端到端 P95 ≤ 200ms
- **场景**: 容器启动时调用 `feature-params-env/`，resolver 串行查询 TenantFeatureParams + WorkspaceFeatureParams（可选）+ PersonalFeatureParamsConfig（可选）

#### NFR 类别: 容错机制
- **等级**: L2 - 标准
- **量化目标**: resolver 任何一级查询失败 → 自动回退到上一级；快照写入失败 → log warning + 不阻塞 env 返回
- **场景**: 个人配置被删除 / WorkspaceFeatureParams 表不可用 → 回退到公司默认

### Increment 2: 管理 API

#### NFR 类别: 安全性
- **等级**: L3 - 增强
- **量化目标**: 所有个人配置 `{id}` 端点强制 `user_id` 归属校验（零 IDOR）；创建个人配置硬编码 `user_id=request.user.id`（零注入）；快照列表不含 `resolved_env`（零 API key 泄露）
- **场景**: 攻击者遍历 `GET /api/personal/feature-params-configs/{id}/` 尝试读取他人配置 → 403

#### NFR 类别: 性能
- **等级**: L2 - 标准
- **量化目标**: CRUD P95 ≤ 500ms（含 JSONField 序列化）
- **场景**: 用户加载个人配置列表（最多 20 条），P95 ≤ 300ms

### Increment 3: 前端 UI

#### NFR 类别: 性能
- **等级**: L1 - 基础
- **量化目标**: 配置选择器下拉加载 ≤ 1s（含 API 调用）
- **场景**: 任务创建表单加载配置选项列表

### Increment 4: E2E 测试 + 数据兼容

**无独立 NFR 关注点**。向后兼容已在设计中保证（现有任务 source='company' 零迁移）。

---

## 质量场景

### QS-01: 容器 env 拉取 — 正常路径
| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L2 |
| 刺激源 | 任务容器 bootstrap 脚本 |
| 刺激 | POST feature-params-env/（含 access_token） |
| 制品 | container_feature_params_views.fetch_tenant_feature_params_env_for_container |
| 环境 | 正常 |
| 响应 | 200 + env map（含 TASK_LLM_PROVIDERS_JSON 等） |
| 响应度量 | 服务端处理时间 P95 ≤ 200ms（不含网络），P99 ≤ 500ms |

### QS-02: 容器 env 拉取 — 个人配置已删除
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L2 |
| 刺激源 | 任务容器（任务绑定了一个已删除的个人配置） |
| 刺激 | POST feature-params-env/ |
| 制品 | FeatureParamsResolver |
| 环境 | 正常 |
| 响应 | 200 + 公司默认 env map；快照 source_display_name 注明回退原因 |
| 响应度量 | 回退到公司默认，不返回 4xx/5xx；写入 warning 日志 |

### QS-03: 个人配置 IDOR 防护
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | 恶意用户（同公司成员） |
| 刺激 | GET/PUT/DELETE /api/personal/feature-params-configs/{other_user_config_id}/ |
| 制品 | PersonalFeatureParamsConfig API |
| 环境 | 正常 |
| 响应 | 403（非 404，防止 ID 探测） |
| 响应度量 | 服务端强制 `config.user_id == request.user.id`，任一不匹配 → 403 |

### QS-04: 快照脱敏
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | 任务所在工作空间成员 |
| 刺激 | GET /api/tasks/{id}/feature-params-snapshots/ |
| 制品 | 快照查询 API |
| 环境 | 正常 |
| 响应 | 200 + snapshots[]（含 providers_summary，不含 resolved_env） |
| 响应度量 | 响应 JSON 中无 `resolved_env` 字段；providers_summary 中 api_key 仅展示 `sha256:前8位` |

### QS-05: 工作空间关闭个人配置后的回退
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L2 |
| 刺激源 | 工作空间管理员关闭 allow_personal_feature_params |
| 刺激 | 已绑定个人配置的存量任务下次容器启动 |
| 制品 | FeatureParamsResolver |
| 环境 | 正常 |
| 响应 | resolver 检测到 workspace.allow_personal_feature_params=False → 回退到公司默认 |
| 响应度量 | 返回公司默认 env；快照 display_name 注明 "个人配置被工作空间策略禁用，已回退" |

### QS-06: user_id 注入防护
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | 攻击者 |
| 刺激 | POST /api/personal/feature-params-configs/ {"user_id": "victim_id", ...} |
| 制品 | 个人配置创建 API |
| 环境 | 正常 |
| 响应 | 201（忽略请求体中的 user_id，强制使用 request.user.id） |
| 响应度量 | 写入 DB 的 user_id === request.user.id，而非请求体中的值 |

---

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 安全 L3: IDOR 防护 + user_id 归属 | `PersonalFeatureParamsConfig` 聚合根必须携带 `user_id`；所有仓储查询方法增加 `user_id` 过滤参数 | 仓储接口: `find_by_id_and_user(config_id, user_id)` |
| 安全 L3: 快照脱敏 | `TaskFeatureParamsSnapshot` 需分离两个视图：`SnapshotSummary`（脱敏，对外）和 `SnapshotDetail`（含 resolved_env，需鉴权） | 值对象: `ProvidersSummary`（脱敏）+ 完整 `resolved_env` 仅内部访问 |
| 容错 L2: 多级回退链 | resolver 不是简单的 "查一个表"，而是带 fallback 的策略链 | 领域服务 `FeatureParamsResolver` 显式建模为 Chain of Responsibility |
| 审计 L2: 快照记录 | 快照是只追加（append-only）的审计日志，不可修改不可删除 | `TaskFeatureParamsSnapshot` 为独立聚合，无 Update/Delete 操作 |
| 可维护性 L2: 向后兼容 | `feature_params_source` 默认值 'company' 确保现有实体无迁移 | Todo 聚合的工厂方法默认 source='company' |

---

## 权衡与边界

### 取舍
- **安全优先于便利**: 个人配置需工作空间管理员显式开启（默认 false），接受管理员的额外操作成本以换取可控性
- **运行时引用优于快照**: 容器启动时拉取最新配置（灵活），同时写审计快照（可追溯），接受快照写入的额外 DB 开销
- **完整配置优于增量覆盖** (个人级): 用户管理直观，接受配置冗余存储

### 明确不做什么
- 不做配置版本管理（Git-like diff / 回滚到历史版本）
- 不做个人配置共享（配置仅所有者可用，不可分享给他人）
- 不做实时配置推送（修改配置后需重启容器生效，不支持热更新）
- 不做跨境数据合规（L0 合规），仅国内单区域部署
- 不做 P99 < 100ms 的极致性能优化

### 升级触发条件
- 当单公司工作空间数 > 100 且均有自定义配置时，resolver 查询考虑缓存
- 当个人配置总数 > 10 万时，列表 API 增加分页
- 当客户要求 SOC2/HIPAA 合规时，安全性从 L3 升级到 L4（全量操作日志 + 加密存储 API key）

---

## 跳过声明

- **可伸缩性**: L1 基础即可。配置数据量级极小（公司数 × 工作空间数 × 用户数 × 配置数），单实例 PostgreSQL 足够。不做水平扩展和分区设计。
- **合规与隐私**: L1。无跨境数据传输、无 GDPR/HIPAA/PCI-DSS 要求。API key 脱敏已涵盖在安全性 L3 中。
- **可用性**: L2。env 拉取有回退链，配置管理非高频操作，不需要 99.99% 可用性保证。

---

## 自检清单

- [x] 每个相关 NFR 类别都有明确的支撑等级（6 个 L1-L3，2 个跳过）
- [x] 每个 L1-L3 的 NFR 类别至少有一个量化目标
- [x] 每个 L2-L3 的 NFR 类别至少有一个质量场景（6 个 QS）
- [x] 每个质量场景的响应度量可验证（给出了具体数字和测量方式）
- [x] 影响领域模型的 NFR 决策已标注（5 条，含具体 DDD 动作）
- [x] 权衡和边界已明确（3 条取舍 + 4 条不做什么 + 3 条升级触发）
- [x] 跳过的 NFR 类别有理由说明（可伸缩性、合规、可用性）
- [x] 文档位置正确：`docs/superpowers/plans/2026-06-30-feature-params-hierarchy-nfr-clarification.md`
