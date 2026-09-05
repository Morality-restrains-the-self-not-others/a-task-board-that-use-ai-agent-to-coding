# Value Stream: runAll Portainer 远程 Docker 管理面板

> Derived from design: `docs/superpowers/specs/2026-06-02-runall-portainer-remote-docker-panel-design.md`

## Value Summary

开发者从 runAll UI 一键打开 Portainer，查看并管理远程 CPU 上全部 Docker 容器；TCP 探活的 infra 服务（如 docker-redis）端口可点击跳转。

## Related Value Streams

- **runall-docker-infra-split**（`2026-05-31-runall-docker-infra-split-value-stream.md`）：**扩展** — 新增 portainer 栈与 UI 入口。
- **runall-remote-docker-no-tunnel-sync**（`2026-06-02-runall-remote-docker-no-tunnel-sync-design.md`）：**依赖** — 远程 sync/stack 机制已就绪。

## End-to-End Flow

[开发者打开 runAll UI] → [启动 docker-portainer 或点击 Portainer 链接] → [Portainer 展示远程容器列表] → [点击 docker-redis 172.20.10.7:6379 → Portainer]

## Value Increments

### Increment 1: Portainer 栈 + 远程启停（Thin Slice）
**Value to user:** 远程 CPU 可运行 Portainer，runAll 可 sync/up/down  
**Scope:** `dockerInfra/portainer/`、`runall-remote-docker.sh`、`runAll.yaml` docker-portainer 服务  
**Depends on:** remote-docker sync 基础设施

### Increment 2: UI 入口 + API
**Value to user:** runAll 顶部可见 Portainer 链接  
**Scope:** `/api/observability` portainer_url、infra-bar  
**Depends on:** Increment 1

### Increment 3: docker-redis TCP 链接修复
**Value to user:** docker-redis 端口可点击，跳转 Portainer  
**Scope:** `management_url` API、`status.html` portLink 扩展、domain 层 TCP 展示  
**Depends on:** Increment 2
