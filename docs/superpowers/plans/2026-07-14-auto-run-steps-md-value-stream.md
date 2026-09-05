# 价值流 — autoRunStep.md（精简）

## 增量切片

| 序 | 增量 | 验收 |
|----|------|------|
| 1 | 容器 GET + 镜像内 md 模板 | S1 |
| 2 | Go extract + 模型字段 + catalog/dev-catalog | S2–S4 |
| 3 | ImageMarket / CreateTask / TaskDetail UI | S5–S8 |

## 触及 YAML streams

- `installed-image-api`（字段）
- `create-task-auto-run-backend-start`（UI）
- 新增建议名：`auto-run-steps-doc`
