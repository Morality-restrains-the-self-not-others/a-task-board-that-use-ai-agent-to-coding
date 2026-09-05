# [运行时] docker-redis 停服报 still reachable at :6379

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-18
- 最后修改：2026-07-18
- 维护者：Trae AI 团队

## 现象

runAll 关闭全部服务时，`docker-redis` 停服失败：

```text
docker-redis: service "docker-redis" still reachable at tcp://127.0.0.1:6379 after stop
```

`bash dockerInfra/redis/run.sh stop`（`compose down`）本身已成功，Compose 栈无运行中容器。

## 环境与上下文

- 本机存在 systemd `redis-server.service`，以用户 `redis` 监听 `127.0.0.1:6379`
- runAll 以普通用户运行；`lsof -t -iTCP:6379 -sTCP:LISTEN` 看不到该进程（无 root）
- `docker-redis` 为 `launch_mode: detach` + `stop_command`，无 tracked PID
- 健康检查为 `health_check.tcp: ${INFRA_HOST}:6379`

## 根因

1. `finalizeServiceStop` 在无 tracked 进程时走 `ensureServiceNotReachable`
2. `probeActivePortListeners` 依赖 `lsof`，对特权外来进程返回「无监听」
3. TCP 探活仍连上系统 Redis → 再尝试按端口杀进程（同样看不见 PID）→ 误报 still reachable
4. 连带效应：系统 Redis 在线时，`docker-redis` 可能从未真正起容器却显示 healthy

## 解决方案

`runAll/src/runner.go` → `ensureServiceNotReachable`：

- 探活仍成功时，先列出端口监听 PID
- 若 **无可视 PID**（外来特权进程），记录 warn 并 **视为托管停服完成**（`stop_command` 已执行）
- 仅当存在可视 PID 且终止后仍可达时，才返回 still reachable 错误

回归：`TestRunner_StopService_DetachStopIgnoresInvisibleForeignReachability`

## 预防

- 开发机若启用系统 `redis-server`，注意与 `docker-redis` 共用 6379：探活无法区分「Compose Redis」与「系统 Redis」
- detach + `stop_command` 的基础设施停服验收应以 **命令成功 + 托管资源释放** 为准，不可把「端口上仍有任意外来应答」一律当失败
- 勿为消报错去 `systemctl stop redis`（可能影响本机其它依赖系统 Redis 的服务）
