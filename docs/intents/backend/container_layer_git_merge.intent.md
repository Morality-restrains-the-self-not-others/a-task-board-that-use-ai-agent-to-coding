# 容器层 Git 合并到目标分支



## 业务意图 → 事件对照

> 精修（2026-07-15）：事件名对齐仓库 MQ / domain events；同步写路径或非 MQ 副作用在例外理由标注「证据豁免」。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 容器层合并到目标分支成功 | ProjectUpdated | PROJECT_UPDATED | 容器 Git merge / Gateway | project_updated 消费者 | — |
## 变更记录
- 2026-07-12：首次。`POST /api/layers/{layer_id}/git/merge`

## 意图
1. 请求体：`target_branch` 必填；可选 `source_ref`（默认当前 HEAD）。
2. 工作区 dirty → 400。
3. 检出目标分支后 merge 源；冲突 → abort + 409。
4. 成功 → `{ ok: true, source_ref, target_branch, summary?, git_remote? }`。
5. SaaS：`container-layer-git-merge` 经 taskContainerGateway L0 转发。
