# Completed OPT Archive — 2026-08-12

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 40 条。
> 归档执行时间：2026-08-13T13:17:38+08:00

## [OPT-20260811-090] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: AiMonitor blackbox 探活已落地并上线：新增 http_2xx_html / http_redirect_no_assets 模块 + www-public/www-public-redirect/www-login job + WwwHomepageDown/WwwHomepageAssetsRedirect/WwwLoginDown 告警；promtool check + blackbox --config.check 通过，容器重启后三探活 target 均 up（prod 实际探测 success）。
- **Created**: 2026-08-11
- **Context**: 本次故障 DNS/TLS 正常、`/health` 200，但首页 302 到空 `/static/assets/`。若探活只打 health 或接受任意 2xx/3xx，告警不会触发。
- **Action**: (1) 为 `https://www.daydaymoney.com/` 增加 blackbox/HTTP 探活：期望最终码 200 且 body 含 `trae-service`/`<!DOCTYPE html>`；(2) 对 `Location: /static/assets/` 或最终 URL 落在空 assets 目录告警；(3) 同步 `/auth/login/` 200 断言。
- **Why**: 有探活才能在 dist 被并发 build 抹掉后分钟级发现，而不是等人报「打不开」。
- **How to apply**: `AiMonitor/prometheus/` blackbox modules；参考既有 `gitlab.daydaymoney.com` 探活条目。

## [OPT-20260811-080] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: dataMigrate 009/008/017 三个裸 ADD COLUMN 迁移改为 guarded_add_column 幂等（033 同模式）。本地 MySQL 临时库验证：fresh apply 全列落库；已含列复跑 no-op 无报错。check_no_duplicate/check_cold_hot PASS；check_mysql_compat 未命中本三文件。pre-push 生产迁移一致性核验 9/9 放行。
- **Created**: 2026-08-11
- **Context**: 本次库表迁移已成功应用 `009_invitation_pending_grants.sql`（`ALTER TABLE … ADD COLUMN pending_grants`）。该文件依赖 `data_migrate_log` L1 防重；若日志被清而列已存在，裸 ADD COLUMN 会失败。同仓 `033_resource_group_grant_effect.sql` 已有 `guarded_add_column` 范例。
- **Action**: (1) 将 009 改为 information_schema 守卫的 ADD COLUMN（或新编号 010 若 009 已入 log 不宜改内容）；(2) 扫描 `dataMigrate/**/*.sql` 中其它裸 `ADD COLUMN` 一并守卫；(3) 对已应用库跑幂等复跑验收。
- **Why**: 迁移脚本应可在日志丢失/手工复跑场景下不炸；与 033 模式对齐降低运维风险。
- **How to apply**: 参考 `dataMigrate/taskAuth/033_resource_group_grant_effect.sql`；目标 `dataMigrate/taskTenantService/009_invitation_pending_grants.sql`（或新增 `010_*`）。

## [OPT-20260811-076] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: PeopleGroups 删除分组改为 modalService.confirm 自定义确认（替代浏览器原生 confirm()，与 WorkPanel 删除任务一致；未新建独立 DeleteGroupConfirmModal.vue 以避免重复实现，modalService 已支持统一视觉与 data-traceId）。新增 PeopleGroups.delete-confirm.test.js 覆盖确认/取消两分支；顺手修正 canManage 权限码 group:members:manage（不存在）→ member:manage/group:manage/group-members:manage（与 shareLib/authz + group_handlers.go 一致，对照 tenant-menu-access-management-design），PeoplePageAccess 相关 12 例由红转绿。taskFE 20 例相关测试全绿，commit e8754d3。
- **Created**: 2026-08-11
- **Context**: 修复「编辑」死按钮时发现 `onDeleteGroup` 仍调用浏览器 `confirm()`；前端规范禁止 alert/confirm/prompt。
- **Action**: (1) 新增 `DeleteGroupConfirmModal.vue`；(2) 删除按钮打开确认框而非 `confirm()`；(3) 补 vitest 覆盖确认/取消路径。
- **Why**: 规范一致性；自定义模态可带 data-traceId 与统一视觉。
- **How to apply**: `taskFE/app/src/views/PeopleGroups.vue`；`components/DeleteGroupConfirmModal.vue`；`.ai/04_frontend_development/00_frontend_development.md` 弹窗规则。

## [OPT-20260811-074] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: 产出 docs/superpowers/specs/2026-08-11-rbac-coarse-code-migration-inventory.md：静态盘点 5 服务 25 处租户粗码门禁 + FE anyOfPerms/hasPerm 入口，拟定粗码→ui_region/page 映射表与 10 步 P1 迁移 PR 清单（只读→写、窄→宽、FE 回退最后），对齐 v75 设计 §8 完成判据。未改代码删除 RequirePerm（按条目范围）。docs b54a9ba。
- **Created**: 2026-08-11
- **Context**: v75 设计锁定角色 UI 只配 page/region，但存量 `RequirePerm(member:manage|billing:manage|…)` 与 FE `anyOfPerms` 仍依赖粗码；设计 §8 要求后续分阶段退役。
- **Action**: (1) 列出全部租户粗码门禁调用点（Go `RequirePerm`/`HasPerm` + FE `hasPerm`/`anyOfPerms`）；(2) 为每个粗码拟定对应 `ui_region`（或 capability-region）；(3) 输出迁移顺序 PR 清单，不在本 OPT 内改代码删除 RequirePerm。
- **Why**: 过早删除会造成「有 region 无 API」权限空洞；需要可执行映射表才能开 P1。
- **How to apply**: 设计 `docs/superpowers/specs/2026-08-11-tenant-role-management-v75-design.md` §8；`shareLib/authz/permissions.go`；各服务 handler。

## [OPT-20260811-055] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: 审计完成：taskEvents 残留 /api/tenant 出站路径均非 legacy——starter.go 命中容器网关 parseRelayToTraePath 唯一形态，replace.go 前缀与 taskCloudService SSOT 对齐；真实 DLT 根因 FetchProject/StartVM 已由 WIP 修复。新增 3 例 starter 回归测试（路径/头部/错误透出），commit dfa4c26 已推送。
- **Created**: 2026-08-11
- **Context**: 修复 TASK_COMMENT_IMAGE_MENTIONED DLT 时发现 `FetchProject`/`StartVM` 仍用已下线的 `/api/tenant/...` 路径；同仓 `containermigrateawaitready/starter.go` 仍有 legacy relay-to-trae 路径。
- **Action**: (1) 搜索 `taskEvents/**/*.go` 中 `/api/tenant/` 出站调用；(2) 对照 taskCloudService/taskProjectService 约定路径批量替换；(3) 补 httptest 回归测。
- **Why**: 同类路径错误会再次把可恢复/可成功的事件永久投 DLT，且错误文案易误导为业务配置缺失。
- **How to apply**: 优先改 `taskEvents/internal/handlers/containermigrateawaitready/starter.go`；参考本次 `taskcommentimagementioned/clients.go` 约定路径。

## [OPT-20260811-068] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: 扫描脚本 + 8 例自测落入 db/scripts/ci 并已推送（db 9c3e0fb）；.pre-commit-config.yaml 已接入 check-vue-template-undefined-identifiers（随 meta f3cb265 提交）。当前树扫描 0 命中；存量真实缺陷 SystemAdminReferralPerformanceDrawer formatDate 已入 KNOWN_ISSUES 并登记 OPT-20260812-007 追踪修复。
- **Created**: 2026-08-11
- **Context**: BillingDashboard 模板调用了不存在的 `centsToYuanInternalStr`，生产有交易数据时渲染抛错导致空白页；同类问题可用扫描在 CI 提前发现。
- **Action**: (1) 将本会话用的「template 函数调用 vs script setup 定义」扫描脚本落入 `taskFE`/`db/scripts/ci`；(2) 接入 pre-commit 或前端 CI，仅对变更 `.vue` 阻断。
- **Why**: 防止再次出现未定义模板标识符导致生产白屏，且无编译期错误。
- **How to apply**: 参考本会话扫描逻辑；注意 props/emits/组件名假阳性过滤（如 `formatDate` 来自 props）。

## [OPT-20260811-049] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: 新增 taskGitOauth/scripts/probe_github_fallback_ips.sh：逐 IP --resolve POST /login/oauth/access_token 探测 TLS+HTTP 可达性，IP 优先读 env 否则提取 Go 源码 []string 字面量；全不可达输出 critical JSON（可挂 cron/AiMonitor）。实测 3 fallback IP http=404 可达。commit 69db8c0 已推送。cron/AiMonitor 定时挂接留待后续（可选）。
- **Created**: 2026-08-11
- **Context**: 2026-08-11 修复 exchange_failed：本机 DNS 的 github.com 边缘不可达，taskGitOauth 改用内置 fallback IP 串行拨号。CDN 边缘 IP 会漂移，列表可能过期。
- **Action**: (1) 写小脚本对 `defaultGithubDialFallbackIPs` 逐 IP `--resolve` POST `/login/oauth/access_token`；(2) 失败告警或自动从探测结果刷新列表；(3) 挂 cron 或 AiMonitor 检查。
- **Why**: fallback IP 失效会再次导致全量 GitHub OAuth 换票失败。
- **How to apply**: `taskGitOauth/infrastructure/outbound_github_dial.go`；可选 `AiMonitor/` 探测任务。

## [OPT-20260811-037] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: 评估+配置落地：保留 Loki /ready 探针（aimonitor-promtail 无 host 端口，探 promtail 自身需动 AiMonitor compose），health_check retries 15→24 覆盖冷启动窗口，yaml 内补运维注释说明 retrying 属预期竞态非故障。commit 6f7c3aa 已推送。
- **Created**: 2026-08-11
- **Context**: goal 清库后 start-all 时，`promtail-local` 短暂 `retrying`（`http://${INFRA_HOST}:3100/ready` → Loki 503），Loki 就绪后自动变 healthy；属启动竞态非业务故障。
- **Action**: (1) 评估 `conf/runAll.yaml` 中 `promtail-local.health_check` 是否应增加初始 backoff / 依赖 `ai-monitor` 内 Loki ready；(2) 或改探针为 promtail 自身 HTTP（若暴露）而非 Loki `/ready`；(3) 补一句运维说明到 `runAll/ai.md`。
- **Why**: 清库+全量启动后 UI 会短暂显示 1 个 retrying，易被误判为失败。
- **How to apply**: `conf/runAll.yaml` `promtail-local`；`runAll/scripts/runall-local-promtail.sh`；AiMonitor compose Loki。

## [OPT-20260812-007] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: SystemAdminReferralPerformanceDrawer 模板 formatDate 复用 adminFormatDate 修复；补 2 渲染测试；KNOWN_ISSUES 豁免移除 + selftest 改断言重新拦截。taskFE dd66149 / db f026f07 已推送。
- **Created**: 2026-08-12
- **Context**: 夜间 OPT-068 扫描脚本在存量代码中发现真实缺陷：`SystemAdminReferralPerformanceDrawer.vue` 模板 `{{ formatDate(item.consumed_at) }}` 调用了未定义标识符（非 prop/import/local，defineProps 仅 visible/userId，无全局 formatDate 注册）。渲染该抽屉的消费记录单元格时会抛 TypeError。已在 OPT-068 门禁 KNOWN_ISSUES 豁免以保证绿灯并拦截新增回归。
- **Action**: (1) 在 `SystemAdminReferralPerformanceDrawer.vue` 引入/定义 `formatDate`（复用 `taskDetailBranchAndRepoUtils.js` 或 `workPanelBranchHelpers.js` 的日期格式化，或抽公共 utils）；(2) 补该组件渲染测试；(3) 从 `db/scripts/ci/check_vue_template_undefined_identifiers.py` 的 KNOWN_ISSUES 移除本条目。
- **Why**: 未定义函数调用在真实渲染时抛错导致区块空白，与 BillingDashboard `centsToYuanInternalStr` 同类。
- **How to apply**: `taskFE/app/src/components/SystemAdminReferralPerformanceDrawer.vue`；`db/scripts/ci/check_vue_template_undefined_identifiers.py`。

## [OPT-20260811-041] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: filterCatalogPages 子项裁剪：页面名未命中时仅保留匹配 children；PeopleRoles 角色矩阵接入过滤输入框（PeopleAccess 已 v75 角色优先无树，改接实际消费端）；补 3 vitest。taskFE bb8ca84 已推送。
- **Created**: 2026-08-11
- **Context**: 访问管理目录已加 `catalogFilter` 快速过滤；当前命中页面名或任一子区域即展示整页及其全部 children，子区域较多时仍需肉眼找目标勾选框。
- **Context（2026-08-12 夜间核）**: `catalogFilter` 过滤 UI 未在 PeopleAccess.vue 接线，`filterCatalogPages` 仅为 peopleAccessCatalog.js 纯函数+单测，无消费端。需先建过滤输入框，再实现子项裁剪。
- **Action**: (1) 当 query 非空且页面名未命中时，仅渲染匹配的 `page.children`；(2) 页面名命中则仍展示全部 children；(3) 补 vitest 断言子项裁剪。
- **Why**: 区域数量随种子增长后，整页展示会削弱「快速」过滤的收益。
- **How to apply**: `taskFE/app/src/views/peopleAccessCatalog.js` 的 `filterCatalogPages`；`PeopleAccess.vue`；`PeoplePageAccess.test.js`。

## [OPT-20260811-042] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: rgDeepLink.js 新增 parseRgKey/findRgTarget/scrollToRgKey/initRgDeepLink；main.js 挂载后初始化 + hashchange + router.afterEach 重扫；styles.css 高亮 class + prefers-reduced-motion 降级；5 vitest 全绿。taskFE c2647f6 已推送。
- **Created**: 2026-08-11
- **Context**: 访问管理「打开页面/定位」已支持 `#rg=<group_key>` 滚动；多区块长页滚动到位后，用户仍可能一时找不到目标区域。
- **Context（2026-08-12 夜间核）**: `#rg=` 链接仅由 tenantConsoleNav.js 生成（`buildTenantResourceHref`），全仓无消费端——无 `scrollToRgKey` 函数、无 hashchange 滚动监听。需先实现定位滚动，再补高亮。
- **Action**: (1) 在 `scrollToRgKey` 成功后为目标元素加短暂 outline/ring class（如 1.5s）；(2) 用 `prefers-reduced-motion` 降级为静态边框；(3) 补 `rgDeepLink.test.js` 断言 class 添加与清理。
- **Why**: 仅 scrollIntoView 在视觉噪音大的设置页不够直观，「定位」语义期望有焦点反馈。
- **How to apply**: `taskFE/app/src/utils/rgDeepLink.js`；可选全局 CSS 类放在 `taskFE/app/src/styles.css`。

## [OPT-20260812-001] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: 硬件面板 fetchAvailableInstances 复用 hasRequiredAvailableInstancesContext + buildAvailableInstancesFilterParams，删除内联 URLSearchParams 拼装；补空 data_disk_category 单测。taskFE 4b4d35a 已推送。
- **Created**: 2026-08-12
- **Context**: `useServerConfigHardwarePanel.js` 内联复制了 `buildAvailableInstancesFilterParams` / 必填校验逻辑；本会话为支持「不要数据盘」已在两处分别改校验，后续易再漂移。
- **Action**: (1) 在 `fetchAvailableInstances` 改调 `hasRequiredAvailableInstancesContext` + `buildAvailableInstancesFilterParams`；(2) 删除重复的 URLSearchParams 拼装；(3) 补/更新 composable 单测覆盖空 data_disk_category。
- **Why**: 双份逻辑会再次出现「工具函数已允许空盘型、面板仍拦截」类回归。
- **How to apply**: `taskFE/app/src/composables/hardwarePanel/useServerConfigHardwarePanel.js`；`taskFE/app/src/utils/availableInstancesQueryParams.js`。

## [OPT-20260812-003] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: trae-agent-docker-push selftest 挂入 .pre-commit-config.yaml，files 正则 gated（scripts/lib/trae-agent-docker-push*.sh|auto-commit.sh 变更时触发）；DRY_RUN 自测不触达 docker。meta af47020 已推送。
- **Created**: 2026-08-12
- **Context**: 约束 46 新增 `scripts/lib/trae-agent-docker-push_selftest.sh`，目前仅手动可跑；与 auto-commit_selftest 类似，未挂 CI 时脚本回归可能被静默破坏。
- **Action**: (1) 在 `.pre-commit-config.yaml` 增加 hook，当改动 `scripts/lib/trae-agent-docker-push*.sh` 或 `auto-commit.sh` 时跑自测；(2) 确认 DRY_RUN 不触达 docker/registry。
- **Why**: 扫描/水位线逻辑是 SessionEnd 兜底的核心，坏了会导致「以为推了其实没登 pending」。
- **How to apply**: 参考 `check-auto-commit-selftest` hook；路径 `scripts/lib/trae-agent-docker-push_selftest.sh`。

## [OPT-20260811-077] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: requireGroupManage（member:manage/group:manage/group-members:manage）接入创建/改名/删除分组；GET 列表保持成员可读；与 FE PeopleGroups 门禁一致；补 403 单测。taskTenantService e500fb7 已推送。
- **Created**: 2026-08-11
- **Context**: 本次新增 `handleGroupUpdate` 与创建/删除一致，仅 `requireCompanyMember`；RBAC 设计文档要求 PUT/DELETE 需 `group:manage`。
- **Action**: (1) create/update/delete 增加 `authz.HasPerm(..., PermGroupManage)`（或与 FE 一致允许 group-members:manage 作用域）；(2) 补 403 单测；(3) 确认 FE 页门禁与后端一致。
- **Why**: 任意公司成员可改组名会造成权限空洞，与 v63/v75 设计不符。
- **How to apply**: `taskTenantService/src/group_handlers.go`；`people_api_test.go`；`shareLib/authz/permissions.go`。

## [OPT-20260811-070] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: 新增 humanizeRequestErrorMessage 并接入 showRequestError/toastRequestError；UserProfile/UserCompanySettings/UserGitIdentities/FeatureParams/AccessTokenManagementPanel 内联展示点与 PendingInvitations toast 路径统一。taskFE 743dc2e 已推送。
- **Created**: 2026-08-11
- **Context**: `showRequestError`/`toastRequestError` 已中文化 Failed to fetch；但 UserProfile*、AccessTokenManagementPanel、PendingInvitations 等仍直接把 `error.message` 写入页面，可能仍露出英文。
- **Action**: (1) 扫描 `error.message ||` 赋值到 UI 文案的路径；(2) 统一经 `humanizeRequestErrorMessage` 或改走 `showRequestError`/`toastRequestError`；(3) 补 1～2 例单测。
- **Why**: 避免同类原始英文网络错误再次出现在非 modal 路径。
- **How to apply**: `requestErrorDisplay.js` 导出的 `humanizeRequestErrorMessage`；各 panel/views 的 catch 分支。

## [OPT-20260811-058] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: 表驱动 TestTaskConventionVerbNotRoutedAsTaskId 覆盖 search/todos 约定动词，断言不被误路由为 taskId；FE 扫描确认当前仅这两动词。taskTaskService 5b37ec6 已推送。
- **Created**: 2026-08-11
- **Context**: `/api/tasks/` 默认把首段当 taskId；FE 新增动作名（如 `search`）若未在 `todos`/`search` 之前拦截，会稳定 404「资源不存在」。同类风险仍在。
- **Action**: (1) 扫描 `taskFE` 中 `/api/tasks/<verb>/` 调用；(2) 对照 `taskTaskService/src/main.go` 早分支列表；(3) 可选 CI：FE 约定动词 ⊆ 后端白名单，或测试表驱动覆盖每个动词路径。
- **Why**: 防止下一个动词（renew/batch/…）再次被误路由成 taskId。
- **How to apply**: `rg "/api/tasks/[a-z_]+/" taskFE`；扩展 `TestSearchTasksConventionPath` 模式为表驱动。

## [OPT-20260811-071] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: 抽取 HomeFeatureCard.vue（icon/title/description/href），Home.vue v-for 替换三份拷贝并收敛数据；补 feature-cards 测试。taskFE 1deb110 已推送。
- **Created**: 2026-08-11
- **Context**: `Home.vue` `#projects` 三张功能卡模板高度重复（图标/标题/文案/CTA 结构相同）；本会话已统一内边距与 hover 样式，后续改文案或样式仍易漏改一张。
- **Action**: (1) 抽出 `HomeFeatureCard.vue`（props: icon/title/description/href）；(2) `Home.vue` 用 v-for 或三次引用替换三份拷贝；(3) 迁移/保留 `Home.feature-cards.test.js` 断言。
- **Why**: 降低样式漂移风险，便于继续美化或 A/B CTA。
- **How to apply**: `taskFE/app/src/views/Home.vue`；新建 `taskFE/app/src/components/HomeFeatureCard.vue`（或 `views/home/` 目录）。

## [OPT-20260811-089] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: 静态相对导入可解析性门禁（check_fe_relative_imports.py + 4 自测）挂入 pre-commit，files:^taskFE$ 触发；拦截 v75 漏提交模块类白屏回归。db 688cacd / meta 3dea6db 已推送。
- **Created**: 2026-08-11
- **Context**: www.daydaymoney.com 整站挂掉根因是 v75 提交遗漏 `saveSubjectResourceAccess.js` / `resourceGrantEffects.js`，`runall-lifecycle.sh build` 先删 dist 再失败，空 vite preview 对 `/` 返回 302→`/static/assets/`→404；runAll `/health` 仍 200 掩盖故障。本会话已补文件 + 原子构建 + health 校验 dist，但仍缺「提交前必过 production build」门禁。
- **Action**: (1) 在 pre-commit/CI 对 `taskFE/app/src/**` 变更跑 `bash taskFE/app/scripts/runall-lifecycle.sh build`（或等价 `vite build --outDir ../dist.next`）；(2) 失败阻断提交；(3) 可选：扫描 unresolved relative import 的静态检查作为更快的前置门。
- **Why**: 仅靠运行时健康检查无法在合入前拦住缺文件；再次漏提交模块会再次清空公网 SPA。
- **How to apply**: `.pre-commit-config.yaml` / `db/scripts/ci/`；调用 `taskFE/app/scripts/atomic-vite-build.sh`（或 `runall-lifecycle.sh build`）；参考 `check_spa_static_assets.py`；勿再写「先 rm -rf dist」。

## [OPT-20260811-073] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: taskFE 内联 error.message 剩余 15 处展示点统一 humanizeRequestErrorMessage（taskDetailGitFns/Editing/FetchFns/ProjectRepoState、EmailBinding/PhoneBinding/GitIdentityCreateModal/MemberList）；新增 Failed to fetch 回归单测；commit 3aa2645 已推送。
- **Created**: 2026-08-11
- **Context**: 本次 `showRequestError`/`toastRequestError` 已自动中文化 `Failed to fetch`；但多处直接写 `errorMessage.value = error.message`（UserProfile、AccessTokenManagementPanel、GitIdentityCreateModal、PendingInvitations toastService.error 等）仍可能露出英文。
- **Action**: (1) grep `error.message ||` 于 taskFE views/components；(2) 对用户可见赋值统一套 `humanizeRequestErrorMessage` 或改走 `showRequestError`/`toastRequestError`；(3) 补 1～2 例单测防回归。
- **Why**: 同类英文网络错误会在其它页面再次出现，体验不一致。
- **How to apply**: `requestErrorDisplay.js` 导出的 `humanizeRequestErrorMessage`；优先人员/资料相关面板。

## [OPT-20260811-066] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: Home.vue 页脚占位链接真实化：首页→/、支持>联系我们→#footer-contact 锚点（联系区加 id）、无目标页的帮助中心/常见问题隐藏；新增 Home.footer-links.test.js 3 断言（无残留 href="#" 占位链）；commit 184ab42 已推送。
- **Created**: 2026-08-11
- **Context**: Hero「了解更多」已改为 `#projects`（OPT-060 闭环）；`Home.vue` 页脚「首页/帮助中心/联系我们/常见问题」仍为 `href="#"`。
- **Action**: (1) 「首页」改为 `/`；(2) 其余三项按产品确认补真实路径或隐藏；(3) 补 vitest 断言。
- **Why**: 占位链点击无导航，页脚可达性断裂。
- **How to apply**: `taskFE/app/src/views/Home.vue` 页脚列表；同步评估 `PageFooter.vue` / `LoginMarketingFooter.vue`。

## [OPT-20260812-008] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: 已清理全部 34 子仓：删除 * copy*（9 处）、mysql/data.old-20260810/、Untitled 等噪声；25 子仓正式落盘 LICENSE/COMMERCIAL.md/README.md（COMMERCIAL 统一为官方版 taskFE 449829d），全部提交并推送。遗留 taskFE/taskCloudService 为并发会话 WIP，未触碰。
- **Created**: 2026-08-12
- **Context**: 本会话提交时多个子仓存在未跟踪的 `COMMERCIAL.md`/`LICENSE`/`README copy.md`/`nohup-*.log`/`mysql/data.old-*`；为避免污染提交与推送已刻意排除。
- **Action**: (1) 确认哪些 LICENSE/COMMERCIAL 属于正式许可落盘，统一命名后正式提交；(2) 删除 `* copy*`、nohup 日志与旧 MySQL 数据目录；(3) 必要时补 `.gitignore`。
- **Why**: 脏未跟踪文件会让 submodule 长期显示 dirty，干扰 meta 指针同步与 auto-commit。
- **How to apply**: `git submodule foreach 'git status --porcelain'`；排除模式见本会话提交脚本。

## [OPT-20260811-036] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: BuildGroup 成功后仅移除该组内已登记且编译成功的服务名（trimRegistrationsAfterGroupBuild），失败/跳过项、他组与并发登记保留；新增单元测试 2 + BuildGroup 集成测试 1，全量 go test ./src 72s 绿；约束 42 验收标准补第 7 条；commit 5cfe5b5 已推送。
- **Created**: 2026-08-11
- **Context**: 本次仅让全量 `BuildAll` 清空登记；分组「全部重新编译」有意不清全局登记，避免只编一组却误消其它组待重启项。
- **Action**: (1) 评估是否在 `BuildGroup` 成功后仅移除该组内已登记且编译成功的服务名；(2) 若做则补测例与文档 42 验收条。
- **Why**: 分组编译后部分登记可能已过时，但全清不安全。
- **How to apply**: `Runner.BuildGroup`；`rewriteRegistrationsAfterRun` 可复用。

## [OPT-20260811-056] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: taskEvents SetRunStatusByParent 对 taskAIComment by-parent status 404 返回 ErrAICommentNotFound 哨兵；调用方 errors.Is 降级 warn 并写明「start-vm 已成功，评论不存在/已清理，状态回写跳过」；新增哨兵单测 + handler 集成测试（404 回写时 dispatch 仍 DispatchSuccess）；commit 42457ae 已推送。
- **Created**: 2026-08-11
- **Context**: 重放 `task_15556574121458564865` 后 start-vm 已成功，但 `SetRunStatusByParent` 对 `cmt_15564826018991865865` 返回 HTTP 404。
- **Action**: (1) 核对 taskAIComment by-parent status 路由与 parent_comment_id 存储；(2) 确认 agent comment 是否未创建或已清理；(3) 404 时降级为 warn 并写明可操作原因，避免误以为启服失败。
- **Why**: 启服成功后状态回写失败会让前端/Agent 线程停留在 pending，用户感知不一致。
- **How to apply**: `taskEvents/.../clients.go` SetRunStatusByParent；`taskAIComment` by-parent status handler。

## [OPT-20260812-010] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: mock/空平台评论 CSC bootstrap 永久失败应收口 failed：bootstrapCommentCSCRuntime 返回哨兵错误 errCommentCSCPlatformUnsupported，ccbStartBinding/attach/promote 三处调用点区分永久失败（置 failed + failed 阶段日志）与 mock 人工回填 reachability 路径（保持 starting）；新增 markCommentContainerBindingFailed + commentCSCReachabilityPossible；回归测例 TestCommentContainerBindingMockPermanentFailureMarkedFailed 全绿，全量 go test ./src/ 通过，commit 24bf93f 已推送。
- **Created**: 2026-08-12
- **Context**: `bootstrapCommentCSCRuntime` 对 platform=mock/空直接报错，但 `ccbStartBinding` 仍记 starting + csc_allocated，可达性若不来自外部路径会永久卡在「启动中」。
- **Action**: (1) 区分永久失败（未配置云平台）与可等待 reachability；(2) 永久失败将 binding 置 failed 并写失败阶段日志；(3) 补 schedule 回归测例，不破坏现有 mock reachability 人工回填路径。
- **Why**: 仅修 runtime 文案后，mock 任务仍会无限显示启动中，用户无法感知需先配云平台。
- **How to apply**: `comment_csc_bootstrap.go`；`comment_container_bindings_schedule.go`；`comment_container_bindings_log_test.go`。

## [OPT-20260811-039] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: 拆分 SystemAdminGitlabResources.vue 498 行（原 688）：抽取 SystemAdminGitlabTenantPanel.vue（租户配额查询/开通）与 SystemAdminGitlabRegionForm.vue（新建区域表单），保留全部 data-testid，cloud-provider-options datalist 上移父级共用；全量 vitest 2014 例 + build 通过，commit db8b0b2 已推送。
- **Created**: 2026-08-11
- **Context**: 滚动修复未改该页，但文件已 688 行（门禁默认 500）；区域 CRUD / 容量 / 租户开通等逻辑堆在单文件。
- **Action**: (1) 按区域表单、区域列表卡片、租户配额表拆成子组件/composable；(2) `wc -l` ≤500；(3) 保留既有 data-testid。
- **Why**: 触发行数门禁或再改该页时必须先削分；提前拆可降低后续回归成本。
- **How to apply**: `.ai/01_project_constraints/27_source_file_line_limit_auto_reduce.md`；`SystemAdminGitlabResources.vue`。

## [OPT-20260812-004] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: runAll 9999 页头新增「推送 trae-agent 镜像」按钮（约束 46 对称 42）：GET /api/trae-agent-push/status 读 pending/水位线/运行锁/日志尾部；POST /api/trae-agent-push 触发 scripts/trae-agent-docker-push.sh --if-pending --background；GET /api/trae-agent-push/progress SSE tail 日志。前端按钮+label 并入 refresh 轮询，陈旧锁按未运行处理。6 单测 + status_page 嵌入选段断言，全量 go test ./src/ 通过，commit a6a1974 已推送。
- **Created**: 2026-08-12
- **Context**: 约束 42 有 UI「精准编译重启」消费登记；约束 46 目前靠 Agent 前台推送 + SessionEnd 后台兜底，运维缺少与 runAll 页面对称的一键入口。
- **Action**: (1) 评估在 9999 页头增加「推送 onlineServiceJS 镜像」；(2) 读 pending/水位线状态；(3) 触发 `scripts/trae-agent-docker-push.sh` 并 SSE 进度。
- **Why**: 后台 nohup 日志不便观察；人工补推需记命令路径。
- **How to apply**: `runAll/src/status_ui/`；参考 precise-restart API；约束文 `46_trae_agent_online_service_docker_push.md`。

## [OPT-20260812-013] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: per-binding 独立缓存启动 TraceId：useCommentContainerBindings 按 commentId 缓存；SSE 总线透传 trace_id；并行竞态单测+2018 vitest 全绿；commit 5027f29 已推 taskFE
- **Created**: 2026-08-12
- **Context**: 当前所有评论执行细节共用任务级 `statusTraceId`；并行独立 CSC 同时启动时后一次启动会覆盖前一次展示。
- **Action**: (1) 在 `useCommentContainerBindings` 按 commentId 缓存 startTraceId；(2) `buildPerBindingServerStatusProps` / CommentsSection 传入对应值；(3) 补并行启动竞态单测。
- **Why**: 多评论并行启动时任务级单槽会互相覆盖，排障时易拿错 trace。
- **How to apply**: `useCommentContainerBindings.js`；`TaskDetailCommentsSection.vue` `:start-trace-id`。

## [OPT-20260811-044] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: 审计 rg RequireRegion( 于 taskAuth/taskTenantService/taskBill/taskCloudService — 仅 taskAuth rbac_resource_groups.go 使用，且已用 HasRegionView/HasRegionOperate，业务 handler 无裸 RequireRegion 误用；无需改动
- **Created:** 2026-08-11
- **Context:** v73 后 view-only 授予不注入遗留 bare `region:key`；仍用 RequireRegion/HasRegion 且仅依赖 bare 码的读路径会对 view-only 误拒（HasRegion 已含 :view，但若 handler 手写查 bare 码则有问题）。多数路径经 HasRegion 已兼容。
- **Action:** 审计租户读 handler，显式 RequireRegionView；写 handler 显式 RequireRegionOperate。
- **Why:** 语义清晰、避免误用。
- **How to apply:** `rg -n 'RequireRegion\(' taskAuth taskTenantService taskBill taskCloudService --glob '*.go'` 逐项改。

## [OPT-20260811-063] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: 抽取 ResourceGrantMatrix.vue 共享授权矩阵组件；InviteAccessGrants 与 PeopleRoles 复用（v-model view/operate 效果映射）；新增组件单测 5 例，people/auth 41 例全绿，build 通过；commit taskFE 5c1a4d2 已推送
- **Created**: 2026-08-11
- **Context**: 邀请页与访问管理页各自维护 page/region 勾选 UI，易漂移。
- **Action**: (1) 抽 `ResourceGrantMatrix.vue`；(2) PeopleAccess 与 InviteAccessGrants 共用；(3) 补单测。
- **Why**: 减少双份 UI 与 effect 切换逻辑分叉。
- **How to apply**: `InviteAccessGrants.vue`；`PeopleAccess.vue` region_matrix 区块。

## [OPT-20260811-075] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: 与 063 同一抽取：PeopleRoles/Invite 两处矩阵内联/分叉已收敛至 ResourceGrantMatrix（含 view/operate）；PeopleAccess 预览为只读 role 分配非授权矩阵，保持原扁平列表；行数门禁满足；commit taskFE 5c1a4d2
- **Created**: 2026-08-11
- **Context**: v75 已落地 PeopleRoles 与 PeopleAccess，但矩阵仍内联/分叉；InviteAccessGrants 亦独立；OPT-063 亦跟踪抽取。
- **Action**: (1) 抽出共享矩阵组件（含 view/operate）；(2) PeopleRoles / Access 预览 / Invite 复用；(3) 行数门禁 ≤500。
- **Why**: 三处拷贝易漂移；角色优先后矩阵成为核心控件。
- **How to apply**: `taskFE/app/src/views/PeopleAccess.vue`、`PeopleRoles.vue`；新建 `components/people/` 或 `domain/auth/`；关联 OPT-20260811-063。

## [OPT-20260811-033] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: runAll /api/stop 新增 preview:true 干跑返回级联计划；UI stop 前预览，若连带停止依赖服务（task-auth→task-gateway→taskFE）弹确认；补 TestAPIStopService_Preview 回归。提交 runAll 359b63f 已推送。
- **Created**: 2026-08-11
- **Context**: goal 执行 OPT-010 时对 `task-auth` 发 `/api/stop`，随后观察到 `taskFE` 与 `task-gateway` 变为 stopped，公网 www 短暂 502；需手动 `/api/start` 恢复。
- **Action**: (1) 复现：仅 stop task-auth，记录其它服务状态变化；(2) 若为会话/依赖连带停止则收紧 stop 语义或 UI 提示；(3) 补回归测试或 runbook「停 auth 验收微信 502」安全步骤。
- **Why**: 验收类破坏性操作不应导致全站边缘不可用。
- **How to apply**: `runAll/src/ui.go` stop 路径；`docs/runbooks/`。

## [OPT-20260811-034] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: 新增 useTenantPageAccess（fail-open）+ TenantPageAccessEmpty；对 cloud/task_panel/feature_params/status/deliverable 五个 settings 页套用 page:* 空态。全量 2032 例绿 + build 通过。taskFE 提交 3994d7d 已推送；依赖前置 fix 09de043（usePermissions 补 hasPage/hasRegion 系列）。
- **Created**: 2026-08-11
- **Context**: OPT-022 已覆盖 company/gitlab/orders + overview/transactions/usage；Explore 指出 cloud/task_panel/feature_params/status/deliverable 等大页仍可直链进入。
- **Action**: (1) 对 `WorkspaceSettingsCloudPlatform` / `TaskPanel` / `FeatureParams` / `Status` / `DeliverableSystemList` 套用 `useTenantPageAccess` + `TenantPageAccessEmpty`；(2) 可选同步 PeopleInvite/Manage/Groups 从粗码迁 page/region；(3) 补 vitest mock。
- **Why**: 侧栏隐藏不是安全边界；直链无空态易误导。
- **How to apply**: 模式抄 `BillingOrders.vue` / `TenantCompanySettings.vue`；page key 见 `tenantConsoleNav.js`。

## [OPT-20260812-006] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: taskFE 新增 CreateProject.auto-clone-nested.playwright.test.js（3 例全绿），断言 checkbox 可见/默认勾选/取消后 POST body=false；已推送 taskFE 0e5af73
- **Created**: 2026-08-12
- **Context**: 已有 vitest 覆盖提交字段；缺 Playwright 在真实页面断言 `create-project-auto-clone-nested-repos` 可见与默认勾选。
- **Action**: (1) 新增 `CreateProject.auto-clone-nested.playwright.test.js`；(2) 填仓库 URL 后断言 checkbox 出现且 checked；(3) 取消勾选后 mock POST 校验 body。
- **Why**: 防止模板条件 `hasAnyGitRepoUrl` 回归导致开关消失。
- **How to apply**: 参考 `taskFE/tests/CreateProject.clone-alias.playwright.test.js`。

## [OPT-20260812-011] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: taskFE 新增 TaskDetail.comment-starting-csc-align.playwright.test.js（1 例全绿），mock Starting+云实例创建中+评论级 CSC 无 instance，断言启动中且无未找到服务器配置记录；已推送 taskFE 0e5af73
- **Created**: 2026-08-12
- **Context**: 已有 Go/vitest 覆盖 Starting 对齐；缺 Playwright 在任务详情「服务器运行状态」Tab 断言文案。
- **Action**: (1) mock `server-runtime-status` 返回 Starting + 云实例创建中；(2) mock binding starting + logs；(3) 断言 lifecycle=启动中且 runtime 展示启动中、不含未找到服务器配置记录。
- **Why**: 防止 FE display map / absent hydrate 回归把创建中误回落为未创建。
- **How to apply**: 参考 `taskFE/tests/TaskDetail.runtime-hydrate-lifecycle.playwright.test.js`。

## [OPT-20260811-059] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: taskFE Home.vue #projects 三张功能卡「了解更多」→ /auth/register/；新增 Home.cta-links.test.js 回归（3 例全绿）；本地 preview 渲染 href 正确，生产 chunk 已含 /auth/register/（提交 a81d0c6）
- **Created**: 2026-08-11
- **Context**: `Home.vue` `#projects` 三张功能卡「了解更多」已从 `href="#"` 改为 `/auth/register/`；本地 vitest 2/2 通过，`taskFE` 已在精准重启登记中。公网仍需发布后硬刷新确认。
- **Action**: (1) 在 http://10.2.150.68:9999/ 执行「精准编译重启」；(2) 打开 https://www.daydaymoney.com/ 并硬刷新；(3) 点击 `#projects` 三张卡「了解更多」，确认均进入 `/auth/register/`。
- **Why**: 单元测不覆盖生产静态托管与旧 chunk 缓存。
- **How to apply**: `taskFE/app/src/views/Home.vue`；chunk 含 Home 的静态资源。

## [OPT-20260811-064] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: taskFE Home.vue Hero「立即开始」→ /auth/register/；Home.cta-links.test.js 覆盖；本地 preview 与生产 chunk 均确认（提交 a81d0c6）
- **Created**: 2026-08-11
- **Context**: `Home.vue` Hero「立即开始」已改为 `/auth/register/`；本地 vitest T3 通过，`taskFE` 已在精准重启登记中。公网仍需发布后硬刷新确认。
- **Action**: (1) 在 http://10.2.150.68:9999/ 执行「精准编译重启」；(2) 打开 https://www.daydaymoney.com/ 并硬刷新；(3) 点击 Hero「立即开始」，确认进入 `/auth/register/`。
- **Why**: 单元测不覆盖生产静态托管与旧 chunk 缓存。
- **How to apply**: `taskFE/app/src/views/Home.vue`；chunk 含 Home 的静态资源。

## [OPT-20260811-065] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: taskFE Home.vue Hero「了解更多」→ #projects；Home.cta-links.test.js 覆盖；本地 preview 与生产 chunk 均确认（提交 a81d0c6）
- **Created**: 2026-08-11
- **Context**: `Home.vue` Hero「了解更多」已改为 `href="#projects"`；本地 vitest T4 通过，`taskFE` 已在精准重启登记中。公网仍需发布后硬刷新确认。
- **Action**: (1) 在 http://10.2.150.68:9999/ 执行「精准编译重启」；(2) 打开 https://www.daydaymoney.com/ 并硬刷新；(3) 点击 Hero「了解更多」，确认滚动到 `#projects` 核心功能区。
- **Why**: 单元测不覆盖生产静态托管与旧 chunk 缓存。
- **How to apply**: `taskFE/app/src/views/Home.vue`；chunk 含 Home 的静态资源。

## [OPT-20260811-054] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: taskFE 修复历史服务器启动记录空态双文案：fetch 空成功不再写 message + 面板 v-else-if 追加 !message；新增面板/ fetch 单测 6 例 + Playwright 空态单条 E2E，修复既有 mock URL；全量 vitest 2038 绿；已推送 taskFE 3ad52e0，生产 dist 已更新
- **Created**: 2026-08-11
- **Context**: 任务详情「历史服务器启动记录」曾同时渲染 `message` 与空态，出现两条「暂无历史服务器启动记录」。已修 fetch 空成功不写 message + 面板 `!message` 才显示空态；本地 6 个 vitest 与 `npm run build` 已绿，`taskFE` 已登记精准重启。
- **Action**: (1) 在 http://10.2.150.68:9999/ 对已登记 `taskFE` 执行「精准编译重启」；(2) 打开原任务详情页并进入服务器配置/历史记录区块；(3) 无记录时确认「暂无历史服务器启动记录」仅出现 1 次；(4) 硬刷新排除旧 chunk。
- **Why**: 单元测不覆盖生产静态托管与浏览器 DOM；未重启则公网仍跑旧 bundle。
- **How to apply**: `ServerConfigServerStartHistoryPanel.vue`；`useServerConfigRuntimeFetch.js`；URL 见会话插件选择的 task-detail。

## [OPT-20260811-046] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: 按 v73 §8「自动按 HTTP method 推断 required_effect」落地：shareLib/authz 新增 `MethodRequiresOperate` / `RequireRegionByMethod` / `HasRegionByMethod`（只读方法需 view，写方法需 operate），4 例单测全绿；taskAuth `handleRoleResourceGroups` PUT 写路径校验改用 `HasRegionByMethod` 自动推断。shareLib 2b3400e / taskAuth d28137b 已推送。注：环境故障后经 /tmp/ram-work-r 重建提交。
- **Created**: 2026-08-11
- **Context**: 方案 C 推迟；可按 HTTP method 或成员标记声明 required effect。
- **Action**: 为 member 增加 required_effect，registry 反查时自动校验。
- **Why**: 减少 handler 手写 View/Operate 遗漏。
- **How to apply**: DDL + shareLib registry + 测试。

## [OPT-20260812-045] completed

- **Status**: completed
- **Completed**: 2026-08-12
- **Summary**: 任务详情拆分 Git 提交身份 / GitHub App 授权双区块；账号中心与 PeopleManage 文案区分；空态去连接引导
- **Created**: 2026-08-12
- **Context**: `TaskDetailLinkedProjectsViewMode` 在 `githubConnectedOptions.length===0` 时仅禁用空下拉，无创建/连接入口；克隆 Git 身份空态已有「+ 创建新的 Git 身份」。用户易把 people/manage 的「Git 身份」与「PR 所用 GitHub 授权账号」混淆。
- **Action**: (1) 空态改为提示「尚未连接 GitHub App」+ 跳转/触发现有 `TaskDetailGithubPrCredentialPanel` 连接流程；(2) 文案区分 Git 身份（name/email）与 GitHub OAuth；(3) 补前端单测。
- **Why**: 即便鉴权修复后，真实未连接用户仍会看到死下拉，无行动路径。
- **How to apply**: `taskFE/app/src/components/task-detail/TaskDetailLinkedProjectsViewMode.vue`；复用 `TaskDetailGithubPrCredentialPanel.vue` 连接按钮逻辑。

