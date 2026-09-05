# runAll 远程 Docker 管理面板（Portainer）设计

> 日期：2026-06-02  
> 状态：**已批准**（auto-flow step 1）  
> 触发：runAll UI 缺少远程 Docker 管理入口；docker-redis TCP 端口不可点击

## 1. 目标

1. 在远程 CPU（`172.20.10.7`）部署 **Portainer CE**，提供完整 Docker 管理能力。
2. runAll UI（`:9999`）增加 **Portainer 入口**（顶部工具栏）。
3. 修复 **docker-redis 端口不可点击**：TCP 探活服务链接到 Portainer，展示 `host:port`。

## 2. 决策（用户确认）

| 项 | 决策 |
|----|------|
| 管理方案 | C — Portainer 独立 Web UI，runAll 只提供入口 |
| 端口 | `9000`（HTTP） |
| 失败策略 | `docker-portainer` 使用 `on_failure: skip`（可选，不阻塞 redis/kafka） |

## 3. 架构

```text
Mac runAll UI (:9999)
  ├─ infra 栏 → Portainer http://172.20.10.7:9000
  ├─ docker-redis → 172.20.10.7:6379 (链接 → Portainer)
  └─ docker-portainer → HTTP /api/status 探活

Remote CPU
  └─ dockerInfra/portainer/ → portainer-ce + docker.sock
```

## 4. Portainer 栈

- 目录：`dockerInfra/portainer/`
- 镜像：`portainer/portainer-ce:2.27.0`，`pull_policy: if_not_present`
- 端口：`9000:9000`（开发主用 HTTP）
- 卷：`/var/run/docker.sock`、`portainer_data`
- SSOT：`conf/docker-infra/config.yaml` → `portainerPort: 9000`

## 5. runAll 集成

### 5.1 新服务 `docker-portainer`

```yaml
- name: docker-portainer
  start_command: "bash scripts/runall-remote-docker.sh stack up portainer"
  stop_command: "bash scripts/runall-remote-docker.sh stack down portainer"
  launch_mode: detach
  health_check:
    url: "http://${INFRA_HOST}:9000/api/status"
  on_failure: skip
```

### 5.2 脚本扩展

`runall-remote-docker.sh`：`sync portainer`、`stack up/down portainer`、`sync all` 含 portainer。

### 5.3 UI

- `/api/observability` 增加 `portainer_url`
- 新增 **infra-bar**：`远程 Docker: [Portainer] · infra: {host}`
- Status API 增加 `management_url`（TCP 探活 + 已配置 Portainer 时）

### 5.4 docker-redis 链接修复

- 后端：TCP 服务 + Portainer URL → `management_url`
- 前端：展示 `host:port`（从 `tcp://` 解析），href 指向 Portainer

## 6. 领域概念

| 概念 | 说明 |
|------|------|
| dev-infrastructure | 远程 Docker 编排与可观测入口 |
| InfraManagementPanelURL | Portainer HTTP 入口 |
| TCPProbeEndpoint | `host:port` 展示与 management 链接 |

## 7. 价值流影响

- 扩展 `runall-docker-infra-split`
- 新增 `runall-portainer-management-ui` 步骤
- 字段：`docker-portainer.runtime.lifecycle_status`、`docker-redis.runtime.management_link`

## 8. 安全（开发环境）

- 仅 LAN 可达；首次访问设置 admin 密码
- Docker socket 挂载 = 完整主机权限

## 9. 非目标

- runAll 内嵌容器列表
- Portainer RBAC / 多租户
- CI 启动 Portainer
