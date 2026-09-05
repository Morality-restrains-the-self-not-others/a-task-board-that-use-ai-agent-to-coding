# Implementation Plan: runAll dockerInfra 拆分

> Design: `docs/superpowers/specs/2026-05-31-runall-docker-infra-split-design.md`  
> Value stream: `docs/superpowers/plans/2026-05-31-runall-docker-infra-split-value-stream.md`

## Task 1: dockerInfra/redis 栈

- [ ] **Step 1:** 写失败测试 — `runAll/src/compose_lifecycle_test.go` 识别 `bash run.sh managed` + `dockerInfra/redis`
- [ ] **Step 2:** 创建 `dockerInfra/redis/docker-compose.yml`
- [ ] **Step 3:** 创建 `dockerInfra/redis/run.sh`（从 run-infra.sh 精简）
- [ ] **Step 4:** `chmod +x dockerInfra/redis/run.sh`

## Task 2: dockerInfra/kafka 栈

- [ ] **Step 1:** 创建 `dockerInfra/kafka/docker-compose.yml`（迁移 zookeeper/kafka/kafka-ui）
- [ ] **Step 2:** 创建 `dockerInfra/kafka/run.sh`
- [ ] **Step 3:** `chmod +x dockerInfra/kafka/run.sh`

## Task 3: runAll TCP 健康检查

- [ ] **Step 1:** `config_test.go` — TCP-only health_check 解析与校验
- [ ] **Step 2:** `HealthCheck` 增加 `TCP` 字段；validate url|tcp 至少其一
- [ ] **Step 3:** `health.go` — `checkProbe` / `waitHealthy` 支持 TCP
- [ ] **Step 4:** `runner.go` — 监控/stop 使用 probe target
- [ ] **Step 5:** `go test ./...` 绿

## Task 4: compose_lifecycle 泛化

- [ ] **Step 1:** 更新 `compose_lifecycle.go` 识别 `dockerInfra` 下 `run.sh`
- [ ] **Step 2:** 更新 `compose_lifecycle_test.go`

## Task 5: runAll 配置与依赖

- [ ] **Step 1:** 更新 `runAll.yaml` — docker-redis + docker-kafka，depends_on 改 docker-redis
- [ ] **Step 2:** 同步 `runAll/config.yaml`

## Task 6: 清理与文档

- [ ] **Step 1:** 删除 `task2app/Saas_project/docker-compose.yml`、`run-infra.sh`
- [ ] **Step 2:** 更新 `task2app/run.sh` 提示文案
- [ ] **Step 3:** 创建 `dockerInfra/README.md`
- [ ] **Step 4:** 更新 `runAll.yaml.ai.md`、`AiMonitor/prometheus/file_sd/runall-health-targets.json`
- [ ] **Step 5:** 更新 `value-stream.yaml` runall-global-start-stop-all 描述

## Task 7: 验收

- [ ] **Step 1:** `cd runAll && go test ./...`
- [ ] **Step 2:** 手工 AC1 — 仅 redis 时 platform 链可启（文档记录）
