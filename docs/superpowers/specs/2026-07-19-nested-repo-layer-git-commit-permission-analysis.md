# 权限分析：嵌套子仓层图提交 / Push / PR

日期：2026-07-19  
对应设计：`2026-07-19-nested-repo-layer-git-commit-design.md`

## 结论

无新公网 API、无新角色。仍走既有：

- 任务详情 → `taskContainerGateway` → 容器 `POST /api/layers/:id/git/commit|oauth-*-push`
- 鉴权：任务成员 / 容器 ACCESS_TOKEN（与现网一致）

## 变更点权限边界

| 能力 | 谁可调用 | 数据范围 |
|------|----------|----------|
| 嵌套发现 / dirty | 只读层图 | 当前任务层工作树 |
| 多仓 commit | 任务可写成员 | 仅该层内已发现 git 根 |
| `.nested-repo-heads` | 同上 | 仅父仓工作树内文件 |
| OAuth push/PR | 同上 + 用户 OAuth token 映射 | 仅 dirty/ahead 且 token 匹配的仓 |

## 风险

- 嵌套仓增多导致误推：已用 dirty/ahead 过滤缓解。
- 自动写入 `.nested-repo-heads`：仅当父仓存在 `.gitmodules`；属显式关联产物，可审阅。

## 审计

无需新增 IAM 策略；日志沿用现有 git push / oauth req log。
