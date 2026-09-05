# 设计：自建 GitLab 与默认机器节点同专有网络提示

- **日期**: 2026-08-26
- **状态**: accepted（/goal 自动采用）
- **架构变更**: 否（无新服务、无新 API、无新表；只读既有 `GET /api/cloud/server-config-default/`，复用既有 create-vpc / create-vswitch）
- **Python 新 API 门禁**: not_applicable

## Context

租户设置页「GitLab」区块二（`data-testid="gitlab-self-hosted-connection"`）只登记自建 GitLab OAuth。若租户已在「工作空间管理 → 机器节点」保存**默认服务器启动配置**（默认机器节点），任务 VM 会落在该配置的专有网络（VPC）与交换机上。自建 GitLab 若在不同网络，任务节点只能走公网访问，占用 GitLab 流量配额。

用户期望：有默认机器节点时，在该区块给出专有网络与交换机的**创建提示**，引导任务机器与 GitLab 节点处于同一 VPC。

## Decision

1. **显隐**：`GET /api/cloud/server-config-default/tenant_id/{tenant}/` 返回至少一条默认配置 → 展示提示；空列表或未登录失败不展示成功态提示。
2. **文案**：说明任务机器节点与自建 GitLab 应共用同一专有网络/交换机，以便内网访问、避免公网流量。
3. **已有网络**：优先展示已带 `vpc_id` 的那条默认配置的 region / VPC ID / 交换机 ID，作为 GitLab 应加入的目标网络。
4. **创建动作**：复用既有 `CreateVpcModal` / `CreateVswitchModal`（与默认机器配置弹窗同一套云 API）。无 VPC 时禁用「创建交换机」。创建 VPC 成功后用响应 `vpc_id` 启用交换机创建（不必立刻写回默认配置）。
5. **导航**：真实 `<a href>` 链到 `/tenant/{id}/settings/task-panel/`（工作空间管理 → 机器节点），`Anti-Replay-OK: real navigation`。
6. **无默认机器**：不展示该提示（避免对未用云主机的租户制造噪音）。

## Alternatives Considered

| 方案 | 拒绝原因 |
|------|----------|
| 仅文字、无创建按钮 | 用户仍要绕到机器策略弹窗；「创建提示」可操作性不足 |
| 新后端 API 聚合网络建议 | 无新数据；既有 list default-config 已够 |
| 自动把 GitLab 实例放进 VPC | 自建 GitLab 不由平台编排，无法代建 |
| 做架构图 / 新事件 | 纯前端只读 + 复用已有写接口，无限界上下文变化 |

## Consequences

- 正面：配置自建 GitLab 时能看见与任务机器对齐网络的指引，降低公网流量误用。
- 负面：创建 VPC 后默认机器配置不会自动更新；文案须写明还需在「机器节点」写入默认配置，并自行把 GitLab 部署进该 VPC。
- 缓解：提示内同时给出目标 VPC/交换机 ID 与工作空间管理链接。
