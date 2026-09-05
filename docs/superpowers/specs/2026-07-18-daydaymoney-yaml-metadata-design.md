# 创建设计：daydaymoney.yaml 仓库元信息全链路

**日期**: 2026-07-18  
**状态**: 已采用（goal-mode 自动采用）  
**迭代**: daydaymoney-yaml-metadata  
**范围**: 仓库元文件 → 项目标签 / HTML header / Chrome 浮窗反查 / 结构化日志 → DaydaymoneyGrafana 匹配

## 1. 目标与成功标准

| # | 标准 | 验收 |
|---|------|------|
| S1 | 各服务仓库根存在 `daydaymoney.yaml`，含稳定 `service_id` + `tags` | CI 可校验 schema；禁止写入 workspace/project/company id |
| S2 | 项目页可从关联仓库解析元信息并自动填充/合并项目 `tags` | Create/Edit/详情提供「从 daydaymoney.yaml 同步」；合并后走既有 tags 规范化 |
| S3 | 前端入口页 `<head>` 暴露元信息 | `meta[name=daydaymoney-service-id]` + `meta[name=daydaymoney-tags]`（保留既有 `trae-service`） |
| S4 | Chrome 浮窗解析页面 meta（或同源 `/daydaymoney.yaml`），调 SaaS resolve API，自动勾选匹配的工作空间/项目 | 一对多全部预选；无匹配不覆盖用户上次选择 |
| S5 | 服务日志 JSON 携带 `daydaymoney_service_id` / `daydaymoney_tags` | Go tracelog / Python 结构化日志默认注入 |
| S6 | DaydaymoneyGrafana 优先按日志元信息匹配项目 tags（精确），再回退既有 regex | unit test 覆盖精确匹配与多项目 |
| S7 | Resolve API 按元信息返回**多** `(company, workspace, project)` | 同一 `service_id` 可映射多工作空间多项目 |

## 2. 非目标

- 不把 `daydaymoney` 开发隧道 nginx 与本元信息子系统混名（脚本仍叫 daydaymoney-daemon；元文件约定独立）
- 不强制一次性改完所有历史 CaptureRule；旧 regex 规则继续可用
- 不在 YAML 中固化租户/工作空间/项目 ID（一对多归属以 SaaS 侧 tags 索引为准）
- 不新建微服务；扩展 `taskProjectService` + 既有前端/插件/日志库

## 3. 核心约束：一对多反查

**问题**：同一仓库可加入不同工作空间的不同项目。

**原则**：

1. `daydaymoney.yaml` 只描述**仓库/服务自身身份**（`service_id`、`tags`、展示名），是 git 内权威。
2. SaaS 项目通过 `tags`（至少包含 `svc:<service_id>`）声明「我关联该服务」。
3. 运行时反查：`service_id` / tags → 查询所有带匹配 tag 的项目及其 `project_workspaces` → 返回多条归属。
4. 客户端（插件/Grafana）在多匹配时全部采纳或让用户多选；**禁止**假设唯一 workspace/project。

```
daydaymoney.yaml (git)          SaaS projects.tags
─────────────────             ──────────────────
service_id: foo    ──svc:──▶  Project A @ WS1   tags: [svc:foo]
tags: [svc:foo]              Project B @ WS2   tags: [svc:foo]
                             Project C @ WS1   tags: [svc:foo, team:x]
```

## 4. 方案选型（自动采用）

| 方案 | 说明 | 结论 |
|------|------|------|
| A. YAML 身份 + 项目 tags 索引 + Resolve API | 解耦多归属；复用现有 tags | **采用** |
| B. YAML 内写 workspace/project id 列表 | 一对多难维护、跨租户泄漏风险 | 否 |
| C. 仅靠 git_repos URL 反查 | URL 别名/fork/monorepo 子路径脆弱 | 否（可作为辅助，非主键） |
| D. Grafana 仅扩 regex | 插件/项目页/日志无法统一 | 否 |

**架构理解（基于 v37 current）**：项目与工作空间多对多已在 `taskProjectService`（`projects.tags` + `project_workspaces`）；Chrome/Grafana 已会拉工作空间与项目列表。缺口是**稳定服务身份文件**与**服务端按 tag/service_id 反查**。

## 5. `daydaymoney.yaml` Schema（v1）

文件位置：各 git 仓库根目录（monorepo 内每个独立服务顶层目录一份；根仓库可选一份 `service_id: ram-work`）。

```yaml
# daydaymoney.yaml — 仓库/服务元信息（禁止写入 workspace_id / project_id / company_id）
version: 1
service_id: taskProjectService          # 必填；稳定标识，建议 = 顶层目录名
display_name: Task Project Service      # 可选；人读名
tags:                                   # 必填；至少含 svc:<service_id>
  - svc:taskProjectService
  - domain:project
description: Project/workspace CRUD     # 可选
```

| 字段 | 约束 |
|------|------|
| `version` | 整数，当前仅 `1` |
| `service_id` | `[A-Za-z][A-Za-z0-9_-]{1,63}`；全局约定唯一 |
| `tags` | 1–20 个；单项 ≤64；必须包含 `svc:<service_id>`（大小写敏感写入，匹配时对 tag 做大小写不敏感） |
| 禁止键 | `workspace_id(s)`、`project_id(s)`、`company_id`、`tenant_id` |

共享解析：`shareLib/daydaymoneymeta`（Go）+ `task2app/.../daydaymoneyMeta.js`（前端）+ 插件同构解析。

## 6. API（Go / taskProjectService）

### 6.1 按元信息反查归属（新建）

```
GET /api/tenant/{tenant_id}/daydaymoney/resolve?service_id={id}
GET /api/tenant/{tenant_id}/daydaymoney/resolve?tag={tag}
GET /api/tenant/{tenant_id}/daydaymoney/resolve?service_id={id}&tag={tag}  # AND
```

鉴权：与项目列表相同（gateway JWT + 租户内用户）。

**200**：
```json
{
  "status": "success",
  "query": { "service_id": "taskProjectService", "tag": "" },
  "matches": [
    {
      "company_id": "t1",
      "project_id": "p1",
      "project_name": "Proj A",
      "workspace_id": "w1",
      "workspace_name": "WS1",
      "matched_tags": ["svc:taskProjectService"]
    },
    {
      "company_id": "t1",
      "project_id": "p2",
      "project_name": "Proj B",
      "workspace_id": "w2",
      "workspace_name": "WS2",
      "matched_tags": ["svc:taskProjectService"]
    }
  ]
}
```

匹配规则：

1. 规范化查询：`service_id` → 隐式 tag `svc:<service_id>`；显式 `tag` 原样。
2. 项目 `tags` 中任一 tag 与查询 tag **大小写不敏感相等** 即命中。
3. 每个命中项目展开其全部 `project_workspaces` 行；无工作空间关联则仍返回 `workspace_id: ""`（便于诊断）。
4. 空查询 → 400。

### 6.2 项目列表可选过滤（增强）

```
GET /api/tenant/{tenant_id}/projects/?tag=svc:foo
```

服务端过滤（减轻前端/插件全量拉取）。与 `workspace_id` 可组合。

### 6.3 从 YAML 文本建议标签（新建，供项目页）

```
POST /api/tenant/{tenant_id}/daydaymoney/parse-yaml
Content-Type: application/json
{ "yaml": "version: 1\n..." }
```

**200**：`{ "status":"success", "service_id":"...", "tags":["svc:..."], "display_name":"..." }`  
校验失败 → 400 + 字段错误。不落库；仅解析。

> 项目页「从仓库同步」：前端/Go 侧拉仓库 raw `daydaymoney.yaml`（既有 git 访问能力）→ `parse-yaml` → merge 进项目 tags PATCH。

Swagger：`taskProjectService/src/openapi.yaml` 同步；`db/api_route_ownership.yaml` 登记 owner=`taskProjectService`。

## 7. 前端

### 7.1 项目页

- `CreateProject` / `ProjectEdit` / `ProjectDetailInlineEditableFields`：增加「从 daydaymoney.yaml 同步标签」
- 流程：对每个 `git_repos` URL 尝试获取默认分支根文件；成功则 merge tags（`normalizeProjectTags`）；失败 toast 提示（带 `data-traceId`）
- 本地 monorepo 开发可选：粘贴 YAML / 上传文件 → `parse-yaml`

### 7.2 Header meta

各服务入口 HTML（及 Vite 注入）：

```html
<meta name="trae-service" content="task2app" />
<meta name="daydaymoney-service-id" content="task2app" />
<meta name="daydaymoney-tags" content="svc:task2app,domain:saas" />
```

- `daydaymoney-tags`：逗号分隔，与 YAML tags 一致
- CI：扩展 `check_frontend_head_trae_service.py` 或新增 `check_daydaymoney_meta.py`（校验 YAML 存在 + head 与 YAML 一致）

构建时：可由脚本从同目录 `daydaymoney.yaml` 生成 meta 片段，避免手写漂移。

## 8. taskChromePlugin

1. 浮窗打开时：读 `document.querySelector('meta[name="daydaymoney-service-id"]')`；若无则 `fetch(origin + '/daydaymoney.yaml')`（同源静态托管时）
2. 对用户每个 `company` 调 `GET .../daydaymoney/resolve?service_id=`
3. 合并 `matches`：
   - 唯一 workspace → 自动选中该 workspace + 勾选全部匹配 projects
   - 多 workspace → 预勾选全部匹配项；UI 提示「已根据页面元信息匹配 N 个项目」
   - 零匹配 → 保留 `Storage` 中上次选择
4. 解析逻辑放 `lib/daydaymoney-meta.js`；API 放 `lib/api.js`（`resolveDaydaymoneyMeta`）

## 9. 日志与 DaydaymoneyGrafana

### 9.1 日志字段（扩展观测规范）

在既有 `service` / `trace_id` 之外增加（可空但键建议始终存在）：

| 字段 | 说明 |
|------|------|
| `daydaymoney_service_id` | 与 YAML `service_id` 一致 |
| `daydaymoney_tags` | 字符串数组或逗号串；推荐 JSON 数组 |

Go：`tracelog.Init` 可选 `InitWithDaydaymoney(service, meta)` 或 `SetDaydaymoneyMeta(serviceID, tags)`，默认 slog.With 注入。  
Python：结构化 Formatter 读环境变量 / 模块常量。  
启动时从同进程工作目录或嵌入的 `daydaymoney.yaml` 加载（找不到则 `daydaymoney_service_id=service` 名、`daydaymoney_tags=["svc:<service>"]`）。

### 9.2 Grafana 匹配

`projectMatcher.ts` 增加路径：

1. 若日志含 `daydaymoney_service_id` 或 `daydaymoney_tags` → 精确匹配项目 tags（`svc:` 或任意 tag）
2. 否则沿用现有 `logFieldRegex` + `projectMatchPattern`

CaptureRule 可增加可选 `preferDaydaymoneyMeta: true`（默认 true）。

## 10. 事件与意图

本迭代以**查询/配置同步**为主：

- 「同步项目标签」成功路径：可选投递领域事件 `PROJECT_TAGS_SYNCED_FROM_AIDEV`（若实现成本低则做；否则书面例外：纯配置 merge、无下游消费者）
- Resolve / parse-yaml 为只读查询 → 书面例外不投递 MQ

## 11. 架构交付物

- `docs/architecture/v38-application-integration-20260718-2340-claude.{puml,archimate,mermaid.md}`
- `VERSION_HISTORY.md` 增加 v38 target

## 12. 测试要点

- Go：resolve 多项目多工作空间；tag 大小写；空参 400；`?tag=` 列表过滤；parse-yaml 校验
- Vitest/JS：YAML 解析；tags merge；meta 读取
- Chrome：resolve 聚合 + 自动勾选
- Grafana：daydaymoney 精确匹配优先于 regex
- CI：抽样仓库存在合法 `daydaymoney.yaml`

## 13. 落地仓库清单（本 monorepo）

为下列顶层目录各放一份 `daydaymoney.yaml`（`service_id` = 目录名）：  
`task2app`, `taskProjectService`, `taskTaskService`, `taskAuth`, `taskChromePlugin`, `DaydaymoneyGrafana`, `taskGateway`, `taskSSE`, `taskEvents`, `taskBill`, `taskCloudService`, `taskContainerGateway`, `taskAiProvider`, `taskGitOauth`, `runAll`, `onlineServiceJS`（若存在）, `shareLib`, 以及仓库根 `daydaymoney.yaml`（`service_id: ram-work`）。

其余服务目录按同约定在实现阶段批量补齐。
