# [运行时] clone-run APISIX `config.yaml` Permission denied → task-gateway READINESS_TIMEOUT

## 基本信息

- 案例编号：RT-20260901-0124
- 录入日期：2026-09-01
- 最后更新：2026-09-01
- 关联服务：taskGateway（`taskgateway-apisix-1`）、runAll `task-gateway`
- 关联意图：`docs/intents/platform/runall_clone_run_task_gateway_apisix_config_perm.intent.md`

## 失败现象

- 新目录 `.daydaymoney-deploy-seed` / `~/bin/daydaymoney-deploy` 部署后，http://192.168.1.10:9999/ 「全部重启」：
  `task-gateway: service "task-gateway" failed to start [READINESS_TIMEOUT]`
- `Get "http://127.0.0.1:18081/api/health/": dial tcp 127.0.0.1:18081: connect: connection refused`
- compose 日志：Network/Container Created + Started，随后 `WARNING: APISIX not ready after 180s`
- 容器内：`/docker-entrypoint.sh: line 35: /usr/local/apisix/conf/config.yaml: Permission denied`，随后 `Exited (1)`

## 根因

1. `APISIX_STAND_ALONE=true` 时 apache/apisix:3.11.0-debian 入口以 **uid 636** 执行：
   `echo "$(sed … config.yaml)" > /usr/local/apisix/conf/config.yaml`
2. bind-mount 的宿主机 `taskGateway/apisix/config.yaml` 经 git/rsync 为 **0644、属主 1000**。
3. 重定向需要写权限 → EACCES → nginx 从未监听 9080 → 宿主机 18081 connection refused。
4. `/tmp/ram-work` 上该文件常被改成 `0666`（tmpfs 工作副本），故源码仓重启看起来正常，clone-run 必现。

对照：`22_apisix_nginx_pid_permission_denied.md` 是 **logs/** 不可写；本例是 **config.yaml** 不可写。

## 修复

1. `taskGateway/run.sh` `prepare_apisix_logs_for_start`：对 `apisix/config.yaml` `chmod a+rw`（host + docker `--user 0`）。不提交 0666 到 git。
2. compose up 后若容器未 running，立即 dump logs 并把 ready 等待收成 1s，避免空等 180s。
3. 测例：`taskGateway/scripts/test_run_sh_apisix_ready_wait.py`

## 验收

```bash
python3 taskGateway/scripts/test_run_sh_apisix_ready_wait.py
# clone-run 树：
bash taskGateway/run.sh prepare-host-mounts
curl -sf --max-time 3 http://127.0.0.1:18081/api/health/
```

## 预防

- 禁止假定 ram-work tmpfs 上的 0666 会随 rsync 到 daydaymoney-deploy。
- 容器 `Started` 但 health connection refused：先 `docker logs` entrypoint，再查 bind-mount 文件对 uid 636 是否可写。
