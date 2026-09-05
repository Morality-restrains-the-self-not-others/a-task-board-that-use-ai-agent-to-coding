# Review: 项目详情子 Git 仓库列表

日期：2026-07-15  
对照：`docs/superpowers/plans/2026-07-15-project-nested-git-repos-plan.md`

## 结论

**通过（可 Ship）** — 无阻断级问题。

## 检查项

| 项 | 结果 |
|----|------|
| T1–T2 解析单测 | ✅ PASS |
| T3 handler + mock GitLab | ✅ PASS |
| OpenAPI `nested-git-repos` | ✅ 已注册 |
| api_route_ownership | ✅ go owner |
| 前端 composable 单测 | ✅ 4 PASS |
| 线上 API（浏览器会话） | ✅ 34 条子仓 |
| 线上 UI `data-testid` | ✅ 34 行含 task2app |
| 跨租户 404 | ✅ |
| 无仓 400 | ✅ |
| Log：nested-git-repos 打点 | ✅ |
| Intent→Event | ✅ 纯查询书面例外 |

## 非阻断建议

1. 多父仓项目时 MVP 仅发现首个关联仓；后续可按行展开。
2. `gitHTTPClient` 受环境 `HTTP_PROXY` 影响时会导致与分支预览相同的授权失败（运维注意 NO_PROXY）。
3. 可选：一键将子仓加入 `project_repos`（明确非目标）。

## Log Audit

- handler 记录 project/user/parent；无 token 明文。
