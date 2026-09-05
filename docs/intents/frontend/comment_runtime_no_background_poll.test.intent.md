# 测试意图：评论运行态禁止无触发后台 Describe 轮询

## 测试目标

证明 Initializing / isServerStarting 不再启动 Describe 定时器；合法触发仍带正确 `comment_id`。

## 测试分层

- 单元：`serverRuntimeHydrate` / `useServerConfigRuntime` 假时钟
- 可选 Playwright：任务详情启动中无周期性 runtime-status 请求

## 用例矩阵

| ID | 给定 | 当 | 则 |
|----|------|----|-----|
| T1 | runtime_status=Initializing | 推进 15s | 0 次 interval `fetchServerRuntimeStatus` |
| T2 | isServerStarting=true 且 status 空 | 推进 60s | 0 次 30s fallback Describe |
| T3 | 运行态面板 commentId=C2 | 挂载（不点按钮） | 0 次 GET runtime-status |
| T4 | SSE 启动成功 payload.comment_id=C1 且 runtime_status=Running | 处理消息 | snapshot 写入 C1，**零**自动 Describe |
| T5 | 用户点「刷新状态」 | 点击 | 1 次 GET，id 为该面板评论 |
| T6 | 源码 | 扫描 | 无 `startingFallbackPollId` / 无 runtime poll controller / 无 `serverStartupStatusPoll` interval / 无 binding 30s timer |

## 通过标准

T1–T6 全绿；无新增 HTTP 契约测试（无新接口）。
