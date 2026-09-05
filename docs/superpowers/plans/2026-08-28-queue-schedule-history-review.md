# 自动调度安排 · 调度历史 — Review

- 日期：2026-08-28
- 范围：本次 diff（queue-schedule 调度历史）

## 五轴

| 轴 | 结论 |
|----|------|
| Correctness | 历史仅状态变化写入；窗口翻转去重；查询带 workspace+tenant |
| Readability | 卡片抽出；store 独立文件 |
| Architecture | TTS 查询模型，无新服务；网关已有 `queue-schedule/*` |
| Security | 同 GET 快照 `hasWorkspaceAccess`；JOIN 不跨 workspace |
| Performance | limit≤50；recent_history 8；无轮询；月分区 |

## Log audit

append 失败 `schedule_history_append_failed`；无密钥。

## Intent → Event

WindowEntered/Exited 新事件；其余沿用既有。查询无事件。
