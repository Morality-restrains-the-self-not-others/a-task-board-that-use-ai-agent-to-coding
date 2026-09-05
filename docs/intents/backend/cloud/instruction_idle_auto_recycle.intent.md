# 意图：指令完成后按工作空间闲置策略回收机器

## 背景与目标

现网 `idle_recycle_minutes` 只在容器卸载后生效。任务详情（容器 `task-detail`）不含该字段。需要：容器按策略在**引导完成（无指令）或指令交付成功后**倒计时；到期无新指令则释放本评论机器；新指令中断旧指令。仅「交付成功」会漏掉克隆完成后从未下发指令的机器（心跳故意不带 `instruction_idle` 键，服务端不改列）。

## 范围与边界

- 范围内：task-detail 返回 `idle_recycle_minutes`；CSC `instruction_idle_since`；timer 扩展；onlineServiceJS 倒计时/抢占/`request-machine-release`；可选 STS（不进评论）。
- 范围外：跨任务闲置复用；跨评论抢占；设置页 UI；指令执行中短心跳按 N 拆机；Python 新接口。

## 约束与风险

- 交付未成功禁止释放（含 STS）。
- STS 禁止出现在评论正文 / context_pack。
- 表 owner：policy 与 CSC 仅 taskCloudService；Credential 只 HTTP internal 读。
- 业务 HTTP 进程不新增 ticker；L2 仍走 `workspace_machine_idle` timer。

## 验收标准

1. task-detail 含与 workspace policy 相同的 `idle_recycle_minutes`。
2. 引导完成或交付成功后 N 分钟无新指令 → 机器释放（L1 或 L2）。
3. 交付失败 → 不释放、不启动倒计时（引导完成仍可进入闲置）。
4. 新指令 → 中断同容器 running/pending job，取消倒计时。
5. `minutes=0` → 不因指令闲置释放。
6. 从未下发指令、但 `userdata_run_verified` 已满 N 分钟且无 running/pending job → L2 释放（旧镜像未上报 `instruction_idle` 的兜底）。
7. `BOOTSTRAP_COMPLETE` → 服务端写入 `instruction_idle_since`（列为空时）；容器同时启动 L1 倒计时并心跳 `instruction_idle=true`。
8. `userdata_run_verified` 为空时，L2 用 `created_at` 作闲置锚点；若按 UTC 解析该 DATETIME 落在未来，则按 Asia/Shanghai 墙钟再转 UTC（CSC `created_at` 常为会话 CST，心跳为 UTC）。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 指令交付成功或引导完成进入闲置 | CONTAINER_INSTRUCTION_IDLE_MARKED | Kafka `container-instruction-idle-marked` | taskCloudService 写 `instruction_idle_since`（仅列为空时；心跳或 BOOTSTRAP_COMPLETE） | 审计；L2 读 DB | — |
| 新指令取消闲置 | CONTAINER_INSTRUCTION_IDLE_CLEARED | Kafka `container-instruction-idle-cleared` | 清空 `instruction_idle_since` | 审计 | — |
| 闲置到期释放机器 | CloudServerStopped | Kafka `cloud-server-stopped`（已有） | request-machine-release / recycle timer / STS 成功后的 clear | 删实例 + clear-after-stop | — |
| 容器拉取 task-detail | — | — | Credential | — | 纯查询 |
| 读取 workspace policy | — | — | Cloud internal GET | Credential 组装 | 纯查询 |

## 实施计划

见 `docs/superpowers/specs/2026-08-22-instruction-idle-recycle-design.md`。
