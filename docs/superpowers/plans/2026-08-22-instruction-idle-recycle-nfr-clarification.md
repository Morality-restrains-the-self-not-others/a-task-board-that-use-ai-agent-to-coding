# NFR 澄清 — 指令闲置回收

- **日期:** 2026-08-22
- **价值流:** `docs/superpowers/plans/2026-08-22-instruction-idle-recycle-value-stream.md`
- **默认等级:** L2；云资源释放 **L3**（幂等/误拆）

## 路径分片键审视

| 路径 | 分片键 | 说明 |
|------|--------|------|
| POST `…/tenant/{tid}/workspace/{wid}/task/{taskId}/comment/{cid}/cloud/server-container-token/task-detail/` | tenant + workspace + task + comment | **合适**：评论级 CSC 与 token 同粒度 |
| GET `/api/internal/cloud/workspace-machine-policy/?company_id=&workspace_id=` | company_id + workspace_id | **合适**：policy 聚合根即此复合键 |
| POST heartbeat / request-machine-release | 同上 comment 路径 | **合适** |
| POST `/api/internal/cloud/compute/recycle-idle-machines/` | 无路径 ID（全库扫描） | **L2**：沿用现 timer；按 workspace policy 分组。升级触发：CSC 年增量 >1000 万再按 company 分片扫描 |
| Kafka `CONTAINER_INSTRUCTION_IDLE_MARKED/CLEARED` | 载荷 `config_id` / comment CSC id | **合适**；禁止用 company_id 作消费键 |
| Kafka `CLOUD_SERVER_STOPPED` | 已有 `stop_request_id` | 合适，不改 |

## 幂等性审视

| 路径 | 副作用 | 判定 |
|------|--------|------|
| GET internal policy / POST task-detail | 无（STS 签发除外） | 查询 **L0**。若签发 STS：重复拉 task-detail 刷新同一 Role 会话，TTL 覆盖；键=`csc_id`+时间窗 |
| heartbeat `instruction_idle=true` | 写列 + 事件 | **L2**：同 CSC 重复 true 只刷新时间或保持；事件可重复，消费无自动 handler |
| heartbeat `false` / 新 job | 清空列 + Cleared | 键=`config_id` |
| request-machine-release | 发 Stopped + 清 CSC | **L3**：沿用 `already_released` + `stop_request_id` |
| recycle timer | 同上 | **L3**：instruction_idle_since 到期才调 release；已 terminal_released 跳过 |
| L3 DeleteInstance | 云资源 | **L3**：仅交付成功且 L1 失败；Resource 锁 instance_id |

资金/云资源释放路径 ≥ L3。无前端新写按钮（设置页已有）。

## 类别定级

| 类别 | 级别 | 说明 |
|------|------|------|
| 可伸缩性 | L2 | 路径已带租户/工作空间；timer 全表扫描维持现状 |
| 数据一致性 | L3 | 闲置与释放以 CSC 列为真源；交付失败无列 |
| 安全 | L3 | STS 不进评论；session policy 单实例；internal secret |
| 可用性 | L2 | L1 失败靠 L2；执行中不因短心跳按 N 拆 |
| 性能 | L2 | policy GET 单行；recycle 沿用现扫描 |
| 可观测性 | L2 | 事件 + `event=instruction_idle_*` 日志；禁 STS 明文 |
| 容错 | L3 | 交付失败不拆；L2 不依赖容器存活 |

## 质量场景

1. 刺激：policy=30，拉 task-detail。响应：minutes=30。
2. 刺激：交付失败。响应：无 idle 列、无 release。
3. 刺激：交付成功后心跳 idle=true。响应：列非空 + Marked。
4. 刺激：新指令。响应：旧 job interrupted，列清空。
5. 刺激：到期且 server_url 仍在、容器已死。响应：timer 仍 Stopped。
6. 刺激：无 Role。响应：无 machine_release_sts。

## 领域模型影响

- 聚合根仍是评论级 CSC；新增 `InstructionIdleSince`
- Policy 仍是 workspace 配置根；Credential 只读端口
- 释放一致性：sole busy + CLOUD_SERVER_STOPPED 幂等键不变
- 禁止把 STS 放进评论聚合
