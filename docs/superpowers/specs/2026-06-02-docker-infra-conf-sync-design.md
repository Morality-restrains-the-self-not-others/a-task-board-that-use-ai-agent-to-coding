# docker-infra 配置 SSOT 与 infra 碎片同步设计

> 日期：2026-06-02  
> 状态：**已批准**（auto-flow step 1）  
> 关联：`2026-06-02-runall-remote-docker-no-tunnel-sync-design.md`

## 1. 目标

1. **`conf/docker-infra/config.yaml`** 为 Redis/Kafka endpoint 唯一权威源（远程 dev 机 `172.20.10.7`）
2. `domain-events`、`task-sse`、`django` 通过 **GENERATED `docker-infra.yaml` 碎片** 引用，禁止手写 infra host
3. Mac 应用 HTTP bind/API 与 infra endpoint **分层**（应用地址可 `config.local.yaml` 覆盖）

## 2. 地址分层

| 角色 | 地址 | 配置位置 |
|------|------|----------|
| 远程 Docker infra | `172.20.10.7` | `conf/docker-infra/config.yaml` → sync 碎片 |
| Mac 应用 HTTP | 本机 LAN | `conf/<app>/config.local.yaml`（gitignore） |
| taskEvents intent 端口 | Mac 本机 | `conf/domain-events/<event>/config.yaml` |

## 3. Sync 拓扑

```text
conf/docker-infra/config.yaml
  ├─► conf/domain-events/docker-infra.yaml
  ├─► conf/task-sse/docker-infra.yaml
  └─► conf/core/django/docker-infra.yaml
```

运行时 merge：`config.yaml` + `config.local.yaml` + `docker-infra.yaml`（GENERATED）

## 4. 非目标

- 不合并 `runAll.yaml` remote_docker 与 conf（后续可对齐）
- 不在 `config.yaml` 提交开发者本机 LAN IP
