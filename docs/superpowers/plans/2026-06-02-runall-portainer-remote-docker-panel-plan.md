# Implementation Plan: runAll Portainer 远程 Docker 管理面板

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 部署远程 Portainer 栈，runAll UI 提供入口链接，修复 docker-redis TCP 端口不可点击。

**Architecture:** `dockerInfra/portainer` compose 栈经 SSH sync/up；runAll Config 推导 PortainerURL；Status API 为 TCP 服务注入 management_url；status.html infra-bar + endpoint 链接。

**Tech Stack:** Go runAll、bash rsync/SSH、Portainer CE、embedded status.html

---

### Task 1: domain — TCP 展示与 Portainer URL

**Files:**
- Create: `runAll/src/domain/infra_management_panel_url_value_object.go`
- Create: `runAll/src/domain/infra_management_panel_url_value_object_test.go`
- Create: `runAll/src/domain/tcp_probe_endpoint_value_object.go`
- Create: `runAll/src/domain/tcp_probe_endpoint_value_object_test.go`

**Steps:** TDD — URL 校验、TCP `tcp://host:port` → `host:port` 解析

### Task 2: Config PortainerURL + management_url payload

**Files:**
- Modify: `runAll/src/remote_docker.go` — PortainerPort、PortainerURL()
- Modify: `runAll/src/ui.go` — serviceStatusPayload.ManagementURL、observability portainer_url
- Test: `runAll/src/remote_docker_test.go`, `runAll/src/ui_test.go`

### Task 3: dockerInfra/portainer 栈

**Files:**
- Create: `dockerInfra/portainer/docker-compose.yml`
- Create: `dockerInfra/portainer/run.sh`
- Modify: `conf/docker-infra/config.yaml` — portainerPort
- Modify: `dockerInfra/README.md`

### Task 4: runall-remote-docker.sh + runAll.yaml

**Files:**
- Modify: `scripts/runall-remote-docker.sh` — portainer sync/stack
- Modify: `runAll.yaml` — docker-portainer 服务
- Modify: `runAll/src/remote_docker.go` — sync path override

### Task 5: status.html UI

**Files:**
- Modify: `runAll/src/status.html` — infra-bar、endpointLink、management_url
- Test: `runAll/src/ui_test.go` — HTML snippets

### Task 6: value-stream.yaml 步骤

**Files:**
- Modify: `value-stream.yaml` — runall-portainer-management-ui step

### Task 7: 全量测试 + build

```bash
cd runAll && go test ./...
```
