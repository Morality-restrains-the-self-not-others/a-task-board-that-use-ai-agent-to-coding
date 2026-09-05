# UserData `${CONTAINER_NAME}` 被 RunInstances 值注入为 `-` → 启动日志缺失且状态卡在「启动中」

- **Date**: 2026-08-11
- **Symptoms**: 任务详情评论执行面板「启动日志」缺少机器节点 UserData 步骤（安装依赖/拉镜像等）；「服务器启动状态」长期「启动中」。实例上 `/root/init_from_task2app.sh` 可见 `COMMENT_ID='-'`、`report_progress` 中 `[ "-" != "-" ]`、`docker inspect … -`。
- **Root cause**:
  1. `replaceUserdataRuntimePlaceholders` 把 `${CONTAINER_NAME}` / `${containerName}` 当作与 `__TASK2APP_CONTAINER_NAME__` 同 slot 的值注入 token；`container_name` 为空时 optional → `-`，破坏 shell 变量与健康检查。
  2. Linux 生成器健康检查使用未 export 的 `${containerName}`（camelCase），依赖上述错误注入。
  3. `linuxBootProgressReportSnippet` 中 `_extra=...` 赋值缺少闭合双引号，`COMMENT_ID` 非 `-` 时 JSON 损坏。
- **Fix**: 仅值替换 `__TASK2APP_*`；`${containerName}` → `${CONTAINER_NAME}` 规范化；健康检查改 `"${CONTAINER_NAME}"`；补齐 `_extra` 引号；`parent_comment_id`/推导 `container_name` 与 SSOT 对齐。
- **Verify**: `go test` userdata replace（taskCloudService + taskEvents）；`node --test tests/userDataBootProgress.unit.test.js`；重生模板后启新实例看 boot-progress。
- **Related**: OPT-20260809-025（旧「值注入」方案已纠正）、OPT-20260811-001/002。
