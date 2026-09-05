# 意图：评论级项目克隆进度条（启动 TraceId 下方）

## 背景与目标

评论「执行细节」启动日志已含 `【项目克隆】… N%` 行（Cloud 将 `container_git_clone_progress` 以 `event_name=server_status_update` 写入 binding 日志），但只出现在可滚动「启动日志」文本里，扫读成本高。

目标：对**该评论**的克隆进度，以**总进度条**显示在对应评论「启动 TraceId」下方；多仓时总进度 = 各仓进度之和 / 仓库总数（`(i/n)` 的 n），子进度默认折叠，点开可见；子仓失败会自动重试，3 次仍失败后出现「手动重试」。

## 范围与边界

- 范围内：taskFE 评论执行细节 UI；从该评论 `statusLogs` 解析进度；SSE `container_git_clone_progress` 按 `comment_id` 镜像到评论级 map，供实时条；失败耗尽后复用既有 `POST /api/cloud/repo-reclone`。容器 `parseGitCloneProgressPhases` 的 overall 取 Receiving 与解压/Checkout 的较大值；同仓 `postCloneProgress` 串行发送，避免 100% 被迟到的 9% HTTP 覆盖。
- 范围外：不改 onlineServiceJS 自动重试次数（默认 3，含首轮）。关联项目区任务级克隆进度条已由评论级身份增量移除（见 `034_comment_level_repo_identity`）；评论执行细节进度条仍为本意图范围。

## 约束与风险

- 纯前端展示，无服务端状态变更。
- 并行评论各自独立 CSC 时，进度必须按 `comment_id` 隔离，禁止后一次 SSE 覆盖前一条评论的条。
- 冷打开无 SSE 时须能从 binding 启动日志还原条。

## 验收标准

1. 启动日志含 `【项目克隆】(i/n) name … N%` 时，该评论 `comment-execution-container-meta` 内、启动 TraceId 下方出现 `data-testid="comment-execution-clone-progress"` 进度条，总百分比 = 各仓 progress 之和 / max(n, 行数)。
2. 多仓时默认只显示总进度条；点开 `comment-execution-clone-progress-sub` 后可见各仓库子进度。
3. `准备第 X/Y 次重试` 为自动重试中（非失败、无手动按钮）；失败且未在重试时显示「手动重试」。冷打开 binding 日志往往只有仓库名（如 `失败 ram-work: git exit 128`）而无 git URL：须用任务关联仓库目录 / 引导克隆日志 / 失败文案内嵌 URL 补全 `repoUrl` 后再显示按钮。点击走既有 `POST /api/cloud/repo-reclone`，**必须**带本评论 `comment_id`（path kv + body）。无 comment_id 不发请求。有 comment_id 时**不得**因任务页 `containerEndpointRegistered === false` 短路为「容器未启动」（引导克隆失败时常尚未登记业务端点；由 Cloud 按 comment_id 解析 CSC，真实不可达时返回 409/404 并带 traceId）。全局「部分失败/均失败」摘要行不单独占一行、不展示无 URL 的重试按钮。子仓失败须带 `parent_repo_url` / `clone_alias`。
4. 日志无克隆相关行且无该评论 live map 时不渲染进度条。
5. SSE 带 `comment_id` 时，条随进度更新，不必等 binding 轮询回填日志；live map 与日志按仓库身份合并，避免已完成仓从总分母中丢失。
6. 切 Tab「服务器运行状态」后进度条仍可见（与容器名 / CSC / TraceId 同属 Tab 外 meta）。
7. 克隆已完成后，迟到的低百分比（日志行、live SSE、`segment.recv_progress`）不得把该仓从 100% 打回 9%/3%；重试/重新克隆/失败文案除外。容器 stderr 尾窗同时出现 `Receiving objects: 3%` 与 `Checking out files: 100%` 时，上报 overall=100。容器引导日志已含 `【项目克隆】克隆完成` 或 SSE `仓库克隆已完成` 时，评论条须保持分仓 100%，不得被后续无 `repo_url` 的完成事件清空后再被 3% 覆盖。
8. 项目 `auto_clone_nested_repos=false` 时，克隆 job 列表与评论进度分母只含父仓；带 `parent_repo_url` 的子仓行不得进入 `overallCloneProgressPct`（避免 1 个父仓 100% + N 个空子仓 ≈ 3%）。日志 `(1/1)` 表示未把子仓算进克隆任务数。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 评论级克隆进度条展示 | — | — | — | — | 无对应事件：纯前端展示，复用既有 SSE `container_git_clone_progress` |

## 实施计划

1. `parseCommentCloneProgressFromLogs` / `resolveCommentCloneProgress` 纯函数 + 单测。
2. SSE 处理抽出 `applyContainerGitCloneProgress`，按 `comment_id` 写入 `containerCloneProgressByCommentId`。
3. `TaskDetailCommentCloneProgress` 挂在 `TaskDetailCommentExecutionDetails` 的 TraceId 下方。
4. 进度单调：`shouldKeepCloneProgress` + 同仓 POST 串行；git overall = max(recv, unpack)。

## 变更记录

| 日期 | 相对旧版 | 原因 |
|------|----------|------|
| 2026-08-20 | 全局「仓库克隆已完成」保留分仓 100%；迟到 3% 不得回退；auto_clone=false 不计 nested 分母；git Checkout 100% 不被 12k 尾窗 Receiving 3% 覆盖 | 未开自动克隆子仓时进度 100% 后又变成 3% |
| 2026-08-20 | 引导日志「克隆完成」时评论条收敛到 100%（冷打开 + live 9%） | 评论条只解析启动日志/SSE，忽略容器引导完成态 |
| 2026-08-20 | 进度单调：完成后不被迟到 9% 覆盖；git overall 取 max(recv, unpack)；同仓进度 POST 串行 | 大仓 ram-work 容器 UI 已 100%，SaaS 评论条卡在 9% |
| 2026-08-17 | 有 comment_id 时不因任务级 endpoint 未注册短路「容器未启动」 | 手动重试被前端门禁拦住，请求未发出 |
| 2026-08-17 | 冷打开失败行按仓库名补 git URL 后显示「手动重试」；引导失败文案内嵌 URL | binding 日志只有 `失败 ram-work` 无 URL，按钮被藏 |
| 2026-08-17 | 手动重试带 comment_id；1/1 全失败不再写「其余已就绪」；去掉无 URL 的全局摘要重试行 | 重试 400「缺少评论ID」；全失败文案误导 |
| 2026-08-16 | 多仓改为总进度条 + 折叠子进度；3 次失败后手动重试 | 子仓全展开导致 30+ 条进度挤在执行细节里，总分母未按 (i/n) 计算 |
| 2026-08-15 | 初版 | 评论启动日志已有克隆进度，需在 TraceId 下可视化 |
