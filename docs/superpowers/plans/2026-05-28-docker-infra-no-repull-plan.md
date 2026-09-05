# Implementation Plan: docker-infra 禁止重复拉镜像

> Design: `docs/superpowers/specs/2026-05-28-docker-infra-no-repull-design.md`

## Tasks

- [x] **Task 1:** 新增 `task2app/Saas_project/run-infra.sh`（start/managed/stop，ensure_images + up --pull never）
- [x] **Task 2:** 更新 `docker-compose.yml`（pull_policy，pin kafka-ui v0.7.2）
- [x] **Task 3:** 更新 `runAll.yaml` + `runAll/config.yaml` command → `bash run-infra.sh managed`
- [x] **Task 4:** `runner.go` — `isDetachComposeUp` 健康检查；`runInfraStopScript` on stop
- [x] **Task 5:** `runner_test.go` — detach compose 健康 + stop script 测试
- [x] **Task 6:** `cd runAll && go test ./...` 全绿
- [x] **Task 7:** 更新 `runAll.yaml.ai.md` 变更日志一行

## Verification

```bash
cd runAll && go test ./...
# 手工：镜像已齐时 bash task2app/Saas_project/run-infra.sh managed 两次，第二次无 Pulling
```
