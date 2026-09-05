# [运行时] task-events fanout 健康检查误用相邻服务端口

## 基本信息
- 案例编号：FE-20260816-EVENTS-FANOUT-HEALTH-PORT
- 录入日期：2026-08-16
- 最后更新：2026-08-16
- 录入人：Trae AI 团队

## 失败现象

`http://10.2.150.68:9999/` 精准编译重启失败：

- 服务：`task-events-task-status-changed-2-fanout-work-panel-sse`
- 错误：`[READINESS_TIMEOUT]` 60s
- 探针：`http://10.2.150.68:18056/api/health/` → `connection refused`

进程是否起来与探针端口无关。wechat 消费者起来后打 18056 会**假阳性 healthy**（探到的是另一进程）。

## 失败环境

- 编排：`conf/runAll.yaml` `health_check.url` / `liveness_url`
- 监听 SSOT：`conf/events/domain-events/task_status_changed/config.yaml` → `2_fanout_work_panel_sse.port: 18048`
- 引入方式：`git blame` `a4b0a097`（2026-08-05 新增微信冲突消费者时拷到 fanout）

## 排查过程

1. 对照 domain-events YAML 与 `taskEvents/config/intent_registry.go`：fanout 监听 **18048**。
2. `conf/runAll.yaml` 该条目探针写成 **18056**（与 `wechat-identity-conflict` 相同）。
3. 全量对照：这是当时唯一 runAll↔SSOT 端口错配。另有 9 个 SSOT intent 未进 runAll（不导致本次超时）。

## 根因

探活地址手写，且从兄弟条目复制 YAML 后只改 `name` / `start_command`，漏改端口。监听 SSOT 与编排探针脱节。

## 解决方案

1. 把 fanout 的 ready/live URL 改回 `:18048`，同步 Prometheus file_sd。
2. 用 `taskEvents/run.sh start task_status_changed/2_fanout_work_panel_sse` 确认 `/api/health/` 与 `/ready` 均为 200。
3. 再点精准编译重启，使 runAll `LoadConfig` 重载 YAML（内存里的旧 18056 不会自动变）。

## 预防措施

1. 元规则：`.ai/01_project_constraints/52_runall_health_port_ssot.md`（约束索引第 47 条）；ADR-0012。
2. CI / pre-commit：`python3 db/scripts/ci/check_task_events_runall_health_ports.py`。
3. 新增 intent 顺序：先 YAML `port`+`groupId` → `intent_registry.go` → runAll health URL（抄同一数字）。
4. 禁止复制相邻 `health_check` 块后只改 `name`。
