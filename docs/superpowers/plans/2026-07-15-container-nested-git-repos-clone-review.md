# Review: 容器 task-detail 下发最新子仓 + 并发克隆

日期：2026-07-15  
对照：`docs/superpowers/plans/2026-07-15-container-nested-git-repos-clone-plan.md`

## 结论

**通过（可 Ship）** — 无阻断级问题。

## 检查项

| 项 | 结果 |
|----|------|
| taskProjectService internal nested API | ✅ 单测 + 线上 smoke（34 子仓含 task2app） |
| OpenAPI internal + tenant nested | ✅ |
| api_route_ownership + api-route-to-owner | ✅ |
| credential MergeNested + InheritIdentities | ✅ 单测；task-detail / clone-credentials 双路径 |
| task-detail 继承 identity 供 auto_run | ✅ |
| 发现失败不阻断父仓 | ✅ |
| onlineServiceJS mapPool 并发上限 | ✅ 默认 8 / `BOOTSTRAP_CLONE_CONCURRENCY` |
| machine_container.md §4.4 | ✅ |
| Intent→Event | ✅ 只读发现 + enrich，书面例外（无新业务状态） |

## 非阻断建议

1. 容器镜像需包含更新后的 `onlineServiceJS`（`mapPool`）后并发上限才生效；平台侧 enrich 不依赖镜像版本。
2. 子仓与父仓异 provider 时，继承的 identity 可能无法换票——现状与「同绑定复用」一致，异 provider 需另绑。
3. 可选：task-detail 响应中对 enrich 后的仓数打结构化 metric。
