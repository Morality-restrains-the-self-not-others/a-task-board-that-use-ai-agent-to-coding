# [运行时] 子仓库已克隆移入父仓后状态仍显示「未开始」

## 现象

任务详情「子仓库克隆状态」汇总为 `0/N 已完成`，各行徽章均为「未开始」；同时项目文件树已能看到子仓目录（如 `DaydaymoneyGrafana`、`AiMonitor`）。

## 根因（非「移入后查不到磁盘」）

状态 UI **不查容器磁盘**，只看：

1. SSE `containerCloneProgressByKey`（按仓库 URL）
2. `bootstrapCloneDone`（引导日志表明整体完成时，无进度行视为已完成）

实际链路：

1. nested 子仓先 staging 再 `relocate` 进父仓；成功时会 `postCloneProgress(..., 100, …已移入…)`。
2. 全部成功后全局事件 `【项目克隆】仓库克隆已完成` 触发前端 **清空整个进度 Map**（隐藏克隆横幅）。
3. `bootstrapCloneDone` 虽在 `TaskDetailNestedReposCloneStatus` 有 props，但 **RuntimeSection 从未传入**，恒为 `false`。
4. 于是：无进度行 + 非 done 回落 → 全部「未开始」。文件树来自 layer files，故与状态脱节。

用户直觉「移入父仓导致状态查不到」接近表象：移入后进度事件已结束并被清空，但缺少日志回落接线。

## 修复

1. 由引导日志计算 `bootstrapCloneDone` 并透传到关联项目面板。
2. 无进度行时解析引导日志段：含「已移入」→ 已完成；含克隆/移入失败 → 失败。
3. 容器就绪时 `GET …/container-bootstrap-clone-log/` 回填日志（覆盖刷新页面）。
4. SSE 空 `bootstrap_log_text` 不覆盖已有完整日志。

## 验证

```bash
cd taskFE/app
npx vitest run \
  src/utils/nestedRepoCloneStatusUtils.test.js \
  src/composables/taskDetail/taskDetailCloneProgress.test.js \
  src/components/task-detail/TaskDetailNestedReposCloneStatus.test.js
```
