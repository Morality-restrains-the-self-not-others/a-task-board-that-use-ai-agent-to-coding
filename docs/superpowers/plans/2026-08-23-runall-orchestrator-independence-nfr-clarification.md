# NFR 澄清：runAll 编排器独立于托管服务

- **Date:** 2026-08-23
- **Value stream:** `2026-08-23-runall-orchestrator-independence-value-stream.md`

## 适用类别与等级

| 类别 | 等级 | 说明 |
|------|------|------|
| 可用性 | L3 | 编排器退出不得使已启动业务面不可用 |
| 容错 | L3 | SIGTERM 与 SIGKILL 两条失败模式都要存活 |
| 可观测性 | L2 | 退出 trigger + keep/stop 结构化日志 |
| 安全 | L2 | 无新公网面；env 拆栈仅本机 |
| 可伸缩性 | L0 | 无新分片路径，见下表 |
| 数据一致性 | L0 | 无租户写路径 |

## 路径分片键强制审视

| 路径 | 分片 ID | 可伸缩性 | 判定 |
|------|---------|----------|------|
| 本机 SIGTERM → runAll | 无 | L0 | 单主机监督器，非租户流量；升级触发：多机编排器集群 |
| POST `/api/shutdown-self` | 无 | L0 | loopback 运维，单实例 9999 |
| POST `/api/stop-all` | 无（本机会话） | L0 | 既有运维口，本增量不改路由 |
| 托管服务健康 URL | 各服务既有 | n/a | 本增量不改 |

全部路径 L0：理由为单机进程监督，无租户级水平扩展需求。升级触发：runAll 变为多副本选主。

## 幂等性强制审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|------------|--------------|--------|----------|
| 编排器退出 keep | 无（不发停进程信号） | 多次 SIGTERM | n/a | L0 | 第二次仍 keep |
| adoptListeningServices | StatusStore 回填 | 重复启动 runAll | 端口+PID | 端口监听集合 | 重复 adopt 覆盖同一快照 |
| stop-all（未改） | 停进程 | 双击/重试 | 本机一次停栈 | 既有 runID | 保持 |
| Setsid 启动 | 建会话 | 每次 start | 单次 start 意图 | 服务名+ownership | 不在本增量改 |

`company_id`/`user_id` 未用作键。资金路径不涉及。

## 质量场景

1. **刺激：** 对 UI 模式 runAll 发 SIGTERM。**响应：** 30s 内业务健康 URL 仍 200；runAll 日志含 keep。
2. **刺激：** SIGKILL runAll。**响应：** 托管 PID 仍存在且 session≠已死 pid。
3. **刺激：** 空窗 `./run.sh`。**响应：** 不 SIGKILL 托管端口；UI 显示 adopted。

## 领域模型影响

- 新增值对象/策略：`OrchestratorExitTrigger` + `KeepManagedServicesOnOrchestratorExit`。
- 不新增聚合；ManagedService 生命周期仍由显式 stop 命令改变。
