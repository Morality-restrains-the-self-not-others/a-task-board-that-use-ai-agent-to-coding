# 意图：Workbench 链接读取当前物理机 CSC（评论级回退）

## 背景与目标

任务详情评论区「Workbench 访问实例」调用 `GET compute/workbench-link`。容器已启动（runtime-status 为 Running）时仍报「服务器配置缺少实例ID或地域」。

根因：`handleWorkbenchLink` 只读**任务级** CSC（`comment_id=''`）。评论级独立实例把 `instance_id`/`region` 落在评论 CSC 行上；`server-runtime-status` / `stop-vm` / `server-content` 已改用 `loadCloudServerConfigForRuntime`，Workbench 未对齐。

目标：Workbench 与运行态读同一条「当前物理机」CSC；**评论卡片传入 `comment_id` 时只读该评论行**（即使 instance 为空也不回退）；无 `comment_id` 时任务级缺实例/地域则回退评论级，并在实例已有、地域为空时从同任务其它 CSC 回填地域。

## 范围与边界

- 范围内：`taskCloudService` `handleWorkbenchLink`；可选 `comment_id` 查询参数优先该评论 CSC。
- 范围外：不改 Aliyun Workbench URL 格式；不新增云厂商平台。

## 约束与风险

- 只读 GET，不改变 CSC 行；多评论并行时与 runtime-status 一样取「最新带 instance_id 的行」（可用 `comment_id` 精确指定）。
- Mock 实例仍拒绝（本地 Docker 无 ECS Workbench）。

## 验收标准

1. 任务级 CSC 无 instance_id，评论级有 instance_id + region + aliyun → 200 且 `workbench_url` 含该实例与地域。
2. 任务级有 instance_id 但 region 空、同任务其它 CSC 有 region → 200，地域取回填值。
3. 任务下任何 CSC 都无 instance_id 或都无 region → 仍 400「服务器配置缺少实例ID或地域」。
4. 既有成功/缺失配置/Mock 拒绝测例仍通过。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 打开 Workbench 链接 | — | — | — | — | 无对应事件：只读查询，生成阿里云控制台 URL，无业务状态变更 |

## 实施计划

1. `handleWorkbenchLink` 改 `loadCloudServerConfigForRuntime`；`comment_id` 非空时优先评论 CSC。
2. 地域空则从任务级或同任务带 region 的 CSC 回填。
3. 回归单测覆盖评论级实例与地域回填。
4. 前端失败文案挂本次请求 `data-traceId`。

## 变更记录

- 2026-08-13：修复评论级 CSC 与任务级模板漂移导致 Workbench 400。
