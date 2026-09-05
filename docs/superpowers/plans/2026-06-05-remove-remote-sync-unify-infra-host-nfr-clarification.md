# NFR 澄清: 移除远程同步，统一 INFRA_HOST

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-06-05-remove-remote-sync-unify-infra-host-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-05-remove-remote-sync-unify-infra-host-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L1 | 本地 Docker Compose 启动不慢于远程 SSH+rsync（预期更快） |
| 可维护性 | L2 | runAll.yaml 无重复变量名；每服务独立 conf_app；无死代码残留 |
| 容错机制 | L1 | 移除 SSH/网络故障点，本地管理无远程依赖 |
| 可观测性 | L2 | 健康检查目标统一为 127.0.0.1，Prometheus 目标同步更新 |
| 数据一致性 | L1 | conf_app host 解析确定性、幂等（同一输入 → 同一输出） |
| 可用性 | L1 | 无专项 SLA，基础设施本地化自然降低外部依赖 |
| 安全性 | L0 | 不适用 — 无认证/授权/加密变更 |
| 可伸缩性 | L0 | 不适用 — 单机本地开发工具，无扩展需求 |
| 合规与隐私 | L0 | 不适用 — 无数据处理变更 |

## 逐增量 NFR 分析

### Increment 1: 配置拆分 — 基础设施各服务独立 conf_app

**可维护性 L2 — 标准**
- 每服务独立 `conf/<app>/config.yaml`，变更隔离
- 删除 `conf/infra/docker-infra/config.yaml` 后无遗留碎片
- 量化目标: 5 个新配置文件，所有 host 字段为 127.0.0.1

**数据一致性 L1 — 基础**
- conf_app 解析确定性：同一 YAML 输入 → 同一 host 输出
- 无并发写入场景

### Increment 2: 统一模板变量 — ${HOST} → ${INFRA_HOST}

**可维护性 L2 — 标准**
- runAll.yaml 中 `${HOST}` 出现次数: 35 → 0
- `${INFRA_HOST}` 成为唯一模板变量
- 量化目标: `grep '\${HOST}' conf/runAll.yaml` 返回空

**可观测性 L2 — 标准**
- 变量解析逻辑统一在 `resolveConfApps` 中，可单步调试
- 量化目标: 健康检查 URL 在 runAll 日志中输出完整解析后的值

### Increment 3: 删除远程同步 — 脚本 + Go 代码

**容错机制 L1 — 基础**
- 移除 SSH 连接超时、rsync 失败等故障模式
- 本地 `run.sh` 直接调用 Docker Compose，无网络依赖

**性能 L1 — 基础**
- 本地 Docker Compose 启动不经过 SSH+rsync 开销
- 预期: docker-redis 启动耗时 ≤ 当前远程启动耗时

### Increment 4: UI + API 清理

**可维护性 L2 — 标准**
- 移除 3 个废弃 API 端点（`/api/remote-docker/sync` 等）
- 移除 `syncable`、`portainer_url`、`management_url` 废弃字段
- 量化目标: `grep -r "remote.docker" runAll/src/ui.go` 返回空

### Increment 5: 辅助脚本 + 监控 + 测试清理
### Increment 6: 端到端验证

**可观测性 L2 — 标准**
- Prometheus 健康检查目标更新为 127.0.0.1
- 量化目标: `runall-health-targets.json` 中所有 targets 使用 127.0.0.1

## 质量场景

### QS-01: 配置解析确定性
| 要素 | 内容 |
|------|------|
| 类别 | 数据一致性 |
| 等级 | L1 |
| 刺激源 | runAll LoadConfig 启动 |
| 刺激 | 读取 conf/runAll.yaml + 各 conf/<app>/config.yaml |
| 制品 | `resolveConfApps()` 模板解析 |
| 环境 | 正常启动 |
| 响应 | 所有 `${INFRA_HOST}` 被替换为各自 conf_app 的 host 值 |
| 响应度量 | Go 单元测试: 给定已知 YAML 输入，断言每个服务的 URL/TCP/Env 中无残留 `${INFRA_HOST}` 或 `${HOST}` |

### QS-02: 基础设施本地启动
| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L1 |
| 刺激源 | 开发者在 runAll UI 点击启动 docker-redis |
| 刺激 | runAll 执行 `bash dockerInfra/redis/run.sh start` |
| 制品 | docker-redis 服务 |
| 环境 | 本地 Docker Desktop 已运行 |
| 响应 | Redis 容器启动，TCP 6379 就绪 |
| 响应度量 | 从点击到 health_check 通过 ≤ 120 秒（与当前 timeout 一致） |

### QS-03: 无死代码残留
| 要素 | 内容 |
|------|------|
| 类别 | 可维护性 |
| 等级 | L2 |
| 刺激源 | CI / 代码审查 |
| 刺激 | 全仓库搜索远程同步残留 |
| 制品 | 代码库 |
| 环境 | 编译 + grep 检查 |
| 响应 | 无 `runall-remote-docker.sh` 引用，无 `RemoteDocker` 类型引用 |
| 响应度量 | `grep -r "runall-remote-docker\|RemoteDocker\|remote_docker\.host\|SyncRemoteDocker\|SyncConfReplicaToRemote\|SyncServiceToRemote\|PortainerURL"` 仅在文档中匹配 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 可维护性 L2 — 每服务独立配置 | `ManagedService` 不再区分 local/remote，统一为本地服务 | 移除 `RemoteDocker` 值对象，`Service` 实体直接持有 `conf_app` 路径 |
| 可维护性 L2 — 变量统一 | 模板解析从两个独立机制合并为单一 `TemplateResolver` | `resolveConfApps` 成为唯一的变量解析域服务 |
| 容错 L1 — 移除远程故障点 | 服务生命周期不再有 `RemoteSync` 阶段 | 删除 `RemoteSyncCompleted` 领域事件，DAG 中移除 sync 节点 |

## 权衡与边界

### 取舍
- 选择移除 Portainer 远程面板（牺牲远程容器可视化）以换取架构简化（删除 remote_docker 全部代码）
- 统一 `${INFRA_HOST}` 后，无 conf_app 的服务使用兜底 host（首个 conf_app 的 host），限制了多主机异构部署

### 明确不做什么
- 不在本变更中实现多主机支持（各服务 host 可不同）—— 当前所有服务同主机（127.0.0.1）
- 不删除 `scripts/conf-sync-all.sh` / `conf-sync.py`（本地配置生成继续使用）
- 不删除 `scripts/runall-local-promtail.sh`（日志推送保留）
- 不修改 `scripts/docker-env.sh` 的 SSH 隧道/端口定义（其他工具可能使用）

### 升级触发条件
- 当需要多主机部署（不同服务在不同 IP）时 → conf_app 按服务解析 host 已天然支持，仅需改各 config.yaml 的 host 值
- 当需要恢复远程管理时 → 本变更是删除性重构，恢复需 revert + 重新设计

## 跳过声明
- **安全性**: 跳过。无认证/授权/加密变更。L0。
- **可伸缩性**: 跳过。单机本地开发工具，无水平扩展需求。L0。
- **合规与隐私**: 跳过。无数据处理/存储变更。L0。
- **可用性**: 跳过（L1 基础）。本地开发工具，无 SLA 要求。本地化自然降低外部依赖。
