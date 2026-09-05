# [运行时] 任务详情文件树「layer not found」与执行日志「server_url 未就绪」

## 失败现象

任务详情 `…/task-detail/task_*/` 评论执行细节同时出现：

1. `[data-testid="project-file-tree-error"]` 红字 `layer not found`（带 `data-traceId`）
2. 执行日志红字 `容器 server_url 未就绪，无法拉取执行日志`

## 环境与上下文

- 页面：独立任务详情（非 work-panel 弹窗）
- 评论级 CSC：`cmt_876722826143363072` 等可转发到容器；另一评论 `cmt_876723217694224384` 仍 `container_target_missing_base_url`（409）
- 样例 traceId：`87d18b17-707a-4ab7-aa36-6d796f0f0a26`（2026-08-16 15:45:46 +08）

## 排查过程

1. Loki `{job=~".+"} | json | trace_id` 1h/24h 为空（采集管道当时几乎只有 collection_status）；D5 在 `logs/task-container-gateway.log` / `task-cloud-service.log` 命中该 trace。
2. 时间线：gateway `cloud_resolve` 200 → 上游 `http://…:8765/…/layers/20260816_074513_e0a004/children` **404** `layer not found`。容器目录不存在（`listLayerChildren` ← `layerGitWorkdirRootsForFileListing` 空）。
3. 执行日志文案来自前端 `refreshZTreeExecutionLog`：页面级 `containerEndpointRegistered === false` 时**不发请求**直接写红错。文件树不看该旗标，仍按评论 `comment_id` 转发，因此同一面板可同时出现「已打到容器的 404」和「前端以为没 server_url」。

## 根因

1. 层图节点上的 `layer_id` 在该评论容器文件系统上尚不存在（新建叠层 / 跨评论共用页级层图 / 克隆未完成）。
2. 执行日志用**任务页单例** `containerEndpointRegistered` 做门禁，与评论级 CSC 的 `server_url` 不同步。

## 解决方案

1. 404 `layer not found`、409 缺业务地址 → 中文等待态（amber），保留 `data-traceId`。
2. 有 `comment_id` 时执行日志仍走评论级转发；端点 false→true 时重置刷新。
3. 不再把「容器 server_url 未就绪…」写入 `layerExecLogTopError`（该字段会挡住克隆/任务日志区）。

## 预防

- 容器 compute 暂态（404 层目录、409 未注册地址）不得当阻断红错。
- 评论级转发与页面级 endpoint 旗标不得混用为唯一门禁。
