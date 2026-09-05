# boot-progress 落到任务级 CSC（已有 instance）→ 评论启动日志缺 UserData 步骤

- **Date**: 2026-08-12
- **TraceId**: `90f7f3b4e2f2ccb93f8079cd8dbfca95`（及同任务其它 boot-progress traces）
- **Symptoms**: 评论执行细节「启动日志」仅有「容器调度排队中 / 正在启动容器实例 / 容器实例已分配」三行；Loki 已有 `sse_redis_published` 且 message 含「开始容器初始化 / 拉取容器镜像」等；实例 `/root/init_from_task2app.sh` 中 `COMMENT_ID='-'`。
- **Root cause**:
  1. UserData 可选槽 `comment_id` 空 → 注入 `-`，`report_progress` 不带 `comment_id`。
  2. `resolveBootProgressCSC` 在 token 落到**任务级** CSC 且 `InstanceID` 非空时提前 return，**不**再切到评论级 CSC（attach/reuse 常见：任务级与评论级共享同一 instance）。
  3. SSE 无 `comment_id` → 前端 `resolveServerStartupBindingCommentId` 不扇出；任务级 statusLogs 标签为 `task2app-container`，与 binding mock 名 `task_<task>_<comment>` 不一致，`filterTaskStatusLogsForBinding` 原逻辑匹配失败。
- **Fix**:
  - 去掉「任务级已有 InstanceID 则不切换」分支；CommentID 空时始终尝试 `loadNewestCloudServerConfigWithInstance`（优先评论级）。
  - 前端 filter 兼容 `task2app-container` / `instanceId` / 单活跃 binding 无标签云调度行。
- **Verify**: `go test ./src/ -run TestResolveBootProgressCSC`；`vitest` `bindingServerStartupLogs` + `useCommentContainerBindings`；精准重启后下一轮 boot-progress SSE 应带 `comment_id`，启动日志出现安装依赖/拉镜像步骤。
- **Related**: `.ai/09_failure_experience/.../89_userdata_container_name_shell_var_dash_inject.md`；OPT-20260810-035 / OPT-20260811-001；OPT-20260812-046。
