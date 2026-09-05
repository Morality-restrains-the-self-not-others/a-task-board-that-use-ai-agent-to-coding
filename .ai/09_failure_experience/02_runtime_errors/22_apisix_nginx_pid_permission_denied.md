# [运行时] APISIX `nginx.pid` Permission denied → task-gateway 重启循环

## 基本信息

- 案例编号：FE-20260717-APISIX-PID-EACCES
- 录入日期：2026-07-17
- 最后更新：2026-07-17
- 关联服务：taskGateway（`taskgateway-apisix-1`）、runAll `task-gateway`

## 失败现象

- `nginx: [emerg] open() "/usr/local/apisix/logs/nginx.pid" failed (13: Permission denied)`
- 容器 `Restarting`；`Get "http://127.0.0.1:18081/api/health/"` → `connection refused`
- runAll：`READINESS_TIMEOUT` / `service "task-gateway" failed to start`

## 根因

1. `docker-compose` 将宿主机 `taskGateway/logs` bind-mount 到 `/usr/local/apisix/logs`。
2. APISIX 镜像以 **uid 636（apisix）** 运行。
3. 宿主机残留 `nginx.pid` 属主为宿主机用户（如 uid 1000）、模式 `0644`。
4. nginx 启动时需 `open()`/`write` 该文件 → EACCES → 进程退出 → Docker 重启循环。
5. **P4 / 排除 `logs/` 的 rsync**：宿主机 `logs/` 不存在时，Docker 创建 bind-mount 目录为 **root:root 755**。uid 636 无法创建 `error.log` → `open() ".../error.log" failed (13: Permission denied)` → 容器 `Exited (1)`。host `chmod 777` 对 root 属主目录会失败。

## 修复

1. 启动前删除运行时文件：`logs/nginx.pid`、`logs/worker_events.sock`（APISIX 会重建）。
2. `taskGateway/run.sh` 的 `start`/`reload` 调用 `prepare_apisix_logs_for_start`（含 `docker run --user 0 ... chmod 777 /logs`）。
3. P4 `materialize-p4-root.sh` 预创建 `taskGateway/logs` 并 `chmod 777`。
4. 验证：`docker ps` 中 apisix 为 Up；`curl -sf http://127.0.0.1:18081/api/health/` 成功。

## 预防

- 勿在宿主机以普通用户创建/改写 `nginx.pid` 后留给容器覆盖。
- 日志截断脚本只 truncate `*.log`，不要 touch/创建 `nginx.pid`。
- 遇同类 emerg：先 `ls -lan taskGateway/logs/nginx.pid` 对照容器 uid（636）。
