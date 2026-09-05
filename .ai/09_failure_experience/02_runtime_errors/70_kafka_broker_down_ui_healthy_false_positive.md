# [运行时] Kafka broker 已退出但 Kafka UI 仍 healthy → saas-backend 503 / 下游依赖失败

## 基本信息

- 案例编号：FE-20260721-KAFKA-UI-FALSE-HEALTHY
- 录入日期：2026-07-21
- 最后更新：2026-07-21
- 关联服务：`docker-kafka`、`saas-backend`、domain-events consumers、runAll

## 失败现象

- `kafka-kafka-1` 为 `Exited (1)`，`kafka-kafka-ui-1` / zookeeper 仍 Up
- 宿主机无 `9092`/`9093` 监听；Kafka UI `:18080` 仍可达
- runAll 将 `docker-kafka` 标为 **healthy**（探活仅打 UI）
- `saas-backend` `/api/health/` → HTTP 503，`checks.kafka`：`Broker transport failure`
- 依赖 saas readiness 的服务（task-gateway、taskFE、多数 task-events-*）启动失败：`dependency "saas-backend" readiness ... unhealthy: HTTP 503`

## 失败环境

- 本机 Docker Compose：`dockerInfra/kafka/docker-compose.yml`（cp-kafka 7.4.0 + ZK）
- runAll：`conf/runAll.yaml` → `docker-kafka` health `http://${INFRA_HOST}:18080`
- Django broker：`conf/core/django/docker-infra.yaml` → `bootstrapServers: localhost:9093`

## 排查过程

1. `/api/status`：25 healthy / 25 failed，失败多挂 saas-backend readiness
2. `curl /api/health/`：migrations/db/redis ok，kafka 失败
3. `ss`：仅 18080，无 9092/9093
4. `docker ps -a`：`kafka-kafka-1 Exited`；日志：`KeeperException$NodeExistsException`（ZK broker 节点残留）

## 解决方案

1. `bash dockerInfra/kafka/run.sh start`（或 `docker compose up -d kafka`）拉起 broker
2. 确认 `docker inspect -f '{{.State.Health.Status}}' kafka-kafka-1` = healthy，且 `9093` 监听
3. 确认 `curl -sf http://127.0.0.1:8001/api/health/` → 200
4. `POST /api/start-all`（带 owning `session_id`）重拉失败服务

若 ZK `NodeExists` 反复复现：停栈后清 ZK 中残留 broker ephemeral 节点，或按运维约定重建 kafka/zookeeper volume（慎用，丢消息）。

## 预防措施

1. runAll `docker-kafka` 使用 `exec: bash dockerInfra/kafka/health.sh`（compose `kafka` running + 容器内 `kafka-topics`），**禁止**仅探 Kafka UI `:18080`（OPT-20260721-007 已落地；原误标 006 因撞号改 007）
2. `docker-kafka` healthy ≠ saas kafka check ok；启动下游前应确认 `/api/health/` 非 503
3. 经验索引见本文件
