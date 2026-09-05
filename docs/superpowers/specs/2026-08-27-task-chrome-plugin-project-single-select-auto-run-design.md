# 设计：Chrome 插件项目单选 + 自动运行随项目能力切换

- **Date:** 2026-08-27
- **Status:** accepted（`/goal` 零交互）
- **Architecture artifacts:** 非架构变更（无新服务/API/事件；仅插件 UI 与纯函数）

## 🕸️ Code Review Graph 分析

CRG `update --brief` 已执行（增量 7 文件、风险 0）。本增量不改后端符号。

既有调用链：

- 浮窗 `content.js#loadProjects` → 项目复选框列表 → `#taskplugin-auto-run`
- DevTools `panel/lib/workspace.js#renderProjectCheckboxes`（含全选）→ `#singleAutoRun` / `#batchAutoRun`
- 能力判定 SSOT：`lib/project-auto-run-label.js#projectAllowsAutoRun`（`server_run_template.default_auto_run === true`）
- 工作面板对齐参考：`taskFE` `useCreateTaskAutoRun`（无项目/不允许则禁用并强制 `auto_run=false`；切到允许项目时默认开启）

上一增量（项目名称旁徽章）明确把「按标注禁用自动运行勾选」列为范围外，原因是多选。本增量消除该原因。

## 当前架构理解（裁剪）

企业景观不变。taskChromePlugin 仍只读既有 GET 项目列表；创建任务 payload 的 `projects[]` 仍由后端校验。本增量把客户端从「多项目勾选」改为「单项目选择」，并把自动运行开关与所选项目能力绑定。

## 问题

1. 浮窗与 DevTools 项目列表是复选框 +（面板）全选，用户可一次关联多个项目。
2. 「是否自动运行」与所选项目是否允许自动运行无关；不可自动运行的项目仍可勾选自动运行，只能在创建失败后才发现。

## 方案（选定）

**单选 radio + 纯函数驱动自动运行控件。**

1. 浮窗、DevTools 单请求、DevTools 批量：项目列表改为 `input[type=radio]`（同组 `name`），去掉「全选/取消」。徽章渲染仍用 `ProjectAutoRunLabel`。
2. 提交 payload 仍为 `projects: [{ project_id, ... }]`，长度为 0 或 1。创建 API 契约不变。
3. SSOT 扩展 `lib/project-auto-run-label.js`：
   - `pickSingleProjectId(preferredIds, availableIds)`：从候选中取第一个仍存在的 id
   - `findProjectById(projects, id)`
   - `resolveAutoRunControlState({ selectedProject, checkedPreference })`：
     - 无项目 → `{ enabled: false, checked: false, hint: '请先选择项目' }`
     - 不允许 → `{ enabled: false, checked: false, hint: '当前项目未允许自动运行…' }`
     - 允许 → `{ enabled: true, checked: Boolean(checkedPreference), hint: '创建后按项目运行模版启动云服务器' }`
   - `applyAutoRunControlToElements(inputEl, hintEl, state)` 写入 `disabled` / `checked` / 提示文案
4. 切换项目时：若新项目允许自动运行，`checkedPreference` 默认为 `true`（对齐工作面板 `default_auto_run`）；用户随后可手动取消。恢复打开快照时以快照 `auto_run` 为 preference，再经 resolve 钳制。
5. daydaymoney 反查命中多个项目时只选第一个。`lastProjectIds` 恢复时只恢复第一个仍存在的 id。
6. 使用说明两份 SSOT 去掉「项目可多选」，写明自动运行随所选项目能力变化。

### 未采用

| 方案 | 拒绝原因 |
|------|----------|
| 项目改成 `<select>` 下拉 | 会挤掉名称旁「可/不可自动运行」徽章的可读性 |
| 批量创建仍多选 | 用户要求插件内项目改为单选；多选时自动运行无法对应单一项目能力 |
| 完整对齐工作面板镜像+硬件模版门禁 | 本增量问的是项目是否能自动运行；镜像/模版仍由创建路径与后端校验 |
| 新 API | 列表已有 `server_run_template` |

## 风险

- 历史快照 / `lastProjectIds` 含多个 id：只取第一个仍在当前工作空间列表中的 id，其余忽略。
- 服务端仍可能因镜像未选等原因拒绝 `auto_run`：控件只反映项目开关，不假装覆盖全部启机门禁。
