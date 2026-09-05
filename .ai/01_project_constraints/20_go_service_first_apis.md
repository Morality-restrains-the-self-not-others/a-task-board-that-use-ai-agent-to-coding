# 新增服务 / 接口优先落 Go（扩展现有或新建）

## 基本信息

- 版本：1.1.8
- 创建日期：2026-07-13
- 最后修改：2026-08-16
- 维护者：Trae AI 团队

## 背景（为何是元规则）

monorepo 内 Django（`task2app` / saas-backend 等）在 runAll 下常为**单线程 WSGI**；长耗时、出站 HTTP、进程内 self-call 易造成整站假死。项目已持续将公网入口与独立限界上下文迁到 Go（见 `docs/superpowers/specs/2026-07-05-task2app-api-go-split-brainstorm-design.md` 与 `docs/architecture/` target）。

若新增业务 HTTP 接口或新服务默认落在 Python/Django/Flask 侧车，会：

- **扩大拥塞面**：单线程排队、锁与出站阻塞回传到主站
- **与 target 架构回退**：已拆出的 Go 边界再次被 Python 公网入口侵蚀
- **运维成本上升**：双栈接口、双份 Swagger/路由、数据所有权更易模糊

因此：**新增服务与新增接口的默认落点是 Go**——优先扩展职责匹配的现有 Go 服务；无匹配限界上下文时**新建 Go 服务**。在 Django / Python 侧车新增接口为**例外**，须可审计说明且经设计门禁。

## 规则分类

### 核心规则

#### 新增服务 / 接口默认落 Go

- **描述**：凡新增 **业务 HTTP/RPC 接口**、**对外能力入口**，或 **独立部署的新后端服务**，默认实现语言与进程形态为 **Go**。须先评估是否并入现有 Go 服务；无法并入时新建 Go 服务。**禁止**将「在 Django / Flask / 其他 Python Web 进程新增 endpoint」作为默认方案。
- **适用场景**：
  - 新功能设计、头脑风暴、价值流 / DDD 落点选型
  - 新增公网 API、网关后 handler、服务间 API、Webhook、长耗时/流式入口
  - 从 Django 拆出新限界上下文、新增 sidecar
  - Code review / 架构评审发现「本可落 Go 却落 Python」
- **优先级**：高
- **规则类型**：禁止忽略（核心规则）

##### 落点决策顺序（必须按序执行）

| 步骤 | 动作 | 完成判据 |
| --- | --- | --- |
| **1. 匹配现有 Go 服务** | 按限界上下文 / 数据 owner / 既有路由前缀，将接口放入职责已匹配的 Go 服务 | 新 handler 落在该服务仓库；表 owner 仍符合 [19_single_service_data_ownership.md](./19_single_service_data_ownership.md) |
| **2. 新建 Go 服务** | 无合适边界时新建 `taskXxxService`（或既有命名约定下的 Go 模块），接入 runAll/网关/观测与 Swagger | 独立进程 + 明确表/库 owner + OpenAPI 可见 |
| **3. Python 例外（最后）** | 仅当 Go 不可行且有书面理由时，才在 Django/Flask 新增接口 | 设计文档含「Python 例外」专节；通过 brainstorming 额外审批门禁 |

##### 现有 Go 服务（选型参考，非穷尽）

选型时以**限界上下文与数据所有权**为准，下表仅为快速对照：

| 服务目录 | 典型职责方向 |
| --- | --- |
| `taskAuth` | 认证、会话、内部鉴权相关 |
| `taskBill` | 计费 / 账单 |
| `taskCloudService` | 云资源相关能力 |
| `taskProjectService` | 项目域 |
| `taskTaskService` | 任务域 |
| `taskCredentialService` | 凭证 / token 真源相关 |
| `taskContainerGateway` | 容器网关 / 代理 |
| `taskEvents` | 事件消费与派发 |
| `taskAIEndPoint` / `taskAIComment` / `taskAgentSupport` | AI 相关入口与支撑 |
| `go_relayToTrae` / `go_run_container` | Trae 中继、容器运行侧车 |

网关（如 `taskGateway` / APISIX）只做路由与策略时，**业务 handler 仍须落在对应 Go 业务服务**，禁止「网关通过后首次实现落在新建 Django 视图」作为默认路径。

##### 明确禁止（默认路径）

| 禁止 | 说明 |
| --- | --- |
| 默认在 Django 新增公网 `@api_view` / urls 路由 | 须先完成上表步骤 1～2 |
| 默认新建 Flask/Python 侧车承载业务 API | 历史 Python 侧车已迁 Go 的路径不得回退 |
| 「先写 Django 以后再迁 Go」 | 未上线阶段（见 `00_project_constraints.md` 第 16 条）应直接采用目标形态 |
| 为省事在 Python 旁路同一张表 | 与单库/单表所有权冲突 |

##### 明确例外（须可审计）

以下可不强制新建 Go，但须在设计/意图文档中写明理由，且不得扩大为常规产品路径：

| 例外 | 说明 |
| --- | --- |
| **存量接口就地缺陷修复** | 仅修 bug / 安全补丁 / 契约兼容，不新增对外能力面 |
| **Django 作为已声明的 internal 真源** | 仅服务间 internal API，且 target 架构已明确该数据仍由 Django owner；公网入口仍须 Go/网关 |
| **框架/运维壳** | 健康检查、静态资源壳、管理命令、一次性迁移脚本（非常驻 HTTP 业务 API） |
| **经批准的 Python 新接口** | 设计文档含 Go 替代评估；brainstorming「Python 新增接口」门禁获准；记录拥塞与回退风险缓解 |

**非例外**：报表「临时接口」、管理后台旁路、Webhook、长轮询/SSE、容器/git 长耗时操作、新业务 CRUD 公网入口。

### 最佳实践

- 设计阶段先写 **接口 → 服务（Go）→ 表 owner** 三联对照；与 [19_single_service_data_ownership.md](./19_single_service_data_ownership.md)、Swagger 元规则一并满足。
- 扩展现有服务时复用该服务既有分层（domain / application / infrastructure）、日志与 OpenAPI 约定。
- 新建 Go 服务时同步：runAll 启动、端口配置、`shareLib` 鉴权/CORS/tracelog（若适用）、观测接入（见 `.ai/03_technical_implementation/12_observability_log_shipping.md`）、网关路由；**禁止**业务进程内嵌 ticker（见 [51_no_service_internal_poll_loop.md](./51_no_service_internal_poll_loop.md)）；runAll 探活端口必须等于监听 SSOT（见 [52_runall_health_port_ssot.md](./52_runall_health_port_ssot.md)）。
- Review 检查清单：本 PR 是否新增 Python HTTP endpoint？若是，是否已证明步骤 1～2 不可行并附例外专节？

### 对照表与 CI（落地）

| 产物 | 路径 |
| --- | --- |
| 机器可读对照表 | `db/api_route_ownership.yaml`（`route_prefixes` + `django_baseline_routes`） |
| 人读摘要 | `docs/architecture/api-route-to-owner.md` |
| 扫描脚本 | `db/scripts/ci/check_django_new_api_routes.py`（经 `runAll/scripts/ci/check_ddd_bdd_compliance.py` 调用） |
| Go OpenAPI 漂移 | `db/scripts/ci/check_go_openapi_routes.py` + `go_openapi_services.yaml`（taskBill/taskAuth/taskProjectService/taskCloudService/taskTaskService/taskAIComment；CI job `go-openapi` 使用 `--fail-on-extra`） |

- **新增** Django `path`/`re_path`/`url` 字面量若不在 baseline 且未登记 `approved_python_exceptions`，CI **失败**。
- **存量** Django 公网 API 的迁出节奏与优先级以 `docs/superpowers/specs/2026-07-05-task2app-api-go-split-brainstorm-design.md` 为准；本规则与 CI **不强制**一次性迁完存量。
- **Go OpenAPI**：已登记服务的 `mountRoutes` 须被 `openapi.yaml` 覆盖；OpenAPI 残留路径 CI **硬失败**（`--fail-on-extra`）。`openapi_internal` + `internal_coverage: complete`（taskAuth / taskCloudService / taskBill）：skip 前缀挂载须全部入 internal。运维入口：`/gateway/ops/docs/`（token/forward-auth），**不**混入公开 `/gateway/docs/`。

### 与其他规则的关系

| 规则 | 关系 |
| --- | --- |
| [19_single_service_data_ownership.md](./19_single_service_data_ownership.md) | Go 落点不得导致跨服务直连共享表；无 owner 匹配时新建服务并迁表或转发 |
| [51_no_service_internal_poll_loop.md](./51_no_service_internal_poll_loop.md) | 新建/扩展 Go 服务禁止在 `main` 内嵌 ticker；周期工作走 taskEvents timer worker |
| [52_runall_health_port_ssot.md](./52_runall_health_port_ssot.md) | 新建/扩展服务的 runAll `health_check` 端口必须等于监听 SSOT；禁止复制相邻条目 |
| [33_new_service_runall_registration.md](./33_new_service_runall_registration.md) | 新服务必须同步注册到 runAll（build/start/stop/health_check） |
| `.cursor/rules/swagger-api-governance.mdc` | Go 新接口同样必须 Swagger/OpenAPI 可见 |
| `.claude/skills/1-brainstorming-design-docs/SKILL.md` | 若仍选 Python，触发「Python 新增接口清单与 Go 替代评估」额外审批 |
| `docs/superpowers/specs/2026-07-05-task2app-api-go-split-brainstorm-design.md` | 存量 Django API 的 Go 化优先级与边界参考 |

## 变更日志

- 2026-08-16：1.1.8 - 交叉引用元规则 47：新建 Go 服务的 runAll 探活端口必须等于监听 SSOT。
- 2026-08-16：1.1.7 - 交叉引用元规则 46：新建 Go 服务禁止进程内 ticker，周期工作走 taskEvents timer worker。
- 2026-07-16：1.1.6 - Cloud/Bill `openapi-internal` 按 handler 补齐 requestBody/query/response schema；taskBill 计费查询分页与 OpenAPI 一并纳入 `feat/gitlab-traffic-pricing`。
- 2026-07-16：1.1.5 - Cloud internal 全量文档化并 `internal_coverage: complete`；taskBill 拆分 `openapi-internal` + `docsInternal` 接入 ops 门户。
- 2026-07-16：1.1.4 - Cloud 已文档化 internal 迁入 `openapi-internal.yaml`（partial）；taskBill 消除重定义并收紧 optional；网关受保护 ops 门户 `/gateway/ops/docs/`。
- 2026-07-16：1.1.3 - 登记 taskCloudService/taskTaskService/taskAIComment；taskAuth 拆分 `openapi-internal.yaml`（运维可见、不进门户）；taskBill 改为 optional。
- 2026-07-16：1.1.2 - 泛化 Go OpenAPI 漂移检查（taskBill/taskAuth/taskProjectService）；CI 改为 `--fail-on-extra`。
- 2026-07-16：1.1.1 - 增加 taskBill `mountRoutes`↔`openapi.yaml` CI（`check_taskbill_openapi_routes.py`，含 stale-doc warn）。
- 2026-07-13：1.1.0 - 落地 `db/api_route_ownership.yaml`、人读对照表与 Django 新增路由 CI；明确存量迁出节奏文档指针。
- 2026-07-13：1.0.0 - 初版：确立新增服务/接口默认落 Go；先扩展现有 Go 服务，否则新建；Python 为可审计例外。
