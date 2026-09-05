# 功能意图：任务详情展示子仓库克隆状态

- **日期**: 2026-07-17
- **状态**: 已完成
- **相关**: `project_nested_git_repos`（发现）、`container_nested_git_repos_clone`（容器下发与克隆）

## 用户故事

作为任务协作者，我在任务详情「关联项目」区域（「克隆所用 Git 身份」附近）看到父仓下各**子仓库的克隆状态**（未开始 / 排队 / 克隆中 / 已完成 / 失败），以便在元仓项目（如 `ram-work`）启动容器后确认子仓是否已就绪，而无需只看父仓进度条。

## 验收标准

1. 关联项目含可发现子仓时，在对应父仓条目下展示「子仓库克隆状态」区块（`data-testid="task-nested-repos-clone-status"`）。
2. 每行展示：状态徽章、path、短 URL、进行中时的进度百分比。
3. 汇总文案：`已完成数/总数`；进行中/失败另有计数。
4. 状态来源：SSE/进度 Map 按 URL 匹配；引导克隆日志已完成时，无进度行可视为已完成。
5. 容器未启动：状态为「等待容器」；容器已就绪尚无进度：「未开始」。
6. 无子仓时不展示空区块（避免噪音）；发现失败展示局部错误，不阻断身份选择。
7. 提供「刷新」重新拉取 nested-git-repos。
8. 不把子仓写入 `project_repos`；不新增 Django/Python 公网接口。

## 范围

- 前端：`TaskDetailNestedReposCloneStatus.vue` + `nestedRepoCloneStatusUtils.js`
- 复用：`useProjectNestedGitRepos` → `GET …/nested-git-repos/`；既有 `cloneProgressEntryByRepoMatchKey`
- 不含：子仓独立「克隆所用 Git 身份」下拉（凭证继承父仓/任务身份，见容器 enrich）

## 业务意图 → 事件对照

**无对应事件**：只读展示 + 既有查询，无平台业务状态变更。

| 业务意图 | 事件名 | 例外理由 |
|---------|--------|---------|
| 任务详情展示子仓库克隆状态 | — | 纯查询/展示例外 |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-17 | 初版 |
| 2026-07-17 | 完成组件入库接线、`bootstrapCloneDone` 下传与单测；公网 SPA build+collectstatic |
