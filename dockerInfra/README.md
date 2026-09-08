# dockerInfra — 本地开发基础设施

Redis、Kafka 与 MySQL 拆分为独立 Docker Compose 栈，位于 monorepo 根目录（与 `runAll/`、`task2app/` 平级），由 runAll 分别编排为 `docker-redis`、`docker-kafka` 与 `docker-mysql`。

## 目录

| 路径 | 内容 | 端口 |
|------|------|------|
| `redis/` | Redis 7 | `6379`（`port_config.json` → `dockerInfra.redisPort`） |
| `kafka/` | Zookeeper + Kafka + Kafka UI | `9093`、`18080`（`dockerInfra.kafkaBootstrapServers` / `kafkaUiPort`） |
| `mysql/` | MySQL 8 | `3306`（`port_config.json` → `dockerInfra.mysqlPort`） |
| `portainer/` | Portainer CE（远程 Docker 管理） | `9000`（`conf/docker-infra/config.yaml` → `portainerPort`） |

## 手工命令

```bash
bash dockerInfra/redis/run.sh start    # 或 managed / stop
bash dockerInfra/kafka/run.sh start
```

### Redis 与本机 systemd redis-server

Compose 默认映射宿主 `6379`。若本机已有 `redis-server`（常见：`systemctl status redis-server`）占用该端口，`compose up` 会报 `address already in use`。

行为：

1. **默认自动复用**：启动前发现宿主 `127.0.0.1:6379` 可 `PING` 且 Compose 未运行时，写入 `redis/.reuse_host_redis` 并跳过 Compose（业务仍连 `6379`）。
2. **强制 Compose**：`DOCKER_REDIS_REUSE_HOST=0 bash dockerInfra/redis/run.sh start`（需先停宿主 Redis，例如 `sudo systemctl stop redis-server`）。
3. **强制复用**：`DOCKER_REDIS_REUSE_HOST=1`。
4. **健康检查**：`health.sh` 优先验收 Compose 容器；仅在复用标记/`DOCKER_REDIS_REUSE_HOST=1` 时接受宿主 PING（避免「系统 Redis 在线却从未起过 Compose」的假健康）。

## runAll

```yaml
# runAll.yaml（working_dir 相对仓库根）
docker-redis: working_dir ., health_check.exec bash dockerInfra/redis/health.sh
docker-kafka: working_dir ., health_check 见 conf/runAll.yaml

# runAll/config.yaml（working_dir 相对 runAll/）
docker-redis: working_dir ../dockerInfra/redis
docker-kafka: working_dir ../dockerInfra/kafka
```

默认开发（`domainEvents.transport=redis`）只需启动 `docker-redis`。Kafka transport 或 AiProvider 镜像 hash 等场景需额外启动 `docker-kafka`。

## 迁移说明

原 `task2app/Saas_project/docker-compose.yml` 与 `run-infra.sh` 已移除，请改用本目录。

## License

本仓库以 MIT License 授权，见 [LICENSE](./LICENSE)。
