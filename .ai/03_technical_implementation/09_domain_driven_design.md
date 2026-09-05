# 领域驱动设计与领域事件规范

## 基本信息
- 版本：1.4.0
- 创建日期：2026-03-18
- 最后修改：2026-07-15
- 维护者：Trae AI 团队

## 规则分类

### 核心规则

#### 服务端开发领域驱动原则
- 描述：服务端开发**必须**采用领域驱动设计（DDD）与领域事件的方式，降低业务逻辑与底层依赖库的耦合；领域层禁止直连 ORM、Kafka、云 SDK 等基础设施实现（详见 `scripts/ci/check_ddd_bdd_compliance.py` 中的静态门禁）
- 适用场景：所有服务端代码开发，包括 API、业务逻辑、数据持久化等
- 优先级：强制（CI：`.github/workflows/ddd-bdd-compliance.yml`）
- 规则：
  - **领域模型优先**：业务逻辑应围绕领域模型组织，而非围绕数据库表或框架结构组织
  - **领域事件驱动**：跨聚合、跨模块的协作优先通过发布/消费领域事件实现，而非直接调用
  - **业务意图必发事件**：每当业务意图被接受（命令/用例成功走完一致性边界），必须经事件总线端口向消息队列投递对应业务/领域事件；禁止有意图无事件（纯查询等例外须书面标注）。文档与测试对照见 `.ai/08_prompt_management/01_intent_driven_development.md`
  - **解耦底层依赖**：业务层不应直接依赖具体的 ORM、消息队列、存储等实现，应通过领域接口或适配层隔离
  - **领域边界清晰**：按业务领域划分模块，同一领域内的逻辑内聚，领域间通过事件或明确接口通信
  - **基础设施隔离**：将数据库、消息队列、外部 API 等基础设施访问封装在适配器/仓储中，领域层仅依赖抽象
  - **进程间数据所有权**：跨服务禁止共享同一库/表的直连；一库或一表仅由一个拥有服务访问，他方经 API/领域事件协作或迁表划清 owner（见 `.ai/01_project_constraints/19_single_service_data_ownership.md`）
  - **新增接口落点**：新服务与新 HTTP/RPC 接口默认落 Go（先扩展现有 Go 服务，否则新建）；不得默认在 Django/Python 扩面（见 `.ai/01_project_constraints/20_go_service_first_apis.md`）

#### 领域事件 trace_id 传播
- 描述：跨服务领域事件须携带 `data.trace_id`，与 HTTP `X-Trace-Id` 对齐
- 规则：
  - Django：`send_event()` → `ensure_trace_in_event_data`（显式 → contextvars → `bg-*`）
  - HTTP 跨服务：Django delegate / taskAuth `djangoPost` 转发 `X-Trace-Id`
  - taskAuth 直发 MQ：`domainevents.PublishEvent` 必须注入 trace
  - Cron：`TraceContextCommand` → `cron-{trace_id}`（invocation 级共享）
  - 设计：`docs/superpowers/specs/2026-06-01-domain-events-trace-id-propagation-design.md`

## 规则冲突处理
- 当规则冲突时，遵循以下优先级：
  1. 核心规则 > 最佳实践 > 风格指南
  2. 文件级规则 > 目录级规则 > 全局规则
  3. 新版本规则覆盖旧版本规则

## 变更日志
- 2026-07-15：版本 1.4.0 - 增补「业务意图必发事件」：意图出现时须向 MQ 投递对应业务/领域事件
- 2026-07-13：版本 1.3.0 - 增补新增接口落点指针（默认 Go，见 `20_go_service_first_apis.md`）
- 2026-07-10：版本 1.2.0 - 增补进程间数据所有权指针（单库/单表单服务，见 `19_single_service_data_ownership.md`）
- 2026-06-01：补充领域事件 trace_id 传播规则
- 2026-05-08：版本 1.1.0 - DDD 升为强制级并接入 CI/pre-commit 静态校验
- 2026-03-18：版本 1.0.0 - 新增服务端开发领域驱动与领域事件规范
