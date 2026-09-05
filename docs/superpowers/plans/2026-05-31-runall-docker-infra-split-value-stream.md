# Value Stream: runAll dockerInfra 拆分（Redis / Kafka 独立启动）

> Derived from design: `docs/superpowers/specs/2026-05-31-runall-docker-infra-split-design.md`

## Value Summary

开发者可按需仅启动 Redis（默认开发路径），或单独启动 Kafka 栈，缩短 platform 依赖链等待并解除基础设施与 `Saas_project/` 的耦合。

## Related Value Streams

- **docker-infra-no-repull**（`2026-05-28-docker-infra-no-repull-value-stream.md`）：**修改** — pull/up 分离模式迁移至 `dockerInfra/*/run.sh`，原 `run-infra.sh` 删除。
- **runall-cascade-lifecycle**：**扩展** — 上游从单一 `docker-infra` 变为 `docker-redis` + optional `docker-kafka`；platform 链默认仅依赖 `docker-redis`。
- **message-queue-kafka-to-redis**：**对齐** — 默认 transport 已 Redis，本变更使 runAll 依赖与配置一致。

## End-to-End Flow

[开发者打开 runAll UI] → [启动 docker-redis] → [6379 TCP 就绪] → [task-sse / saas-backend 级联启动]

可选：[启动 docker-kafka] → [18080 Kafka UI 就绪] → [kafka transport 场景可用]

## Value Increments

### Increment 1: dockerInfra 目录 + Redis 栈（Thin Slice）
**Value to user:** 日常开发只需 Redis，platform 链可启动  
**Scope:** `dockerInfra/redis/` compose + run.sh；runAll `docker-redis` 服务；TCP 健康检查  
**Depends on:** nothing

### Increment 2: Kafka 栈独立 + runAll 服务
**Value to user:** Kafka 可单独启停，不拖 Redis  
**Scope:** `dockerInfra/kafka/`；runAll `docker-kafka`；HTTP 18080 探针  
**Depends on:** Increment 1（目录模式已建立）

### Increment 3: 依赖迁移 + 旧路径清理
**Value to user:** 无残留 `Saas_project/docker-compose.yml` 困惑  
**Scope:** `depends_on` 改 `docker-redis`；删除旧 compose/run-infra.sh；文档与 Prometheus  
**Depends on:** Increment 1–2

### Increment 4: runner 生命周期泛化 + 测试
**Value to user:** UI stop 正确 compose down；重启秒级 healthy  
**Scope:** `compose_lifecycle.go` 识别 `dockerInfra/*/run.sh`；Go 测试 AC5  
**Depends on:** Increment 1–2
