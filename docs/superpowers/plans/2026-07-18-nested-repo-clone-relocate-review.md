# Review: 子仓克隆后移入父仓 path

日期：2026-07-18  
设计：`docs/superpowers/specs/2026-07-18-nested-repo-clone-relocate-design.md`

## 结论

**通过（无 critical）**

## 检查

| 项 | 结果 |
|----|------|
| nested 先 staging 再 relocate | ✅ `planBootstrapCloneJobs` + `relocateClonedRepo` |
| `parent_repo_url` enrich | ✅ `RepoCloneEntry` + MergeNested |
| path alias 含 `/` | ✅ `sanitizeCloneRelPath` |
| reclone 对齐 | ✅ staging→move + parent |
| 父仓缺失时不误 relocate | ✅ `requireParentDir` |
| 单测 | ✅ Go enrich；JS plan/relocate/collect |
| 契约文档 | ✅ machine_container §4.4；skill.md；intent |

## Log / Intent 审计

- 无新业务 MQ 事件（只读 enrich + 本地 FS）— 书面例外沿用既有 intent。
- relocate 成功/失败写入 bootstrap clone log body / failNote。
