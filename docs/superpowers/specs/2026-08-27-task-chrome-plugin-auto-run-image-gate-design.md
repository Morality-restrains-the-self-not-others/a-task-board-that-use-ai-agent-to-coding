# 设计：Chrome 插件自动运行与已安装镜像必选挂钩

- **Date:** 2026-08-27
- **Status:** accepted（`/goal` 零交互）
- **Architecture artifacts:** 非架构变更（无新服务/API/事件/数据所有权；仅插件 UI 与纯函数）

## 🕸️ Code Review Graph 分析

CRG `update --brief` 已执行（增量文件、风险 0）。本增量不改后端符号。

既有调用链：

- 服务端门禁：`taskTaskService/src/auto_run.go#validateAutoRunPrerequisites` — `imageID` 为空则 `AUTO_RUN_IMAGE_REQUIRED`（「自动运行需要选择已安装镜像」）
- 工作面板：`taskFE` `useCreateTaskAutoRun` + `autoRunGateHints.js` — 无已选镜像则 `canEnableAutoRun=false`，提示「请先选择已安装镜像」
- 插件浮窗：`content.js` 镜像 `<select id="taskplugin-image">` 默认「无」，自动运行仅按项目 `default_auto_run` 启用；创建走 `CreateTaskPayload.validateCreateTaskForm`（不校验镜像）
- 插件 DevTools：`#singleContainerImage` / `#batchContainerImage` 同样可选空；`workspace-projects.js#syncContainerAutoRun` 不读镜像
- 同类挂钩先例：`lib/create-task-git-identity.js` — Git 身份仅在 `auto_run===true` 时必选

上一增量 `task_chrome_plugin_project_single_select_auto_run` 明确把「镜像/硬件模版完整启机门禁 UI」列为范围外。本增量补镜像门禁。

## 当前架构理解（裁剪）

企业景观 v114 current。taskChromePlugin 仍只读既有 GET 已安装镜像与 POST 创建任务。不新增组件、不改服务间 Rel_Flow。🐍 Python 新增接口：not_applicable。

## 问题

用户在浮窗勾选「是否自动运行」后点创建，请求打到后端才返回 `❌ 创建失败: 自动运行需要选择已安装镜像`。镜像下拉默认「无」，UI 未把「镜像必选」与自动运行绑定。

## 方案（选定）

**双向挂钩，对齐工作面板 `canEnableAutoRun`，并加提交门禁兜底。**

1. **自动运行依赖镜像（主路径）**  
   扩展 `resolveAutoRunControlState`：在项目已选且允许自动运行之后，若未选已安装镜像 → `{ enabled: false, checked: false, hint: '请先选择已安装镜像' }`。  
   选中镜像后才启用自动运行；允许自动运行的项目在镜像就绪时默认勾选（与上一增量一致）。用户可再取消勾选。清空镜像则强制取消自动运行。

2. **镜像必选外观跟自动运行挂钩**  
   仅当所选项目允许自动运行时，镜像标签显示 `*自动运行必选`，空选项文案为「请选择已安装镜像」，`aria-required=true`。项目不允许或不选项目时，空选项仍为「无」，不标必填（创建任务可不带镜像）。

3. **提交门禁（兜底，对齐后端语义）**  
   `validateCreateTaskForm`：`auto_run===true` 且无 `container_image_id` → 返回「请先选择已安装镜像」，不发请求。  
   纯前端校验，不伪造 `data-traceId`。

4. **入口**  
   浮窗、DevTools 单请求、DevTools 批量三处共用同一纯函数。镜像 `<select>` `change` 后重算自动运行状态。

5. **使用说明**  
   `docs/USER_GUIDE.md` + `lib/user-guide.js` 写明：自动运行须先选已安装镜像；不自动运行时镜像可选。

### 未采用

| 方案 | 拒绝原因 |
|------|----------|
| 仅提交时报错、不禁用勾选 | 用户已经踩过这条路径；工作面板是禁用勾选 |
| 无自动运行也强制选镜像 | 与后端契约不符；非自动运行任务允许无镜像 |
| 禁用自动运行直到镜像，但不改标签 | 用户看不到「为什么勾不了」与镜像字段的关系 |
| 新 API / 改 taskTaskService | 后端门禁已正确；缺陷在插件未前置 |
| 完整对齐硬件模版门禁 | 超出本目标；模版仍由后端 `AUTO_RUN_RUN_TEMPLATE_REQUIRED` 校验 |

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|----------|--------|--------|--------|---------|
| 勾选自动运行前选择已安装镜像 | （无） | — | — | **无对应事件**：纯客户端 UI 门禁；创建任务成功仍走既有任务创建事件 |

## 价值流影响

既有「插件创建任务」流增加客户端门禁：自动运行 ↔ 已安装镜像。无新 stream YAML（插件 UI 增量，与上一 chrome 插件切片一致）。

## 🏛️ 架构变更影响

- **迭代版本:** 不升版
- **理由:** 无服务/接口/数据流/基础设施增删改
- **已有文件（未修改）:** `docs/architecture/v114-*-20260827-0200-cursor.*`（current）
