# [运行时] 引导多仓克隆失败时日志与关联仓库列表不点名失败仓

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-17
- 最后修改：2026-07-17
- 维护者：Trae AI 团队

## 现象

- 任务详情（`relayToTrae=true`）容器启动后，引导克隆日志末尾仅：`【项目克隆】已结束（存在失败）。`
- 用户无法从摘要判断哪几个仓库失败（需在数十段 git 输出中人工翻找 `[bootstrap-clone] 克隆失败`）。
- 关联项目仓库列表：失败仓无「克隆失败」徽章；「重新克隆」仅在 `containerEndpointRegistered` 时显示——克隆失败时常尚未注册 endpoint，按钮缺失。

复现任务示例：`relayToTrae.git`、`scripts.git` 因 GitLab not found / exit 128 失败，其余仓成功。

## 根因

1. `onlineServiceJS` `cloneReposIntoSharedLayer` 汇总失败后只写笼统「存在失败」，未输出 `failedJobs` 列表。
2. 前端从 bootstrap 日志解析失败仓时，页脚格式缺失导致无法驱动 per-repo `kind === 'error'`。
3. 「重新克隆」门禁过严：依赖容器 endpoint 已注册，与「引导克隆失败、需立即重试」场景冲突。

## 解决方案

- 后端：`formatBootstrapCloneFailureFooter(failedJobs)` 输出 `失败仓库（N）：` + `- name — url（err）`；全局摘要改为部分失败且**不 abort**（`resolveBootstrapCloneFailurePolicy` → 引导继续 / `BOOTSTRAP_COMPLETE`），避免误杀业务端点就绪。
- 前端：`extractBootstrapCloneFailedRepoUrls` / `isCloneProgressFailureMessage`（含「未完成」）驱动进度与徽章；`shouldShowRepoRecloneButton` = endpoint 已注册 **或** `relayToTrae` **或** 已判定 error。
- 单测：`bootstrap.cloneFailureFooter.test.mjs`；`taskDetailContainerCloneProgress` / `TaskDetailLinkedProjectsPanel` vitest。

## 预防

- 多仓并行失败汇总必须点名实体（URL/目录名），禁止仅布尔「存在失败」。
- 失败态 UI 控件（重试）不得仅绑「成功路径前置条件」（如 endpoint 已注册）。
- 页脚解析正则对 URL 用非贪婪，避免吞掉全角括号内的 err。

## 验证

```bash
cd trae-agent/onlineServiceJS && node --test src/bootstrap.cloneFailureFooter.test.mjs
cd taskFE/app && npx vitest run \
  src/utils/taskDetailContainerCloneProgress.test.js \
  src/composables/taskDetail/taskDetailCloneProgress.test.js \
  src/components/task-detail/TaskDetailLinkedProjectsPanel.test.js
```

公网生效需重建/推送 onlineServiceJS 镜像并发布前端。
