# [运行时] task-task/project SQLite malformed → LAUNCH_PROCESS_EXITED

## 基本信息

- 版本：1.0.0
- 案例编号：RT-20260722-0078
- 录入日期：2026-07-22
- 最后更新：2026-07-22
- 录入人：Cursor Agent

## 现象

- runAll 启动 `task-task-service`（及同批 `task-project-service`）失败：
  `LAUNCH_PROCESS_EXITED` / `launch process exited before readiness (exit=1)`
- UI/`error` 附带的 **recent stderr** 只见 `[confload] base.yaml …` 与 config 摘要，**不见真正致命原因**
- 完整 tee 日志（stdout）可见：
  `[taskTaskService] db: migration: database disk image is malformed (11)`

## 环境与上下文

- 工作区：`/tmp/ram-work`（tmpfs）
- 库路径：`data/task_task.db`、`data/task_project.db`（`dbload.ResolveDatabasePath`）
- 相关：`.learnings/OPTIMIZATION_TODOS.md` 中 `OPT-20260722-063`（bulk URL rewrite 曾写空 0 字节库）

## 根因

1. SQLite 文件损坏（`PRAGMA integrity_check` → `database disk image is malformed (11)`），migration/`CREATE TABLE` 即失败 → 进程 exit 1。
2. 致命日志经结构化 logger 落在 **stdout**（且常为 `level=info` 包裹的 `log.Fatalf` 文案）；runAll 对 `LAUNCH_PROCESS_EXITED` **优先拼 stderr**，stderr 只有 confload → 表象误导。

## 修复

1. 备份损坏库到 `data/corrupt-backup/`，删除 `data/task_task.db` / `data/task_project.db`（及 wal/shm）。
2. 经 runAll `POST /api/start`（需 `session_id`）重启两服务；migration 重建空库后就绪。
3. runAll：`enrichStartupFailureWithRecentLogs` 在 stderr 无 fatal 特征时，改附带 stdout/全流中含 `malformed` / `"level":"error"` 等的 **recent logs**。
4. 数据恢复（OPT-20260722-063）：对损坏库 `sqlite3 .recover` → 解析 `lost_and_found`，脚本 `data/corrupt-backup/restore_from_lost_and_found.py` 写回业务表（project 侧较完整；task 侧仅部分全量行 + 关联 stub）。

## 验收

```bash
sqlite3 data/task_task.db 'PRAGMA integrity_check;'   # ok
sqlite3 data/task_project.db 'PRAGMA integrity_check;'
curl -sS http://127.0.0.1:8017/api/health             # 200
curl -sS http://127.0.0.1:8016/api/health             # 200
# runAll UI：两服务 status=healthy
cd runAll && go test ./src -count=1 -run 'TestEnrichStartupFailureWithRecentLogs'
```

## 预防

- tmpfs 磁盘压力/异常断电后，先 `PRAGMA integrity_check` 再查代码。
- 启动失败看 tee：`logs/task-task-service.log`，勿只信 UI recent stderr。
- 业务进程启动失败优先 `tracelog.Fatalf`（`level=error`）；勿依赖 stderr 噪音诊断。
- 本地可接受丢数时可用 runAll Dev：`/api/dev/clear-databases` + `init-databases`（需 `confirm=`）。

## 相关

- `taskTaskService/src/db.go` / `main.go`
- `taskProjectService/src/db.go` / `main.go`
- `runAll/src/runner.go`（`enrichStartupFailureWithRecentLogs`）
- `OPT-20260722-063`（业务数据恢复，若需）
