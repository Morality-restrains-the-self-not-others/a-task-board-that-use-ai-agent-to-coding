# 测试意图：服务器运行状态评论归属（无全局 header）

## T11 — 实例详情展示流量带宽（2026-08-17）

- 给定 runtime-status 的 `instance_attribute.body` 含
  `InternetChargeType=PayByTraffic`、出/入带宽均为 5
- 当 `buildServerRuntimeStatusDetails` 构建实例详情行
- 则出现「带宽计费模式：按流量计费」「公网出带宽：5 Mbps」
  「公网入带宽：5 Mbps」
- 给定仅 EIP 带计费模式/`Bandwidth`、实例公网带宽字段缺失
- 或实例出带宽为 0 且 EIP 带宽 > 0
- 则回退展示 EIP 计费模式与出带宽；不展示入带宽行
- 给定带宽字段均缺失
- 则不展示上述三行
- 后端：`mapDescribeInstancesInstance` 透传
  `InternetChargeType` / `InternetMaxBandwidthOut` /
  `InternetMaxBandwidthIn` /
  `EipAddress.{IpAddress,InternetChargeType,Bandwidth}`

## T10 — 评论列表顶端不得渲染全局镜像运行 header（2026-08-15）

- 给定任务详情评论区已有评论，且服务器正在运行
- 当渲染 `TaskDetailCommentsPanel`
- 则不存在 `[data-testid=image-runtime-entries-section]` / `[data-testid=image-runtime-entry]`
- 并且源码管道不再包含 `buildImageRuntimeEntries` / `imageRuntimeEntries` 透传
- 启动日志、SSE、镜像关联信息仅出现在对应评论「执行细节」内（F-030）

## T8 — 授权缺失不与「已启动」矛盾（2026-08-11）

- 给定 CSC 有 instance_id + last_runtime_status=Running，authorization_id 已失效
- 当 GET server-runtime-status
- 则 HTTP 200、`runtime_status=Running`、`auth_missing=true`（非 400 导致 FE「未知」）
- 给定租户仍有同平台有效授权
- 则优先用公司级授权 Describe
- 前端：`resolveRuntimeStatusOnAuthError` 在 isServerRunning + 授权文案时对齐 Running

## T9 — 运行状态文案挂 data-traceId（2026-08-11）

- 给定 server-runtime-status 返回 message + trace_id（或 FE 从 response.traceId 解析）
- 当渲染 ServerConfigRuntimeStatusSection
- 则 `[data-testid=server-runtime-status-message]` 带 `data-traceId`
- 后端：writeServerRuntimeStatusJSON / writeErrorMapJSON 回写 body.trace_id 与响应头 X-Trace-Id

## 已退役（不再执行）

- T1–T4：`buildCommentRuntimeServerTabs` / `ServerConfigCommentRuntimeTabs` / 评论区顶部镜像卡片。运行态已迁入评论执行细节 Tab。
