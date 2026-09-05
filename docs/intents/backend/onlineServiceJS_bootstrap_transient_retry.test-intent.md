# onlineServiceJS 引导瞬时断连 — 测试意图

## 覆盖点

1. **瞬时重试成功**：连续 2 次 `ECONNREFUSED` 后第 3 次 200，`postJson` 返回成功且 outbound 日志含 `transient retry`。
2. **重试耗尽**：全部失败时抛出含 `ECONNREFUSED`/`fetch failed` 的错误。
3. **4xx 不重试**：HTTP 401 仅请求 1 次。
4. **runAll**：`GracefulShutdownSelf=true` 时 `cleanupOrphanManagedServices` 早退，不 panic。

## 前端单测

1. `classifyBootstrapMilestoneLogLine`：`BOOTSTRAP_COMPLETE` / 旧「任务引导完成」→ `complete`；`BOOTSTRAP_FAILED` / post-listen error → `failed`；`BOOTSTRAP_PHASE=` → `phase`。
2. `resolveBootstrapStatusFromLogs`：空日志 → `idle`；phase → `running`；complete → `complete`；failed 优先且 label 含阶段。

## 手工回归

1. 保持 `task-agent-support`(:8011) / `task-cloud-service`(:8018) / `task-credential-service`(:8015) 健康。
2. 若页面会话因服务重启掉线：重新登录后再打开任务详情 `?relayToTrae=true`，点「启动」。
3. 期望启动日志上方出现状态条徽章：`引导中` / `引导完成` / `引导失败`；完成行与失败行仍高亮。
4. runAll 热替换：`cd runAll && ./build.sh` 须打印 `Verified skip-orphan capability`；`grep -aF 'skip orphan port cleanup' bin/runAll` 成功。
5. 可选：启动中对 :8011 短暂 `kill -STOP`/`CONT` 或重启 `task-agent-support`，引导应在重试窗口内恢复。

## 变更记录

- 2026-07-10：与实现同批建立。
- 2026-07-10：补充 `BOOTSTRAP_*` 前端分类单测与手工回归说明。
- 2026-07-10：补充引导状态条/失败徽章与 runAll skip-orphan 构建校验回归。
