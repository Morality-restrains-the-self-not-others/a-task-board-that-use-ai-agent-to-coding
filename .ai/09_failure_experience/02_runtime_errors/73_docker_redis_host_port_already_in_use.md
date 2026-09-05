# [运行时] docker-redis 启动失败：host port 6379 address already in use

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-22
- 最后修改：2026-07-22
- 维护者：Trae AI 团队

## 现象

runAll 启动 `docker-redis` 健康检查超时：

```text
docker-redis: [docker-redis] health check failed: health check timed out after 120s
(last error: exit status 1: compose redis container not running ...);
recent stderr: ... Error response from daemon: ... failed to bind host port 0.0.0.0:6379/tcp: address already in use
```

## 环境与上下文

- 本机 `redis-server.service`（systemd）监听 `127.0.0.1:6379`，`redis-cli ping` → `PONG`
- `dockerInfra/redis` Compose 同样映射宿主 `6379:6379`
- `health.sh` 只认 Compose 容器健康（或显式 host-reuse），故容器未起时健康检查失败并空转至 timeout

## 根因

宿主端口被系统 Redis 占用 → Compose 容器停留在 `Created` / 无法 `Starting` → `health.sh` 报 `compose redis container not running`。

## 解决方案

`dockerInfra/redis/run.sh` + `health.sh`：

1. 启动前检测宿主端口；若可 `PING` 且 Compose 未运行 → **自动 host-reuse**（写 `.reuse_host_redis`，跳过 Compose）
2. `health.sh` 在复用标记或 `DOCKER_REDIS_REUSE_HOST=1` 时接受宿主 PING
3. `DOCKER_REDIS_REUSE_HOST=0` 可强制要求 Compose（需先释放端口）
4. `stop` 清除复用标记并 `compose down`，**不**停止宿主 systemd Redis

手工释放端口（只要 Compose、不要系统 Redis 时）：

```bash
sudo systemctl stop redis-server
sudo systemctl disable redis-server
# 彻底移除包（推荐开发机只保留 Compose Redis）
sudo apt-get purge -y redis-server redis-tools
sudo apt-get autoremove -y
rm -f dockerInfra/redis/.reuse_host_redis
DOCKER_REDIS_REUSE_HOST=0 bash dockerInfra/redis/run.sh start
bash dockerInfra/redis/health.sh
```

## 预防

- 开发机同时装系统 Redis 与 docker-redis 时，优先依赖自动复用，或统一只用一端
- 勿把「TCP :6379 通」单独当作 Compose 已就绪（见 `48_docker_redis_still_reachable_after_stop.md`）
- 出现 `address already in use` 时先 `ss -ltnp | rg ':6379\b'`，再决定复用或停宿主服务
