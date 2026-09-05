# DDD 领域建模：daydaymoney.yaml 元信息

**日期**: 2026-07-18  
**迭代**: daydaymoney-yaml-metadata

## 限界上下文

| 上下文 | 职责 |
|--------|------|
| **DaydaymoneyIdentity**（新概念，非新服务） | 仓库身份：`ServiceId`, `DaydaymoneyTags`；解析/校验 YAML |
| **ProjectCatalog**（既有 taskProjectService） | 项目 tags 索引；Resolve 归属；工作空间关联 |
| **Observability**（tracelog / Loki / Grafana） | 日志携带身份；匹配建任务 |
| **BrowserAssist**（Chrome 插件） | 读页面身份 → 调 Resolve → 填充 UI |

## 领域对象

| 对象 | 类型 | 说明 |
|------|------|------|
| `ServiceId` | VO | 稳定字符串；约束字符集 |
| `DaydaymoneyTag` | VO | 单项 tag；规范含 `svc:<ServiceId>` |
| `DaydaymoneyMeta` | Entity/Document | YAML 根：version + serviceId + tags + displayName |
| `AidevMatch` | VO | `(companyId, workspaceId, projectId, matchedTags)` |
| `Project.tags` | 既有 | 作为反向索引 |

## 领域服务

- `ParseAidevYAML(raw) → DaydaymoneyMeta | Error`
- `ResolveByMeta(tenantId, serviceId?, tag?) → []AidevMatch`
- `MergeProjectTags(existing, fromMeta) → tags`（union + normalize）

## 端口

| 端口 | 适配器 |
|------|--------|
| `DaydaymoneyMetaParser` | Go yaml / JS js-yaml 或轻量解析 |
| `ProjectTagIndex` | SQLite projects.tags + project_workspaces |
| `PageMetaReader` | DOM meta / fetch `/daydaymoney.yaml` |
| `LogEnricher` | tracelog With 字段 |

## 事件

| 事件 | 何时 | 消费者 |
|------|------|--------|
| `PROJECT_TAGS_SYNCED_FROM_AIDEV` | 可选；同步成功后 | 暂无 → **书面例外**：与普通 tags PATCH 相同持久化路径，不单独投递 MQ |

只读 resolve/parse：**书面例外**不投递。
