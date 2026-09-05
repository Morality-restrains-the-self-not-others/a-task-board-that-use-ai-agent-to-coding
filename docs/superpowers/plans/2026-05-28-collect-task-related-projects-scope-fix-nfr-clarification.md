# NFR 澄清：collect_task_related_projects 范围修复

## 适用类别与等级（L2 Standard）

| 类别 | 等级 | 说明 |
|------|------|------|
| 正确性 | L3 | 查询结果必须与 `projects_taskproject` 语义一致 |
| 性能 | L2 | 单次 TaskProject JOIN，比全 workspace 扫描更轻 |
| 安全 | L2 | 不暴露未关联任务的仓库 URL 给 OAuth/PR 逻辑 |
| 可测试性 | L2 | 单元测试覆盖多项目 workspace + 单任务关联 |

## 质量场景

1. **刺激**：任务 T 仅关联项目 P1（GitLab），同 workspace 另有 P2（GitHub）。**响应**：`collect_task_related_projects(T)` 仅含 P1；PR 后续 `no_github_repo`。
2. **刺激**：任务未关联任何项目。**响应**：空列表；下游不报错。
3. **刺激**：TaskProject 指向其他 workspace 的项目。**响应**：被 `project__workspaces__id` 过滤排除。

## 领域模型影响

- `TaskProject` 为任务-项目关联的权威来源；`collect_task_related_projects` 为应用服务读模型，不应扩大至 workspace 全集。

## 不做

- 不引入缓存层
- 不改变 API 响应 schema
