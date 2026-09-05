# 设计增量：闲置策略三项优化（E2E / 18044 recycle intent / 多容器共驻）

- 日期：2026-07-13
- 状态：已采纳（goal-mode 自动决策）
- 基于：`2026-07-13-workspace-machine-idle-policy-design.md`
- 架构：v19 扩展（同一 Plateau 补强，不升 v20；文档记增量）

## 成功标准

| ID | 标准 |
|----|------|
| E1 | Playwright：settings 打开机器节点模态并可保存策略 |
| E2 | Playwright：work-panel 可见 `data-alias=workspace-machine-summary` |
| R1 | taskEvents intent `workspace_machine_idle/1_recycle_idle_nodes` @18044，ticker 调 internal recycle |
| R2 | taskCloudService 内置 ticker 默认关闭（`IDLE_RECYCLE_TICK_SEC=0`），避免双扫 |
| M1 | reuse 改为绑定共驻：目标获得 instance，源任务保留 instance_id |
| M2 | 闲置/回收/汇总按 **instance_id** 去重；仅当该 instance 上无任何 busy 容器时方可回收 |

## 方案要点

### 多容器共驻

- `migrateIdleMachineToTask` → `bindSharedMachineToTask`：不再清空源 `instance_id`
- `findIdleMachineForReuse`：按 instance 聚合，仅当该 instance 全部任务 `server_url=''` 时可选
- `computeWorkspaceMachineCounts` / recycle：按 unique instance_id

### recycle intent

- 非 Kafka 消费；独立 binary：health :18044 + ticker POST `/api/internal/cloud/compute/recycle-idle-machines/`
- runAll 可启停

### E2E

- `WorkspaceSettings.machine-policy-modal.playwright.test.js`
- `WorkPanel.workspace-machine-idle-summary.playwright.test.js`
