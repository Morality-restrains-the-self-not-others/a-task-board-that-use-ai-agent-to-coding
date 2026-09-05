# 意图：任务详情文件树 / 执行日志对「层未落地、地址未注册」显示等待态

## 背景与目标

任务详情评论执行细节中，项目文件树曾把容器 `404 layer not found` 原样画成红错；执行日志在页面级 `containerEndpointRegistered === false` 时直接写「容器 server_url 未就绪，无法拉取执行日志」，甚至不发起评论级转发。二者都是启动过程中的暂态（叠层目录尚未落地、register-reachability 尚未写入 `server_url`，或页面级就绪旗标与当前评论 CSC 不一致），应显示等待提示并在就绪后自动再拉，而不是阻断红错。

## 范围与边界

- 范围内：`TaskDetailProjectFileTree`、`refreshZTreeExecutionLog`、执行日志错误节点样式；映射 `layer not found` / 409 缺业务地址。
- 范围外：不改容器 overlay 创建、不改评论级 CSC 绑定模型、不把层图改为每评论一份快照（记入 OPT）。

## 约束与风险

- 等待态须保留 `data-traceId`（若该次请求有 trace）。
- 禁止为等待态新增 `setInterval` 轮询；依赖已有 endpoint / nonce / 层图 watcher。
- `containerEndpointRegistered === false` 时仍应尝试评论级转发（CSC 可能已有 `server_url`）。

## 验收标准

1. 文件树收到 `404` + `detail: layer not found` 时：`project-file-tree-error` 不出现；`project-file-tree-waiting` 显示中文等待文案，带 traceId。
2. 执行日志在 `containerEndpointRegistered === false` 且有 `comment_id` 时仍发 clone/job 请求；不再把「容器 server_url 未就绪…」写入 `layerExecLogTopError`。
3. 409「尚未注册可用业务地址」映射为等待文案，不以页面级硬错误挡住克隆/任务日志区。
4. `containerEndpointRegistered` 从 false→true 时重置刷新执行日志。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 暂态等待替代硬错误 | — | — | — | — | 纯前端展示与请求门控，无新服务端事实 |

## 实施计划

1. 抽取 `containerComputeWaiting` 映射函数与单测。
2. 文件树 / 执行日志消费映射；错误节点 amber 等待态。
3. watcher：端点就绪后 `refreshZTreeExecutionLog({ reset: true })`。
