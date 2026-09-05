# Completed OPT Archive — 2026-07-28

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 11 条。
> 归档执行时间：2026-07-29T01:08:43+08:00

## [OPT-20260728-001] completed

- **Status**: completed
- **Completed**: 2026-07-28
- **Created**: 2026-07-28
- **Context**: 当前 taskFE 中存在多种 API 错误消息提取实现：(1) `apiUtils.js` 的 `extractErrorMessage()` 被少数调用方使用；但内联 `result.message || result.error || result.detail` 模式遍布 10+ 个文件。本次修复已将 8 个文件的导入和调用统一为 `extractErrorMessage(result, response, 'fallback')`，并增强了函数签名（增加 fallback 参数 + 字符串类型检查 + trim 空值过滤）。
- **Related**:

## [OPT-20260728-002] completed

- **Status**: completed
- **Completed**: 2026-07-28
- **Created**: 2026-07-28，包含修改前的 `rawMessage` 提取逻辑（不含 `non_field_errors`）。源代码已修复但生产环境依赖构建产物。
- **Action**: 运行 `npm run build`（或项目对应的构建命令）重新生成 `app/static/assets/` 下的 minified JS；确认新 bundle 中包含 `non_field_errors` 处理；部署后验证登录错误提示。
- **Why**: 源代码修复需要反映到构建产物才能在生产环境生效。
- **How to apply**: 1) 在 taskFE 目录执行构建；2) 在 minified 输出中 grep 验证 `non_field_errors` 关键词存在；3) 部署并 E2E 验证登录失败场景。
- **Related**:

## [OPT-20260728-013] completed

- **Status**: pending
- **Created**: 2026-07-28
- **Context**: G5 Go 迁移完成后，Django `core/kafka/handlers/` 目录已删除，但 `core/paths_loader.py` 中的 `kafka_handlers_dir()` 函数仍存在（指向已删除的目录），且 `paths.conf` 中可能有 `KAFKA_HANDLERS_DIR` 存量配置。
- **Action**: 从 `core/paths_loader.py` 移除 `kafka_handlers_dir()` 函数；从 `paths.conf` 移除 `KAFKA_HANDLERS_DIR` 条目；更新唯一引用点 `test_user_activation_welcome_event.py` 中的测试逻辑。
- **Why**: 保留指向不存在目录的配置函数增加困惑和误用风险。
- **How to apply**: 搜索 `kafka_handlers_dir` / `KAFKA_HANDLERS_DIR` 全仓库引用，逐一清理。

## [OPT-20260728-012] completed

- **Status**: pending
- **Created**: 2026-07-28
- **Context**: `conf/events/domain-events/member_joined/config.yaml` 定义了 Kafka topic（`topic: member-joined`），但项目中不存在任何消费者（Go 和 Django 均无）。该事件在 Django 中也无 `send_event("MEMBER_JOINED", ...)` 的调用点。
- **Action**: 确认 MEMBER_JOINED 业务需求后：a) 若有需求 → 实现 Go 消费者；b) 若无需求 → 删除 `conf/events/domain-events/member_joined/` 配置和 `KAFKA_TOPICS` 中对应条目。
- **Why**: 孤儿 topic 配置造成架构理解困难，增加新成员不必要的猜测。
- **How to apply**: 与产品确认成员加入通知需求，决定实现或删除。

## [OPT-20260728-011] completed

- **Status**: pending
- **Created**: 2026-07-28
- **Context**: Go cmd 中存在无对应 `conf/events/domain-events/` 配置的二进制：`cmd/authorize_sg_ingress`（编译错误）、`cmd/task_created/*`、`cmd/task_deleted/*`、`cmd/user_created/2_sync_user_profile`。这些二进制要么是死代码（功能已被其他 intent 覆盖），要么是未完成的功能。
- **Action**: 对 task_created/task_deleted 的 fanout 二进制（功能已被 task_status_changed/2_fanout_work_panel_sse 覆盖）→ 删除；对 user_created/2_sync_user_profile → 补齐 config 或删除；对 authorize_sg_ingress → 删除（长期编译错误未维护）；运行 `go build ./cmd/...` 确保全部编译通过。
- **Why**: 死代码增加维护负担和困惑，编译失败的二进制可能阻塞 CI。
- **How to apply**: 逐个检查二进制功能，确认覆盖情况后删除或补齐 config。

## [OPT-20260728-003] completed

- **Status**: pending
- **Created**: 2026-07-28
- **Context**: 系统环境变量 `http_proxy`/`https_proxy`/`all_proxy` 均指向 `socks5h://127.0.0.1:1234`，该代理不稳定，会导致 Playwright 浏览器网络请求失败 (`net::ERR_PROXY_CONNECTION_FAILED`)。已在 `tmp/playwright-login-daydaymoney.py` 中通过 `os.environ.pop()` 在 import playwright 前清除代理变量修复。仓库内其他 Playwright Python 脚本（如 `DaydaymoneyGrafana/playwright/`、`runAll/playwright/`）若直接 `launch()` 也会遇到相同问题。
- **Action**: 创建共享 `playwright_proxy_bypass.py` 工具模块（或在每个 playwright 脚本入口统一清除代理变量 + 注释引用 memory [[httpclient-socks-proxy-bypass]]），确保所有 Playwright Python 脚本不受 SOCKS 代理干扰。
- **Why**: 避免每个新脚本独立踩坑重修复，统一代理 bypass 模式。
- **How to apply**: 在 repo 根创建 `scripts/playwright_utils.py`，包含 `bypass_socks_proxy()` 函数；搜索所有 `from playwright` 引用点补充调用；在 `CLAUDE.md` 中记录此约束。

## [OPT-20260728-007] completed

- **Status**: pending
- **Created**: 2026-07-28
- **Context**: `db/saas/init.sh` 中 `create-default-legal-documents` 调用 taskBill Go API，`03_03_repair_users_without_company.py` 调用 taskTenantService/taskProjectService Go API（workspace 创建）。如果这些 Go 服务未启动，初始化步骤静默失败或仅部分成功。当前 init.sh 使用 `set -euo pipefail`，但 `|| true` 后忽略错误。
- **Action**: 在 `db/saas/init.sh` 开头添加 Go 服务可用性预检（`wait_for_service` 函数轮询各服务 health 端点），超时则提示并退出，而非静默跳过。
- **Why**: 半初始化状态（schema 存在但 seed data 缺失）更难排查，应在初始化阶段尽早发现。
- **How to apply**: 参考 `03_02_init_tenant.py` 中的 `_check_taskauth_available()` 模式，为 taskBill、taskTenantService、taskProjectService 添加类似检查。

## [OPT-20260728-005] completed

- **Status**: pending
- **Created**: 2026-07-28
- **Context**: 修复 OPT-20260728-004 后，无公司用户登录后跳转到 `/system-admin/`，但该页面为空白（/me/ API 失败 + Vite WS 混合内容错误 + 页面内容为 0 字符）。用户 `contact@daydaymoney.com` 的 `companies: []` 为空，无任何租户/工作空间。现有路由缺乏独立的"创建公司/租户"onboarding 页面（`/tenant/:tenant/create-workspace/` 需要有 tenant）。
- **Action**: 新增无租户用户的 onboarding 流程：创建 `/onboarding/` 或 `/create-company/` 路由 + 页面组件；或给 `/system-admin/` 页面增加"创建您的第一个公司"引导 UI（当 `/me/` 返回 `companies: []` 时）。同时修复 Vite WebSocket 在 `https://www.daydaymoney.com` 下的混合内容问题（`ws://` → `wss://`）。
- **Why**: 当前无公司用户登录后停留在空白页，无任何可操作 UI，仍属 UX 断裂。
- **How to apply**: 前端新增路由 + 创建公司 API + `/me/` API 状态驱动的 UI 分支。后端可能需要新增 tenant/company 创建 API（如尚无公开端点）。

## [OPT-20260728-014] CI route check — completed 2026-07-28

- **Status**: completed
- **Completed**: 2026-07-28
- **Context**: `taskGateway/scripts/ci/check_go_routes_vs_apisix.py` — 覆盖全部 11 个 Go 服务的 HandleFunc 路由注册与 routes.yaml 声明的一致性比对。首次运行即发现 `/api/public/email-invitation/` 缺少网关路由，已修复。原有 `check_routes.sh`（仅覆盖 taskBill）改为调用此 Python 脚本。
- **Related**: [[fix-one-search-all-meta-rule]]

## [OPT-20260728-002b] UserSerializer workspace cache — completed 2026-07-28

- **Status**: completed
- **Completed**: 2026-07-28
- **Context**: user_serializer.py 新增 `_cached_get_workspace()` 辅助函数，基于 User 实例属性对 `(tenant_id, workspace_id)` 键做幂等去重，消除 `get_current_workspace` 的 N+1 HTTP 调用。
- **Related**: [[fix-one-search-all-meta-rule]]

## [OPT-20260728-015] is_tenant event-driven — completed 2026-07-28

- **Status**: completed
- **Completed**: 2026-07-28
- **Context**: BILLING_TRANSACTION_CREATED intent 2: auto-mark paying users as tenant. Django intent endpoint + Go IsTenantIntent handler + cmd binary + runAll.yaml entry. Go compiles clean.
- **Related**: [[domain-events-transport-kafka-switch]] [[billing-account-event-driven-init]]

