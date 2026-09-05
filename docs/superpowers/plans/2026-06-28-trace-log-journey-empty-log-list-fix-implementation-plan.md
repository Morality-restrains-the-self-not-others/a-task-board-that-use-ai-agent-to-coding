# 实施计划: Trace Log Journey 日志目录对齐修复

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-06-28-trace-log-journey-empty-log-list-fix.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-28-trace-log-journey-empty-log-list-fix-value-stream.md`
> - DDD 文档: `docs/superpowers/plans/2026-06-28-trace-log-journey-empty-log-list-fix-ddd-domain-model.md`

## 变更摘要

将 `conf/runAll.yaml` 的 `logging.file_root` 从 `../logs` 改为 `logs`，使 runAll 与 Promtail 解析到同一日志目录。

## 任务清单

### Task 1: 修复 runAll.yaml 日志根路径

- [ ] 修改 `conf/runAll.yaml`：`file_root: ../logs` → `file_root: logs`
- 文件: `conf/runAll.yaml` 第 15 行
- 验证: `grep "file_root" conf/runAll.yaml` 输出 `file_root: logs`

### Task 2: 加固 Promtail 启动脚本的相对路径解析 (可选)

- [ ] 修改 `runAll/scripts/runall-local-promtail.sh` 的 `resolve_runall_log_root()` 函数
- 将 `else` 分支从 `${ROOT}/logs` 改为 `$(cd "${ROOT}" && realpath "$raw")`
- 文件: `runAll/scripts/runall-local-promtail.sh`
- 验证: `bash runAll/scripts/runall-local-promtail.sh --help` 未报错

### Task 3: 重启服务使配置生效

- [ ] 重启 runAll (使新 file_root 生效)
- [ ] 重启 Promtail: `bash runAll/scripts/runall-local-promtail.sh down && bash runAll/scripts/runall-local-promtail.sh up`
- 验证: `docker exec aimonitor-promtail-local ls /var/log/runall/` 可见服务 .log 文件

### Task 4: 验证 Grafana 仪表盘恢复

- [ ] 打开 `http://183.250.1.132:3000/d/trace-log-journey/trace-log-journey`
- [ ] 选择最近 1 小时时间范围，确认日志列表有数据
- [ ] 输入一个已知 trace_id，验证跨服务日志可检索

### Task 5: 清理残留

- [ ] 删除残留空目录 `/tmp/runall-logs/` (如果存在)
- [ ] 可选择性清理旧日志 `/tmp/logs/` (迁移前的数据)
