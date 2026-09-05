# 前端开发规范

## 基本信息
- 版本：2.7.12
- 创建日期：2026-01-27
- 最后修改：2026-09-02
- 维护者：Trae AI 团队

## 子规则文件

- [平台设计规范技能包](01_platform_design_skills.md)：引入 [ehmo/platform-design-skills](https://github.com/ehmo/platform-design-skills)，涵盖 Apple HIG、Material Design 3、WCAG 2.2，用于界面设计、设计评审与可访问性审计
- [DESIGN.md 集成规范](02_design_md_integration.md)：引入 DESIGN.md 作为项目视觉基线输入，统一颜色、字体、间距与组件语义，减少页面风格漂移。可执行技能 `.claude/skills/design-md`；产品 SSOT 为仓库根 [`DESIGN.md`](../../DESIGN.md)

## 规则分类

### 核心规则
> 影响前端开发质量和安全性的关键规则，必须严格遵守

#### 执行规则
- 描述：定义前端开发的核心执行规范
- 适用场景：所有前端开发
- 优先级：高
- 规则：
  - 严格按照流程执行，确保每个步骤完整
  - 必须输出可直接运行的完整代码
  - 代码结构清晰、注释完整
  - 优先考虑用户体验和性能表现
  - **前端站点与 API 不同域名**时，须遵守项目约束 [异域前端与 API 域名分离](../01_project_constraints/17_cross_domain_frontend_api.md)（CORS、Cookie、鉴权载体、可配置 API 基地址、回调与 E2E 等）
  - **静态资源经 URL 引用且可能被缓存**时，须遵守 [静态资源缓存击穿（hash query）](../01_project_constraints/18_static_resource_cache_bust_query.md)：URL 须附带与文件内容绑定的 `?h=` / `?v=` 等查询参数；优先 `import` 模块图而非长期缓存的裸 `/utils/*.js`
  - **请求失败的错误展示**须在错误 DOM 上设置 **`data-traceId`**（值为该次请求 traceId），见 [请求报错展示须带 data-traceId](../01_project_constraints/24_frontend_error_data_trace_id.md)；纯前端校验不得伪造。**Agent 排障**：有 `data-traceId` 时须优先查 Loki 重建全链路路径，再改代码（[traceid-log-first-diagnosis.md](../../.claude/skills/1-brainstorming-design-docs/references/traceid-log-first-diagnosis.md)）
  - **入口 HTML `<head>` 须标明提供页面的服务**：`<meta name="trae-service" content="<serviceId>" />`（如 `taskAiProvider`、`task2app`），见 [HTML head 须标明提供页面的服务](../01_project_constraints/26_frontend_head_trae_service.md)；门禁 `python3 db/scripts/ci/check_frontend_head_trae_service.py`
  - **公网 SPA（taskFE）改动后必须完整 build**：修改会进入生产包的前端源码后，在 `taskFE/app/` 执行 `npm run build`（或 `bash scripts/runall-lifecycle.sh build`）。Django 已退役：`npm run build` = 纯 Vite（写出 `public/releases/<id>/` + 原子切换 `public/html` symlink），由 **Docker nginx**（`taskfe-nginx` :4000）托管；**不得**再串联 Django `collectstatic`（`collectstatic-after-vite.sh` 仅为无声 no-op，且不挂在 build 上）。目录 companion：[`taskFE/app/ai.md`](../../taskFE/app/ai.md)；历史故障经验（Django STATIC_ROOT 时代）：[`04_public_spa_static_js_404_after_vite_build.md`](../09_failure_experience/02_runtime_errors/04_public_spa_static_js_404_after_vite_build.md)
  - 严格遵循视觉风格规范
  - 弹窗必须使用自定义模态框实现，禁止使用浏览器自带弹窗（如 alert、confirm、prompt 等）
  - **导航链接禁止点击拦截**：凡「打开某个地址」的控件须使用真实 `<a href>`（或整页 `location.href`）；禁止 `@click.prevent` + `router.push` 冒充链接；鉴权失败禁止静默 `replace` 回用户来源页（须 modal 确认）。详见 [禁止链接点击拦截](../01_project_constraints/49_no_link_click_interception.md)
  - **副作用按钮须防重放**：写操作点击必须同步门闩 + `disabled`/`aria-busy` + 短窗 debounce，并在被接受的点击上生成 `Idempotency-Key`（同意图重试回传同一键）。禁止只靠下一帧 `disabled`。详见 [前端按钮点击须有防重放设计](../01_project_constraints/57_frontend_button_anti_replay.md)；参考 `taskFE/app/src/utils/clickGuard.js`
  - **无必要禁止 Teleport**：默认在视觉父就地挂载组件；禁止用 Vue `<Teleport>` / React `createPortal` 把功能卡送到远亲槽位。仅当祖先 overflow/transform 裁剪或错位浮层且无法就地解决时才允许，并须 `Teleport-OK` 注释。详见 [前端无必要禁止 Teleport](../01_project_constraints/50_no_unnecessary_vue_teleport.md)
  - **禁止无触发后台 API 轮询**：状态刷新须由用户操作或 SSE/推送驱动，禁止 `setInterval` 盲打 Describe / runtime-status。详见 [业务服务禁止进程内轮询 / 循环](../01_project_constraints/51_no_service_internal_poll_loop.md) 第 4 节；意图 `docs/intents/frontend/comment_runtime_no_background_poll.intent.md`
  - 每个模态框必须放置在独立的 Vue 组件文件中，以便于后续修改时进行定位
  - 模态框组件命名应遵循语义化原则，清晰表达其功能，如 `{Feature}Modal.vue` 格式
  - 模态框组件应具有独立的状态管理和逻辑处理能力，减少与父组件的耦合
  - 子模态框应位于主模态框之上（通过 z-index 确保层级正确），子模态框的背景遮罩应使用半透明黑色（如 `rgba(0, 0, 0, 0.5)` 或 `background: black; opacity: 0.5`）

#### 下拉菜单交互规范
- 描述：定义下拉类控件（含选择器、弹出菜单、组合框等）触发区域的指针点击行为
- 适用场景：所有带下拉/浮层面板的可点击触发器
- 优先级：高
- 规则：
  - **单击**（`click`）：展开/打开下拉菜单
  - **双击**（`dblclick`）：收起/关闭已打开的下拉菜单
  - 实现时需处理浏览器在双击前后会派发 `click` 的事件顺序，保证最终表现符合「单击打开、双击关闭」，避免闪烁或状态错乱

#### DESIGN.md 视觉基线规范
- 描述：前端页面在设计和实现时必须遵循项目级 `DESIGN.md` 作为统一视觉输入，并与平台规范协同执行
- 适用场景：所有 Web 前端页面与组件开发、重构、设计评审
- 优先级：高
- 规则：
  - 开发前加载技能 `.claude/skills/design-md`，并确认仓库根 `DESIGN.md` 的视觉令牌和组件样式约束
  - 页面实现优先复用 `DESIGN.md` 的语义角色（颜色、字号、间距、圆角、阴影）与 `taskFE` CSS 变量，禁止临时定义无语义样式
  - 禁止把 awesome-design-md 上游品牌文件整份覆盖到仓库根
  - 与平台规范冲突时，优先满足可访问性和可用性要求，再调整视觉参数
  - 详细落地流程请参考 `02_design_md_integration.md`

#### JS 加载规范
- 描述：规范JS代码的加载方式
- 适用场景：所有前端JS代码开发
- 优先级：高
- 规则：
  - JS 代码应仅在对应的 Vue 控件中加载，避免全局加载
  - 内联 JS：应直接在对应 Vue 组件的 script 标签中编写
  - 独立文件 JS：应通过 import 语句在对应 Vue 组件中按需加载，确保仅在该组件中生效
  - 禁止在全局范围内加载仅用于特定组件的 JS 代码
  - 模板或运行时若必须使用**裸 URL** 加载 `.js`/`.css`，须带内容 hash 查询参数，见 [18_static_resource_cache_bust_query.md](../01_project_constraints/18_static_resource_cache_bust_query.md)

#### 全局属性规范
- 描述：规范前端页面的全局属性使用方式
- 适用场景：所有前端页面开发
- 优先级：高
- 规则：
  - 禁止在 window 对象上挂载全局属性
  - 如需在多个组件间共享代码或数据，应采用独立文件导出并通过 import 语句引用的方式
  - 确保所有代码和数据的作用域清晰可控，避免全局污染

#### 大型组件行数门禁（自动拆分）
- 描述：单个前端组件文件体量过大时，可读性、复测与协作成本显著上升；须在触及该类文件时主动拆分为更小单元；**门禁一旦触发必须当场削到阈值以下**
- 适用场景：所有前端组件源码文件（含 Vue、React、Svelte 等单文件组件及等价 `.tsx`/`.jsx` 页面组件）；以物理文件行数为准（含模板、样式、脚本）
- 优先级：高
- 规则：
  - **阈值**：同一组件文件（单一路径）总行数 **大于 500 行** 时，视为超标（行数门禁触发）
  - **自动拆分**：在新建、大改或例行维护触及该文件时，**必须**将职责拆到多个子组件或 composable/store 等边界清晰的单元，使每个组件文件回落至阈值以内或与拆分同步收窄规模；禁止在无拆分前提下继续堆叠功能
  - **门禁触发后强制削减**：pre-commit/CI 行数门禁失败、或自检/用户点名超标时，Agent **必须立即**将输出中的全部目标文件削减到 **≤ 阈值** 并复跑验收，再继续其它改动；CI 遗留净增长容差**不免除**削文件义务。全局元规则见 [27_source_file_line_limit_auto_reduce.md](../01_project_constraints/27_source_file_line_limit_auto_reduce.md)
  - **度量**：以编辑器/仓库内实际行数为准；样式极长的文件可按项目惯例抽到独立样式模块或子组件，但仍计入拆分义务
  - **例外**：仅生成代码或第三方拷贝且明确标注不可改的胶水文件，经评审可暂缓拆分，须在注释或文档中说明理由与后续计划，且不得借例外继续净增长
  - **自动化门禁**：`task2app/scripts/ci/check_frontend_component_line_limit.py` —— `scripts/hooks/pre-commit` 对已暂存组件文件校验；GitHub Actions `DDD BDD Compliance` 工作流对 PR/推送的 diff（`TASK2APP_DIFF_BASE` / `TASK2APP_DIFF_HEAD`）执行校验，默认阈值 500（`TASK2APP_FRONTEND_MAX_LINES`）。CI 对**基线已超标**的遗留文件额外使用 **`TASK2APP_FRONTEND_LEGACY_GROWTH_GRACE`**（净增长容差，渐进拆分期间）；pre-commit 默认容差 0，禁止在未拆分前提下大块堆代码

#### 错误反馈展示规范
- 描述：规范前端对接口错误的用户提示与研发调试信息展示方式
- 适用场景：所有前端接口调用和异常处理
- 优先级：高
- 规则：
  - 用户可见错误提示必须使用 toast（或全局消息提示）展示友好文案，禁止直接展示原始异常堆栈
  - 友好文案优先使用后端返回的 `message` 字段；当无 `message` 时使用统一兜底文案
  - 研发调试所需的 `error_detail`、`request_id` 等细节应保留在控制台日志或调试面板中，便于快速定位问题
  - toast 文案应简短、可操作（说明失败类型和下一步建议），避免技术术语泛滥

### 最佳实践
> 提升前端开发效率和代码可维护性的建议

#### 前端开发工作流
- 描述：定义前端开发的标准工作流程
- 适用场景：所有前端开发项目
- 优先级：中
- 规则：
  1. 需求分析与功能拆分
  2. 设计策略与视觉方案制定
  3. 信息架构与交互流程设计
  4. UI 界面实现与代码输出
  5. 性能与用户体验优化

#### Vue 控件规范
- 描述：规范Vue控件的组织和命名
- 适用场景：所有Vue控件开发
- 优先级：中
- 规则：
  - 每个控件一个 Vue 文件
  - 每个控件分配一个固定别名
  - 别名需展现在 div 属性上，以便开发时快速定位

#### 组件分类规范
- 描述：规范前端组件的分类和实现方式
- 适用场景：所有前端组件开发
- 优先级：高
- 规则：
  - 界面组件（UI Components）：专注于展示逻辑，优先使用函数式组件实现，避免包含副作用
  - 逻辑组件（Logic Components）：负责业务逻辑处理，允许包含副作用（如API调用、状态管理等）
  - 组件拆分原则：将界面展示与业务逻辑分离，提高代码可维护性和复用性
  - 命名规范：
    - 界面组件：采用 `{Feature}.ui.vue` 格式命名，如 `Navbar.ui.vue`
    - 逻辑组件：采用 `{Feature}.logic.vue` 格式命名，如 `Auth.logic.vue`
    - 现有组件：在修改时逐步迁移，新组件必须遵循此规范
  - 项目修改原则：
    - 在项目修改过程中，必须对涉及的组件按照此原则进行调整
    - 每次修改至少确保一个组件符合规范
    - 建立组件迁移计划，分阶段完成所有组件的规范化
    - 定期检查规范执行情况，确保所有新代码都符合要求

### 风格指南
> 统一前端开发风格和格式的规范

#### 角色定位
- 描述：定义前端开发的角色定位
- 适用场景：前端团队角色划分
- 优先级：低
- 规则：
  - 资深 UI/UX 设计师，专注于 B 端界面的优秀用户体验与高级感视觉实现

#### 核心职责
- 描述：定义前端开发的核心职责
- 适用场景：前端团队职责划分
- 优先级：低
- 规则：
  1. 深度理解用户需求，制定设计策略和视觉方案
  2. 输出完整的 B 端 UI 设计方案与可运行代码

#### 技能要求
- 描述：定义前端开发的技能要求
- 适用场景：前端团队技能评估
- 优先级：低
- 规则：
  1. 准确解读需求，拆分功能点，提炼设计要点
  2. 制定符合用户习惯和业务目标的设计方案
  3. 构建清晰的信息架构和流畅的交互流程
  4. 运用色彩、字体、布局创造美观统一的界面

## 变更日志
- 2026-09-02：版本 2.7.12 - DESIGN.md 视觉基线落地为仓库根文件 + `design-md` 技能；禁止上游品牌覆盖
- 2026-08-19：版本 2.7.11 - 执行规则增补：副作用按钮须防重放（交叉引用 `57_frontend_button_anti_replay.md`）
- 2026-08-16：版本 2.7.10 - 执行规则增补：禁止无触发后台 API 轮询（交叉引用 `51_no_service_internal_poll_loop.md`）
- 2026-08-13：版本 2.7.9 - 执行规则增补：无必要禁止 Teleport / createPortal（交叉引用 `50_no_unnecessary_vue_teleport.md`）
- 2026-07-19：版本 2.7.8 - 「大型组件行数门禁」增补：**门禁触发后须立即自动削减至 ≤ 阈值**；交叉引用 `27_source_file_line_limit_auto_reduce.md`；门禁脚本路径写明 `task2app/scripts/ci/`
- 2026-07-17：版本 2.7.7 - 执行规则增补：入口 HTML head 须声明 `trae-service` meta（交叉引用 `26_frontend_head_trae_service.md`）
- 2026-07-22：版本 2.7.8 - `npm run build` 内置 collectstatic；禁止仅用 `build:vite` 收尾公网
- 2026-07-15：版本 2.7.5 - 执行规则增补：公网 SPA 前端改动后必须 `runall-lifecycle.sh build`（build + collectstatic）；交叉引用 `front_project/ai.md`
- 2026-05-17：版本 2.7.4 - 执行规则与 JS 加载规范增补：静态资源裸 URL 须带内容 hash 查询参数（`18_static_resource_cache_bust_query.md`）
- 2026-05-16：版本 2.7.3 - 执行规则增补：前端与 API 分域名时须遵守 `01_project_constraints/17_cross_domain_frontend_api.md`
- 2026-05-11：版本 2.7.2 - 「大型组件行数门禁」阈值自 1000 行收紧为 **500 行**；与 `project_rules.md` 5.93.3、`check_frontend_component_line_limit.py` 默认上限一致
- 2026-05-10：版本 2.7.1 - 门禁脚本 CI 模式补充「遗留超大文件」相对基线净增长容差说明（`TASK2APP_FRONTEND_LEGACY_GROWTH_GRACE`）；pre-commit 仍为严格默认
- 2026-05-10：版本 2.7.0 - 新增「大型组件行数门禁（自动拆分）」：单文件超过 1000 行须在触及文件时拆分至阈值内或同步收窄；同日接入 `scripts/ci/check_frontend_component_line_limit.py`（pre-commit + CI）
- 2026-04-11：版本 2.6.0 - 新增 DESIGN.md 视觉基线规范，并引入 `02_design_md_integration.md` 作为前端统一视觉治理子规则
- 2026-04-11：版本 2.5.0 - 新增错误反馈展示规范，要求用户可见错误采用 toast 友好提示，同时保留调试细节用于研发定位
