# Value Stream: docker-infra 禁止重复拉镜像

> Derived from design: `docs/superpowers/specs/2026-05-28-docker-infra-no-repull-design.md`

## Value Summary

开发者在 runAll UI 启动/重启 docker-infra 时，镜像已齐则秒级就绪，不再重复 Pulling，从而 unblock platform 依赖链。

## Related Value Streams

- **runall-stability-first**：同属 runAll 可靠性；本变更聚焦 infrastructure 组。
- **runall-cascade-lifecycle**：链式启动依赖 docker-infra 先 healthy；本变更缩短其上游等待。
- Greenfield 增量 — 无既有 docker-infra 专用 value stream。

## End-to-End Flow

[开发者点启动/重启 docker-infra] → [run-infra.sh 检测镜像] → [缺则 pull 一次 / 否则 skip] → [compose up --pull never] → [8080 健康] → [task-sse / saas-backend 可链式启动]

## Value Increments

### Increment 1: 脚本化 pull/up 分离（Thin Slice）
**Value to user:** 首次仍能自动拉镜像；再次启动日志无 Pulling  
**Scope:** `run-infra.sh` + compose pull_policy + runAll command  
**Depends on:** nothing

### Increment 2: runAll stop + detach 健康检查
**Value to user:** UI 关闭真正 down 栈；重启不再误判 failed  
**Scope:** `runner.go` stop hook + `up -d` 健康等待  
**Depends on:** Increment 1

### Increment 3: 验收与文档
**Value to user:** 可回归验证 AC1–AC4  
**Scope:** Go 测试 + `runAll.yaml.ai.md` 一行  
**Depends on:** Increment 2
