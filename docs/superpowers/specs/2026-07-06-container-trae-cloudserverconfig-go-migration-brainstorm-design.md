# 设计文档：容器 / Trae / CloudServerConfig 迁移 Go（Phase 3 深化）

- **版本**: v1.1 草案（用户修订 2026-07-06）
- **作者**: claude
- **日期**: 2026-07-06 19:30（修订 19:45）
- **迭代**: Phase 3 taskCloudService 深化 — 消除 AI 评论与容器运行时对 Django 的依赖
- **前置**:
  - v8 Phase 2 已交付（taskTaskService / taskAIComment / Todo 零残留）
  - taskAIComment 真实现中 **校验与 instruct 执行仍曾短暂委托 Django internal**（过渡实现，**本设计 v1.1 起废止**）
  - v6 设计将 `CloudServerConfig` 归属 **taskCloudService (:8018)**，当前 Go 侧多为 stub
  - **container_access_token** 已迁 **taskCredentialService (:8015)** SSOT

---

## 1. 背景与动机

### 1.1 为何现在做

| 现状问题 | 影响 |
|---------|------|
| `CloudServerConfig` 仍在 Django PostgreSQL/SQLite ORM | taskAIComment POST 校验、容器转发、relay 均 HTTP 回 `:8001`，**单线程 WSGI 拥塞面未收敛** |
| Trae/legacy instruct 流在 `projects/services/ai_instruct_stream_service/`（Python） | 长轮询 + 出站 HTTP + SSE/Kafka 编排占 Django 线程；与 Go 化方向冲突 |
| `cloud/views/` 约 26 文件、Aliyun provider 约 13 文件仍在 Django | v6 Phase 3 Gap 未关闭；APISIX 大量 `/cloud/*` 路由仍指向 saas-backend |
| taskCloudService 已占位 :8018 但 **CloudServerConfig CRUD / 实例 LCM 未落地** | 架构文档与实现漂移 |

### 1.2 成功标准（SMART）

1. **数据真源**：`CloudServerConfig` + `CloudServerConfigHistory` 读写仅经 **taskCloudService** SQLite/PostgreSQL（与 taskProject/taskTask 同模式）
2. **运行时解析**：`resolve_container_target`（server_url + token）在 **Go** 完成，调用 taskCredentialService
3. **AI instruct 路径**：taskAIComment POST → **Go 同步校验**（server_url + Trae 锚点）→ Go 落库 → **Go 后台 worker** 拉容器/Trae 流 → taskSSE + Kafka（**不再**调 Django `start-instruct-stream`）
4. **公网 cloud API**：APISIX 切流至 :8018；Django `cloud/views/` 删除
5. **切流窗口**：migration 导出 Django 行 → Go import → DROP Django 表（对齐 0053/0054 模式）

---

## 2. 架构基线确认

根据 `docs/architecture/` 与 `VERSION_HISTORY.md`，当前理解如下：

> **系统现状摘要**
>
> - **current 基线（文件）**：v1 enterprise/application（2026-07-01）；**交付态**以 v7/v8 target 为准（Phase 1/2 已落地）
> - **v8 target**：taskTaskService + taskAIComment 真源；**Gap Phase 3 taskCloudService 仍 OPEN**
> - **应用层（云域）**：
>   - `taskCloudService` Go :8018 — 授权/OAuth stub，Aliyun SDK pending
>   - `taskCredentialService` :8015 — container token SSOT
>   - `taskContainerGateway` :8014 — 容器层 graph/clone/layer 代理
>   - `go_relayToTrae` :8797 — relay 生命周期
>   - `onlineServiceJS` — Trae 业务容器 API（task-gate、jobs、interrupt）
>   - `saas-backend` Django :8001 — **仍持有 CloudServerConfig ORM + cloud/views + instruct 流**
> - **数据流（AI 评论，Phase 3b/3c 已收口）**：
>   ```text
>   公网 POST ai-comments → APISIX → taskAIComment (:8019)   ← 唯一公网入口
>     ├─ 同步校验 → taskCloudService validate（仅 Go）
>     ├─ SQLite 落库
>     └─ 异步 instruct → taskAIComment 同进程 instruct_worker
>          → taskCloudService lookup + taskCredentialService token
>          → onlineServiceJS | legacy /ai/instructs/
>          → taskSSE + Kafka → taskEvents → PATCH taskAIComment
>   ```
>
> **Instruct 流归属澄清（回应架构疑问）**：
>
> | 维度 | 说明 |
> |------|------|
> | **谁对外提供 ai-comments？** | 始终是 **taskAIComment**（APISIX → :8019），不是 Django |
> | **谁执行容器/Trae 拉流？** | **taskAIComment 内 instruct_worker**（同进程 goroutine） |
> | **CloudServerConfig 从哪读？** | **taskCloudService**；POST 校验已无 Django 回退（Phase 3b ✅） |

📋 **架构版本历史（相关）**

| 版本 | 状态 | 要点 |
|------|------|------|
| v6 | 🎯 target | 定义 taskCloudService 范围含 CloudServerConfig |
| v7 | 🎯 部分交付 | 项目域 Go 真源 |
| v8 | 🎯 Phase 2 交付 | 任务域 + taskAIComment；Phase 3 待续 |

---

## 3. 范围边界

### 3.1 纳入本次迁移

| 模块 | Django 现状 | Go 目标 |
|------|------------|---------|
| **CloudServerConfig** | `cloud/models/cloud_server_config.py` | taskCloudService 表 + CRUD API |
| **CloudServerConfigHistory** | 同上 | taskCloudService 历史表 + 查询 API |
| **CloudServerConfigDefault** | workspace 默认配置 | taskCloudService（读 taskProjectService workspace） |
| **container_target_resolver** | Python，读 ORM + Credential Go | taskCloudService `GET .../container-target` internal |
| **detect_container_instruct_stream_backend** | Python probe task-gate | taskCloudService 或共享 `pkg/containerinstruct` |
| **iter_container_instruct_response** | Python Trae poll + legacy stream | **taskInstructWorker**（见 §5.2） |
| **ai_comment_post_validation** | Django internal validate-post | **删除**；taskAIComment → taskCloudService **唯一路径** |
| **ai_instruct_stream_consumer** | Django 后台线程 | **删除**；taskAIComment `instruct_worker`（Phase 3c） |
| **cloud/views/** 公网路由 | ~26 文件 | taskCloudService handlers + APISIX 切流 |
| **cloud/providers/aliyun/** | Python SDK | aliyun-sdk-go（v6 已规划） |

### 3.2 明确不纳入（或后续独立迭代）

| 模块 | 理由 | 归属 |
|------|------|------|
| LLM Budget / TaskModelBudget | v6 已划 taskAIEndPoint | :8013 |
| container-layer-* 大量转发 | 已在 taskContainerGateway | :8014 |
| relay-to-trae 生命周期 | go_relayToTrae + taskEvents 已 Go 化 | :8797 / events |
| accounts/company internal | Django 真源保留 | :8001 internal |
| Git OAuth / layer github tokens | 跨 git-oauth + Django 编排 | 单独迭代 |

### 3.3 依赖关系（迁移后）

```text
taskAIComment (:8019)  ← 公网 ai-comments 唯一入口 + instruct 编排真源
  ├─ POST 同步校验 → taskCloudService validate-ai-comment-post/（仅 Go，无 Django 回退）
  ├─ POST 落库 SQLite
  └─ async instruct_worker（Phase 3c，同进程 goroutine）
        ├─ taskCloudService: lookup + container-target + stream-backend probe
        ├─ taskCredentialService: token by scope
        ├─ onlineServiceJS | legacy container HTTP
        ├─ taskSSE: publish chunks
        └─ Kafka: AI_ASSISTANT_REPLY_COMPLETED

taskCloudService (:8018)
  ├─ CloudServerConfig CRUD (公网 + internal)
  ├─ Instance LCM / Aliyun API
  ├─ taskProjectService: workspace 默认配置
  ├─ taskTaskService: loose task_id 存在性
  └─ taskCredentialService: token 读写（不 duplication）

APISIX
  └─ /api/tenant/*/workspace/*/task/*/cloud/* → :8018（除已切 Gateway 的 layer 路径）
```

---

## 4. 领域概念清单（供 /6-ddd 消费）

### 4.1 Bounded Context

| 上下文 | 聚合根 | 说明 |
|--------|--------|------|
| **Cloud Runtime** (taskCloudService) | `CloudServerConfig` | 任务级云实例运行时快照（server_url、region、instance_id…） |
| | `CloudServerConfigHistory` | 会话级历史 |
| | `CloudPlatformAuthorization` | 租户云平台授权（已有 stub） |
| **Container Instruct** (**taskAIComment** 子模块) | `InstructSession` | 一次 AI 评论对应的容器/Trae 执行会话；**不**再归属 Django |
| **AI Comment** (taskAIComment) | `AITaskComment` | 已交付；仅移除对 Django 的回调 |

### 4.2 领域事件（保持契约）

| 事件 | 生产者 | 消费者 |
|------|--------|--------|
| `AI_ASSISTANT_REPLY_COMPLETED` | instruct worker (Go) | taskEvents → PATCH taskAIComment |
| `RELAY_LIFECYCLE` | relay / gateway | 已有 taskEvents handlers |
| Cloud 实例状态变更 | taskCloudService | taskTaskService PATCH（可选） |

### 4.3 跨上下文术语

- **server_url**：容器 HTTP 根；Trae 时指向 onlineServiceJS
- **stream_backend**：`trae` | `legacy`（task-gate 探测结果）
- **container_job_context**：`parent_job_id` / `repo_layer_id` / `command_kind`（Trae skill.md 约束）

---

## 5. 分阶段实施计划

> 原则：**垂直切片 + 切流窗口**；**禁止 Strangler 双读**（Go 与 Django ORM 不同时作为运行时真源）。

### Phase 3a — CloudServerConfig 数据真源（2–3 天）

| 步骤 | 交付物 |
|------|--------|
| SQLite schema + migration | `task_cloud.db` 表 `cloud_server_configs`, `cloud_server_config_histories` |
| CRUD HTTP | `GET/POST/PATCH` 租户/workspace/task 维度配置 |
| import + 切流 | `cutover_cloud_configs` + Go `POST /api/internal/cloud-server-config/import/` |
| **切流前置** | 生产/测试环境 CloudServerConfig **必须先 import 至 Go**，再启用仅 Go 读路径 |
| APISIX | 先 internal，公网 shadow |
| 测试 | Go table tests + 迁移对账；pytest **仅**向 Go seed（`cloud_server_config_client.import`） |

**验收**：运行时 **无** Django `CloudServerConfig.objects` 读路径；ORM 仅保留至 Phase 3e DROP。

### Phase 3b — container-target 与校验下沉 + **删除 Django 校验代码**（1–2 天）

| 步骤 | 交付物 |
|------|--------|
| Go `ResolveContainerTarget` / `DetectStreamBackend` / `ValidateAICommentPost` | 已在切片 1 部分落地 |
| taskAIComment | **仅**调 taskCloudService validate；**移除** `validatePostViaDjango` 与 Django fallback |
| **删除 Django 代码** | 见 §6.5 删除清单 |
| 测试 | `test_ai_comment_post_validation.py` 改为 Go HTTP 集成测或删除；POST 400 行为不变 |

**验收**：grep 无 `validate-post`、`ai_comment_post_validation` 运行时引用；taskAIComment 502 时 **fail fast**（不回退 Django）。

### Phase 3c — Instruct 流式 Worker 迁入 **taskAIComment**（3–5 天，关键路径）

| 步骤 | 交付物 |
|------|--------|
| **归属** | **taskAIComment 同进程 `instruct_worker` 包**（公网已在 :8019，执行体也归此服务） |
| Trae 客户端 | `POST /api/jobs`、poll output、`POST .../interrupt`（对齐 onlineServiceJS/skill.md） |
| Legacy 客户端 | `POST .../ai/instructs/` 字节流 |
| SSE 发布 | HTTP → taskSSE（Go client） |
| Kafka 发布 | `AI_ASSISTANT_REPLY_COMPLETED`（对齐 envelope） |
| taskAIComment | **删除** `startInstructStreamAsync` → Django；改为本地 `go instruct_worker.Run(...)` |
| **删除 Django 代码** | `start_instruct_stream`、`ai_instruct_stream_consumer.py`、`ai_instruct_stream_service/` |
| 测试 | mock onlineServiceJS + pytest 流式用例 **不依赖** :8001 |

**验收**：`test_post_ai_task_comment_creates_row_and_listable` 在 **saas-backend 未启动** 时仍可 PASS（仅依赖 :8019/:8018/:8015/:8798/Kafka）。

### Phase 3d — cloud/views 公网切流（5–8 天，可与 3a 并行后半）

| 步骤 | 交付物 |
|------|--------|
| 按 v6 清单逐 ViewSet 移植 | 授权、网络、镜像、compute、userdata… |
| Aliyun SDK | `aliyun-sdk-go` 替换 `cloud/providers/aliyun/` |
| APISIX 路由 | `task-cloud-service` priority 覆盖 Django cloud 路径 |
| runAll | depends_on 链更新 |

**验收**：`cloud/view_test/*` 改指向 :8018；Playwright 云启动路径 smoke。

### Phase 3e — Django ORM 删除与 migration（切流窗口）

| 步骤 | 交付物 |
|------|--------|
| migration | 类似 0053：`RunPython` export → Go import → `DROP cloud_cloudserverconfig*` |
| 管理命令 | `cutover_cloud_configs`（对齐 `cutover_ai_comments` 模式） |
| 删除 | `cloud/models/cloud_server_config*.py`、残留 views/services |

### Phase 3f — 架构 v9 与文档收口

| 步骤 | 交付物 |
|------|--------|
| v9 六件套 | enterprise + application × (puml + archimate + mermaid) |
| VERSION_HISTORY | 关闭 Gap Phase 3；Plateau v9 |
| value-stream.yaml | `saas-backend.cloud_*` → `task-cloud-service.*` |
| 设计文档 §16/§17 交叉引用更新 |

---

## 6. 关键设计决策

### 6.1 Instruct Worker 放哪？

| 方案 | 优点 | 缺点 | 推荐 |
|------|------|------|------|
| **A. taskAIComment 内嵌 instruct_worker** | 与公网 ai-comments 同服务；comment_id 本地；少一跳 | 服务职责变宽（存储 + 流式编排） | ⭐ **唯一推荐** |
| B. 独立 taskInstructService | 边界清晰 | 多端口、与 :8019 双跳 | 仅当 instruct 复用到非 AI 评论场景时再议 |
| ~~C. 留 Django internal~~ | — | 单线程拥塞、架构回退 | ❌ **废止**（用户 v1.1 决策） |

### 6.5 Go 唯一真源 — **废止 Strangler 双读**（用户 v1.1 决策）

| 原则 | 说明 |
|------|------|
| **运行时** | CloudServerConfig 与 POST 校验 **仅**经 taskCloudService Go；taskAIComment **禁止**回退 Django `validate-post` |
| **切流** | 启用仅 Go 路径前，**必须**完成 export → import → verify；测试用 `cloud_server_config_client.import`，**不**依赖 Django ORM 镜像 |
| **失败策略** | taskCloudService 不可达 → taskAIComment 返回 **502**（与 taskProject 切流后一致），**不**静默回退 ORM |
| **代码删除** | 下列 Django 模块在对应 Phase **物理删除**（非 deprecated）： |

**Phase 3b 删除清单（校验域）**

| 路径 | 说明 |
|------|------|
| `projects/task_ai_comment/internal_views.py` → `validate_post` | 整函数删除 |
| `projects/urls_task_ai_comment_internal.py` → `validate-post/` | 路由删除 |
| `projects/services/ai_comment_post_validation.py` | 文件删除 |
| `tests/test_ai_comment_post_validation.py` | 迁至 Go 集成测或删除 |
| `taskAIComment/src/cloud_client.go` → `validatePostViaDjango` | 删除 fallback 分支 |

**Phase 3c 删除清单（instruct 域）**

| 路径 | 说明 |
|------|------|
| `projects/task_ai_comment/internal_views.py` → `start_instruct_stream` | 整函数删除 |
| `projects/urls_task_ai_comment_internal.py` → `start-instruct-stream/` | 路由删除 |
| `projects/services/ai_instruct_stream_consumer.py` | 文件删除 |
| `projects/services/ai_instruct_stream_service/` | 目录删除 |
| `taskAIComment/src/django_client.go` → `startInstructStreamAsync` | 改为 `instruct_worker` |

**Phase 3e 删除清单（数据域）**

| 路径 | 说明 |
|------|------|
| `cloud/models/cloud_server_config.py` | migration DROP 后删除 |
| `cloud/services/container_target_resolver.py` | 逻辑已在 taskCloudService |

### 6.6 Instruct 流与 taskAIComment 的关系（架构 FAQ）

```text
┌─────────────┐     POST ai-comments      ┌──────────────────┐
│   Client    │ ────────────────────────► │  taskAIComment   │  :8019 公网
└─────────────┘                           │  (编排 + 存储)    │
                                          └────────┬─────────┘
                                                   │
                     ┌─────────────────────────────┼─────────────────────────────┐
                     │ 同步                         │ 异步 (Phase 3c)              │
                     ▼                             ▼                             │
            ┌─────────────────┐          ┌─────────────────────┐                  │
            │ taskCloudService│          │ instruct_worker     │  ← 同进程，非 Django │
            │ validate/lookup │          │ (Trae/legacy 拉流)  │                  │
            └─────────────────┘          └──────────┬──────────┘                  │
                                                      │ SSE + Kafka                 │
                                                      ▼                             │
                                            taskSSE / taskEvents                    │
```

**常见误解**：「instruct 还在 Django」指的是 **当前过渡实现** 把执行委托给了 Django Python 线程；**目标架构** 下 instruct **仍从 taskAIComment 触发**，只是执行体从 Django 改为 **taskAIComment 内 Go worker**。

### 6.2 CloudServerConfig 与 Credential 分工

- **taskCloudService** 存：server_url、business_api_endpoint、instance 元数据、region…
- **taskCredentialService** 存：access/refresh token、expires（**禁止**回写 Django ORM 字段）
- `resolve_container_target`：**永远** token 来自 Credential Go；URL 来自 CloudServerConfig Go

### 6.3 校验同步 vs 异步

- 维持 **POST 前同步 400**（已在 taskAIComment 实现，数据源改为 taskCloudService）
- 流式错误仍走 SSE error phase + Kafka（行为不变）

### 6.4 数据库

| 环境 | 策略 |
|------|------|
| dev | SQLite `db/task-cloud/task_cloud.db`（与现 taskProject/taskTask 一致） |
| prod | PostgreSQL `task_cloud_db` 或 SQLite 单文件（与运维约定一致） |

---

## 7. 数据迁移与切流

### 7.1 切流窗口脚本（规划）

```bash
# 1. 前置
python3 manage.py cutover_cloud_configs --step preflight   # TCS :8018 + TCS health

# 2. 备份
python3 manage.py cutover_cloud_configs --step export --json /backup/cloud_configs.json

# 3. 导入 Go
python3 manage.py cutover_cloud_configs --step import --json /backup/cloud_configs.json

# 4. migrate（RunPython 内亦可 import，与 0053 相同模式）
python3 manage.py migrate cloud 00XX_drop_cloud_server_config_django_table

# 5. 对账
python3 manage.py cutover_cloud_configs --step verify
```

### 7.2 与 taskAIComment 切流顺序

1. Phase 3a：**import CloudServerConfig → Go**（切流 verify PASS）
2. Phase 3b：**删除 Django 校验** + taskAIComment 仅 Go validate
3. Phase 3c：**instruct_worker 迁入 taskAIComment** + 删除 Django stream 代码
4. Phase 3e → DROP Django cloud 表

**禁止**：在 Go 数据未 import 前启用仅 Go 读路径；**禁止**：保留 Django validate/instruct 作为运行时 fallback。

---

## 8. 🐍 Python 新增接口清单与 Go 替代评估

> 本迭代目标是 **净减少** Python 接口；过渡期允许 **0 个新公网** endpoint。

### 8.1 拟废弃（交付后删除）

| 方法 | 路径 | 归属 | 替代 |
|------|------|------|------|
| POST | `/api/internal/task-ai-comment/validate-post/` | saas-backend | taskCloudService internal 或 taskAIComment 内置 |
| POST | `/api/internal/task-ai-comment/start-instruct-stream/` | saas-backend | taskAIComment Go worker |
| * | `/api/tenant/.../cloud/*`（~40+ 路由） | saas-backend | taskCloudService :8018 |

### 8.2 过渡期内允许（internal-only，仅 migration）

| 方法 | 路径 | 用途 | 退役条件 |
|------|------|------|----------|
| POST | `/api/internal/cloud-server-config/import/` | 一次性/测试 seed | 长期保留（幂等 upsert） |

> **废止**：`health-django-bridge`、Django `validate-post` / `start-instruct-stream` — 不再作为过渡接口。

### 8.3 Go 替代方案摘要

| 能力 | 目标 | 推荐 |
|------|------|------|
| CloudServerConfig CRUD | taskCloudService | ⭐ |
| container-target | taskCloudService | ⭐ |
| instruct stream | taskAIComment worker | ⭐ |
| SSE publish | taskSSE HTTP client from Go | ⭐ |
| Kafka event | 现有 Go producer 模式（taskEvents 已有） | ⭐ |

**选型结论（草案）**：

- **最终选择**：上述 Go 方案；Django cloud 公网 **全部退役**
- **Swagger**：新接口在 taskCloudService 侧维护（网关聚合或独立 OpenAPI）
- **python_api_approval**: `scoped-down`（无新增 Python 公网；仅可选 internal 导入）

---

## 9. 价值流影响

对照 `conf/value-stream.yaml`，主要影响流：

| 流 | 变更 |
|----|------|
| `cloud-integration` | fields 前缀 `saas-backend.cloud_*` → `task-cloud-service.*` |
| `container-token-go-migration` | 完成后 Django 不再查 CloudServerConfig ORM |
| `container-runtime-context` | server_url 字段迁至 task-cloud-service |
| `task-container-gateway` | resolve 改调 taskCloudService（非 Django ORM） |
| AI 评论相关（Inc-8 已部分 Go） | 去除 Django instruct 依赖 |

完整切片与 YAML 更新在 **/4-value-stream** 阶段执行；本节仅标识影响面。

---

## 10. 🏛️ 架构变更影响（v9 target 规划）

> 用户批准本设计后，按 `/1-brainstorming` 规则创建 **v9 六件套**（非本草案范围，此处仅规划）。

| 项 | 内容 |
|----|------|
| **迭代版本** | v9 🎯 target |
| **迭代名称** | Cloud 域 Go 真源 + Instruct Worker 下沉 |
| **Plateau v8 → v9** | 关闭 Gap Phase 3；taskCloudService live |
| **🟢 NEW** | taskCloudService CloudServerConfig 真源；taskAIComment instruct worker |
| **🟡 MODIFIED** | taskAIComment 不再依赖 Django internal；APISIX cloud 路由 → :8018 |
| **🔴 DEPRECATED** | Django `cloud/views/`、`cloud/providers/aliyun/`、instruct Python 包 |

### .archimate 架构变迁要点（规划）

| 元素 | 内容 |
|------|------|
| Plateau v8 | Phase 2 交付；AI 评论流仍回 Django |
| Gap | CloudServerConfig + instruct 在 Django |
| WorkPackage | WP Phase 3a–3f |
| Plateau v9 | Cloud 真源 + AI instruct 全 Go |
| 视图 | `架构变迁 v8→v9 — Cloud/Instruct Go` |

---

## 11. 风险与缓解

| 风险 | 缓解 |
|------|------|
| Trae poll 语义与 Python 不一致 | 共享 golden test vectors；对照 onlineServiceJS skill.md |
| SSE/Kafka envelope 漂移 | 复制现有 pytest 断言；contract test |
| Aliyun SDK 移植遗漏 | Phase 3d 逐 API 对照；保留 Python 参考至 canary 结束 |
| 双写窗口数据不一致 | **切流窗口 import + verify**；废止 Strangler，不做双读 |
| taskAIComment 职责膨胀 | 3c 后评估是否拆 taskInstructService |

---

## 12. 验收清单

| # | 检查项 |
|---|--------|
| 1 | taskCloudService CloudServerConfig CRUD + import 幂等 |
| 2 | taskAIComment POST 400 **仅**经 taskCloudService；**无** Django validate-post |
| 3 | AI 评论流式回复经 **taskAIComment instruct_worker** → SSE + Kafka（saas-backend 可不启动） |
| 4 | APISIX cloud 路由指向 :8018；Django cloud 公网 404/503 |
| 5 | migration DROP Django cloud 表；verify 无残留 |
| 6 | v9 架构制品 + value-stream 前缀更新 |

---

## 13. 建议执行顺序（/goal 或 /0-auto-flow）

```text
Week 1: Phase 3a + 3b（数据 + 校验）
Week 2: Phase 3c（instruct worker，最高风险）
Week 3–4: Phase 3d（cloud views 分批）
Week 5: Phase 3e + 3f（切流 + 架构 v9）
```

**立即可启动的垂直切片（推荐第一个 PR）**：

> Phase 3b 收口：移除 Strangler fallback + 删除 Django 校验代码（前置：Go import 已就绪）

**Phase 3b 收口（2026-07-06）**：✅

| 项 | 状态 |
|----|------|
| taskAIComment validate **仅** taskCloudService | ✅ 无 Django fallback |
| 删除 Django `validate-post` / `ai_comment_post_validation.py` | ✅ |
| 删除 `test_ai_comment_post_validation.py` | ✅（覆盖在 taskCloudService Go tests） |

**Phase 3c 收口（2026-07-06）**：✅

| 项 | 状态 |
|----|------|
| taskAIComment `instruct_worker` 同进程 goroutine | ✅ legacy + Trae jobs 拉流 |
| SSE → taskSSE `/internal/publish` | ✅ |
| Kafka `AI_ASSISTANT_REPLY_COMPLETED` | ✅ topic `ai-assistant-reply-completed` |
| 删除 Django `start-instruct-stream` / consumer / stream_service | ✅ |
| `validate_ai_comment.go` 使用 `resolveContainerTarget` baseURL | ✅（task-gate 探测修复） |

**Phase 3a 收口（2026-07-06）**：✅

| 项 | 状态 |
|----|------|
| `cloud_server_configs` + `cloud_server_config_histories` schema | ✅ |
| GET/POST/PATCH server-config + histories 查询 | ✅ |
| internal container-target / import-histories | ✅ |
| `manage.py cutover_cloud_configs` | ✅ |
| migration `cloud/0049` export → Go → DROP | ✅ |

**Phase 3d 收口（2026-07-06）**：✅（核心读路径）

| 项 | 状态 |
|----|------|
| APISIX workspace/task cloud → :8018 | ✅ |
| Go `container-task-ui-context` / server-start-history / previous-server-config | ✅ |
| Aliyun SDK 全量 compute LCM | ⚠️ stub（`aliyun-sdk-go integration pending`） |

**Phase 3e 收口（2026-07-06）**：✅

| 项 | 状态 |
|----|------|
| `cutover_cloud_configs` + migration 0049 | ✅ |
| Go facade `cloud_server_config_go.py` | ✅ |
| runtime context repository → Go lookup | ✅ |

**Phase 3f 收口（2026-07-06）**：✅

| 项 | 状态 |
|----|------|
| v9 六件套 | ✅ `docs/architecture/v9-*-20260706-2045-claude.*` |
| VERSION_HISTORY v9 | ✅ |
| value-stream cloud-integration 前缀 | ✅（`task-cloud-service.cloud_*`） |

---

## 14. 变更记录

| 日期 | 变更 |
|------|------|
| 2026-07-06 | 初稿：响应 taskAIComment 交付后「容器/Trae/CloudServerConfig 迁 Go」规划需求 |
| 2026-07-06 v1.1 | **用户修订**：废止 Strangler 双读，校验/配置仅 Go 并删除 Django 相关代码；澄清 instruct **公网与编排归 taskAIComment**，Phase 3c 迁移的是执行体（Django Python → :8019 instruct_worker） |
