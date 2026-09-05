# runAll 远程 Docker — 去 SSH 隧道 + 配置同步方案

> 日期：2026-06-02  
> 状态：**已实施**（2026-06-02）  
> 触发：GitLab 远程启动失败（`docker-desktop-helper.sh` 不存在）；隧道方案端口映射混乱；用户希望「同步配置 → 远程启动」

---

## 1. 问题陈述

### 1.1 当前日志根因（git-service）

```
run.sh: line 12: .../scripts/docker-desktop-helper.sh: No such file or directory
```

| 现象 | 根因 |
|------|------|
| 远程 `run.sh` 第 12 行直接 source helper | 远程仅有 **gitService 子目录** 的旧版 `run.sh`，未包含本机已有的 Linux 回退逻辑 |
| rsync/clone 不完整 | 当前 `ensure_subrepo` 只同步子仓库，**不含** monorepo `scripts/` |
| 依赖 Mac 专用脚本 | `docker-desktop-helper.sh` 仅适用于 Docker Desktop，远程 Linux 不应引用 |

### 1.2 隧道方案的结构性问题

| 问题 | 说明 |
|------|------|
| **双 IP 语义** | 应用连 `127.0.0.1:6379`，infra 实际在远程；本机 Homebrew Redis 可让 `ssh-tunnel` 误报 healthy |
| **端口映射表过长** | 6379/9093/18080/3000/3100/4317/4318/8012/2222 共 9 个转发，排查困难 |
| **启动链路过长** | remote up → tunnel start → local probe，任一步失败难以定位 |
| **部分 clone** | aimonitor/gitservice 需 bind mount，但远程工作区常不完整 |

### 1.3 用户新诉求（已确认方向）

1. **去除 SSH 隧道**
2. **本机集中配置 IP/端口**（单一 SSOT，避免 localhost vs 远程 IP 混用）
3. **「同步」**：将 docker-compose 等配置推到远程
4. **「启动/关闭/重启」**：SSH 到远程执行 compose，健康检查直连远程 IP

---

## 2. 目标与非目标

### 2.1 目标

1. Mac 上 **runAll UI + 应用服务**（platform/domain-events 等）继续本地运行
2. **Docker infra 栈**（redis/kafka/ai-monitor/git-service）仅在远程 CPU 运行
3. **无 SSH 隧道**；Mac 应用与 runAll 探活均使用 `remote_docker.host`（如 `172.20.10.7`）
4. **显式 Sync 清单**：按 stack 同步 compose + 最小 run 脚本，不依赖全量 monorepo clone
5. UI 保留现有 **启动/关闭/重启/全部启动**；新增 **「同步远端」** 按钮（组级 + 单服务）

### 2.2 非目标

- 不把 platform 应用整体迁到远程（下一阶段可选）
- 不做多远程主机 HA / 多开发者隔离 prod 级方案
- 不改变业务 value-stream 步骤语义
- CI 仍不通过 runAll 拉远程 infra（CI 自有 Docker profile）

---

## 3. 价值流影响

| 问题 | 结论 |
|------|------|
| 影响哪些 stream？ | **planned** 的 `runall-docker-infra-split`、`runall-global-start-stop-all`（探活 URL 从 localhost 改为 infraHost） |
| 新 stream？ | 可选新增 `runall-remote-docker-sync`（同步 + 远程启停验收） |
| 字段影响 | 无业务 DB 字段；新增运行时字段如 `docker-redis.runtime.remote_sync_revision`（验收用） |
| 测试影响 | `runAll/src/*_test.go`、Playwright infra 相关用例；业务 pytest 不变 |
| CI 影响 | 无 |

---

## 4. 领域概念清单（供 `/5-ddd`）

| 概念 | 说明 |
|------|------|
| 限界上下文 **dev-infrastructure** | 远程 Docker 编排、同步、启停 |
| 实体 **RemoteDockerHost** | SSH 目标 + 探活/连接用 IP（`cpu` / `172.20.10.7`） |
| 实体 **InfraStack** | redis / kafka / aimonitor / gitservice |
| 值对象 **InfraEndpoint** | `host:port` 或 `http://host:port/path`，来自 SSOT |
| 值对象 **StackSyncManifest** | 某 stack 需 rsync 的路径列表 + 排除规则 |
| 值对象 **SyncRevision** | 末次同步时间/hash，供 UI 展示「是否需要同步」 |
| 领域服务 **RemoteStackSyncService** | 执行 rsync、校验远程目录 |
| 领域服务 **RemoteStackExecutor** | SSH 远程 `compose up/down` |
| 领域服务 **InfraEndpointResolver** | 将 runAll.yaml 中 `${INFRA_HOST}` 解析为实际 IP |

---

## 5. 方案对比

### 方案 A — 保留隧道，仅加强同步（不推荐）

- 继续 `127.0.0.1` 探活 + tunnel
- 修复 sync 清单与 gitService run.sh
- **缺点**：不解决用户「IP/端口混乱」核心痛点

### 方案 B — 远程直连 + 配置同步（**推荐**）

```
┌──────────────── Mac ─────────────────┐
│ runAll UI (:9999)                      │
│ platform 应用 (:8001, :4000, …)        │
│         │                              │
│         │  TCP/HTTP 直连              │
└─────────┼──────────────────────────────┘
          ▼
┌──────────────── Remote CPU ────────────┐
│ 172.20.10.7:6379   redis               │
│ 172.20.10.7:9093   kafka               │
│ 172.20.10.7:3000   grafana/aimonitor   │
│ 172.20.0.7:8012    gitlab              │
└────────────────────────────────────────┘
```

- 去除 `ssh-tunnel` 服务及 `docker-tunnel-remote.sh` 在 lifecycle 中的调用
- infra 健康检查 URL/TCP 改为 `172.20.10.7:*`
- platform 服务 env 由 runAll 注入 `INFRA_HOST` 或从 `port_config` 读取

### 方案 C — 全栈远程（Mac 仅 UI）

- 所有服务 SSH 到远程启动
- **缺点**：改动面过大，与当前「Mac 跑 Python/Vue 热重载」工作流冲突

**决策：采用方案 B。**

---

## 6. 配置 SSOT

### 6.1 `runAll.yaml` 扩展

```yaml
remote_docker:
  ssh_host: cpu                    # SSH 别名（~/.ssh/config）
  host: 172.20.10.7                # 探活 + 应用连接 IP（LAN）
  workspace: ~/gitClone/ramDisk/ram-mount
  sync:
    auto_before_start: true        # 启动前若 revision 过期则自动 sync（可关）

groups:
  - name: infrastructure
    services:
      - name: docker-redis
        start_command: "bash scripts/runall-remote-docker.sh stack up redis"
        stop_command: "bash scripts/runall-remote-docker.sh stack down redis"
        health_check:
          tcp: "${INFRA_HOST}:6379"    # runAll 启动时解析
        # 删除 ssh-tunnel depends_on
```

### 6.2 应用连接（Mac → 远程 infra）

**原则**：Mac 上运行的服务不再使用 `127.0.0.1` 访问 redis/kafka/otel/gitlab。

| 消费者 | 现状 | 改后 |
|--------|------|------|
| saas-backend redis/kafka | `localhost:6379/9093` | `${INFRA_HOST}:6379/9093` |
| OTEL | `127.0.0.1:4317` | `${INFRA_HOST}:4317` |
| runAll observability 链接 | `127.0.0.1:3000` | `http://${INFRA_HOST}:3000` |
| git-oauth → gitlab | `127.0.0.1:8012` | `${INFRA_HOST}:8012` |

**实现路径（二选一，推荐 6.2.1）**：

#### 6.2.1 runAll 启动 platform 时注入 env（推荐）

```yaml
# saas-backend env 示例
env:
  REDIS_URL: "${INFRA_HOST}:6379"
  KAFKA_BOOTSTRAP: "${INFRA_HOST}:9093"
  OTEL_EXPORTER_OTLP_ENDPOINT: "${INFRA_HOST}:4317"
```

runAll Go：`resolveInfraHost(cfg)` → 替换 `${INFRA_HOST}`。

#### 6.2.2 `port_config.local.json`（gitignore）

开发者手动维护 `infraHost`；runAll 不注入。  
**缺点**：双 SSOT，易漂移。

---

## 7. Sync 设计

### 7.1 Stack 同步清单（manifest）

| Stack | 同步路径 | 说明 |
|-------|----------|------|
| **redis** | `dockerInfra/redis/` | compose + run.sh |
| **kafka** | `dockerInfra/kafka/` | compose + run.sh |
| **aimonitor** | `AiMonitor/`, `DaydaymoneyGrafana/dist/` | compose + run.sh；dist 可本地 build 后 sync |
| **gitservice** | `gitService/` | compose + run.sh + `gitlab_home/` 空目录结构 |
| **shared** | `scripts/docker-env.sh`, `scripts/runall-remote-docker.sh`, `scripts/remote-compose-helper.sh`（新） | 远程 Linux 专用 helper，**不含** docker-desktop-helper |

**不同步**：`task2app/`、`runAll/` 源码、Mac 应用代码。

### 7.2 Sync 命令与 API

```bash
# CLI
bash scripts/runall-remote-docker.sh sync redis
bash scripts/runall-remote-docker.sh sync all

# runAll API（新）
POST /api/remote-docker/sync
{ "stack": "redis" | "all", "session_id": "..." }
```

### 7.3 远程 run 脚本约束

新建 `scripts/remote-compose-helper.sh`（Linux only）：

- `compose_cmd()` — docker compose 探测
- `daemon_ready()` — docker info
- **禁止**引用 `docker-desktop-helper.sh`

各 stack 的 `run.sh` 在远程仅 source 此 helper（或通过 sync 写入远程 `scripts/`）。

### 7.4 Sync 状态

- 本地记录 `~/.cache/runall-remote-sync.json`：每 stack 的 `last_sync_at`、`source_hash`
- UI infrastructure 组显示：`已同步 2 分钟前` / `⚠ 源码已变更，建议同步`

---

## 8. 启停流程（去隧道）

### 8.1 `stack up redis`（新流程）

```
1. preflight（SSH + 远程 docker info）
2. [optional] sync redis（若 auto_before_start 或 manifest 过期）
3. ssh cpu "cd $WORKSPACE/dockerInfra/redis && docker compose up -d"
4. wait_remote_tcp $INFRA_HOST:6379
5. exit 0（runAll detach 探活继续）
```

**删除**：`tunnel start`、`wait_local_tcp 127.0.0.1`。

### 8.2 runAll.yaml infrastructure 组（定稿 4 服务）

| service | start | stop | health | on_failure |
|---------|-------|------|--------|------------|
| `docker-redis` | stack up redis | stack down redis | tcp `${INFRA_HOST}:6379` | exit |
| `docker-kafka` | stack up kafka | stack down kafka | url `http://${INFRA_HOST}:18080` | exit |
| `ai-monitor` | stack up aimonitor | stack down aimonitor | url `http://${INFRA_HOST}:3000/api/health` | skip |
| `git-service` | stack up gitservice | stack down gitservice | url `http://${INFRA_HOST}:8012/users/sign_in` | skip |

**删除 `ssh-tunnel` 服务。**

### 8.3 infra up / infra down

- `infra up` = sync all → 按 DAG 并行 stack up（无 tunnel）
- `infra down` = 反序 stack down（无 tunnel stop）

---

## 9. UI 变更

| 位置 | 变更 |
|------|------|
| infrastructure 组标题栏 | 新增 **「同步远端」** 按钮 → `POST /api/remote-docker/sync?stack=all` |
| 单服务行 | 可选 **「同步」** 小按钮（仅 infra 栈） |
| 服务 URL 列 | infra 服务显示 `172.20.10.7:6379` 而非 `127.0.0.1` |
| observability bar | Grafana/Loki 链接指向 `${INFRA_HOST}` |

---

## 10. 网络与安全前提

1. Mac 与 CPU 在同一 LAN（当前 `172.20.10.7` 可达）
2. 远程 Docker 绑定 `0.0.0.0` 或 LAN IP（compose 默认 publish 端口即可）
3. **防火墙**：CPU 需放行 6379/9093/18080/3000/8012 等来自 Mac 网段
4. 仅开发环境；不暴露公网

---

## 11. 风险与缓解

| 风险 | 缓解 |
|------|------|
| Mac 离线时无法访问远程 infra | 文档说明；runAll 探活失败明确提示「无法连接 ${INFRA_HOST}」 |
| 多开发者共用一个 CPU | 远程 compose project name 加开发者前缀（后续）；V1 单用户 |
| gitlab OAuth 回调 URL 仍写 localhost | git-oauth env 同步改为 `${INFRA_HOST}:8012`；port_config 文档更新 |
| 现有 port_config 大量 localhost | platform env 注入优先；逐步迁移 port_config 文档 |

---

## 12. 实施分期

### Phase 1 —  unblock git-service + 去隧道 infra（MVP）

- [ ] 实现 `sync` 子命令 + manifest
- [ ] 新增 `remote-compose-helper.sh`；修复远程 run.sh 依赖
- [ ] `stack up/down` 改为远程直连探活，删除 tunnel 步骤
- [ ] runAll.yaml：删除 ssh-tunnel；health 改用 `${INFRA_HOST}`
- [ ] runAll Go：`${INFRA_HOST}` 模板解析

### Phase 2 — platform 连接 + UI

- [ ] platform 服务 env 注入 INFRA_HOST 相关变量
- [ ] UI「同步远端」按钮 + sync 状态展示
- [ ] observability URL 改用 infra host

### Phase 3 — 清理

- [ ] 废弃 `docker-tunnel-remote.sh` 在 lifecycle 中的引用
- [ ] 更新 `2026-06-01-runall-remote-docker-only-design.md` 为 superseded
- [ ] Playwright / Go 集成测试更新

---

## 13. 待用户确认的问题

1. **`remote_docker.host` 固定为 `172.20.10.7` 还是可 per-developer 配置？**（建议：runAll.yaml 可配，默认 172.20.10.7）
2. **启动前是否默认 auto-sync？**（建议：`auto_before_start: true`，可 env 关闭）
3. **DaydaymoneyGrafana/dist 由本机 build 后 sync，还是远程 npm build？**（建议：本机 build + sync dist，减少远程 Node 依赖）

---

## 14. 与当前报错的关系

用户日志中的失败 **将在 Phase 1 通过 sync 解决**：

1. 「同步远端」将推送含 Linux 回退的 `gitService/run.sh` + `scripts/remote-compose-helper.sh`
2. 远程不再引用 `docker-desktop-helper.sh`
3. 去除隧道后，git-service 探活改为 `http://172.20.10.7:8012/...`，与真实运行位置一致
