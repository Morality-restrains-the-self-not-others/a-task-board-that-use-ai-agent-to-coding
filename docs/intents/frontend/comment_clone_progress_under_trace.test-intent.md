# 测试意图：评论级项目克隆进度条（启动 TraceId 下方）

## 测试目标

验证评论执行细节在「启动 TraceId」下方按该评论日志 / live SSE 展示项目克隆**总进度条**；多仓子进度默认折叠；自动重试中不显示手动按钮，失败耗尽后显示。

## 测试分层

| 层 | 落点 |
|----|------|
| 单元 | `taskFE/app/src/utils/commentCloneProgressFromLogs.test.js` |
| 单元 | `taskFE/app/src/utils/commentCloneProgressRepoCatalog.test.js` |
| 单元 | `taskFE/app/src/composables/taskDetail/applyContainerGitCloneProgress.test.js` |
| 单元 | `taskFE/app/src/components/task-detail/useCommentSectionCloneProgressRows.test.js` |
| 单元 | `taskFE/app/src/composables/taskDetail/taskDetailFetchFns.repoCloneIdentity.test.js` |
| 单元 | `trae-agent/onlineServiceJS/src/bootstrap.cloneFailureFooter.test.mjs` |
| 单元 | `trae-agent/onlineServiceJS/src/saasTaskCloud.parseGitCloneProgress.test.mjs` |
| 单元 | `taskFE/app/src/utils/taskDetailContainerCloneProgress.test.js` |
| 组件 | `taskFE/app/src/components/task-detail/TaskDetailCommentCloneProgress.test.js` |
| 组件 | `taskFE/app/src/components/task-detail/TaskDetailCommentExecutionDetails.test.js` |

## 用例矩阵

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| T1 | 日志含 `【项目克隆】(1/2) alpha … 33%` 后 `… 66%` | `parseCommentCloneProgressFromLogs` | 一行 alpha，progress=66 |
| T2 | 两仓各有百分比行 | 解析 | 两行，各自最新百分比 |
| T3 | `项目克隆 (1/1) 完成 alpha` | 解析 | alpha progress=100 |
| T4 | `【项目克隆】(1/2) 失败 relayToTrae: fatal` | 解析 | 该行 failed=true |
| T5 | 仅「容器调度排队中」 | 解析 | `[]` |
| T6 | live map 有 comment C1 的仓进度，日志较旧 | `resolveCommentCloneProgress({ commentId: C1 })` | 同仓采用 live 百分比；其它日志仓保留 |
| T7 | 无 live；soleActive 为 C1；任务级 entries 非空 | resolve C1 | 回退任务级 entries |
| T8 | SSE `comment_id=C1` progress=40 | `applyContainerGitCloneProgress` | `byCommentId.C1` 有对应仓；任务级 map 仍更新 |
| T9 | ExecutionDetails 传入 rows + startTraceId | 挂载 | TraceId 下方存在 `comment-execution-clone-progress` |
| T10 | rows 为空 | 挂载 | 无进度条节点 |
| T11 | tab 模式切到服务器运行状态 | 点击 runtime tab | 进度条仍在 container-meta 内 |
| T12 | 日志含 `(31/34) … 100%` 与 `(32/34) 准备第 2/3 次重试` | `overallCloneProgressPct` | 分母=34；重试行非 failed |
| T13 | 同仓先 40% 再「准备第 2/3 次重试」 | 解析 | 保留 40%；无手动重试 |
| T14 | 重试后出现 `失败 name: fatal`（无 URL） | 解析 | failed=true；无 catalog 时不显示手动重试 |
| T24 | 冷打开 `(1/1) 失败 ram-work: git exit 128` + 关联仓目录含 ram-work.git | enrich catalog | 补上 repoUrl，显示「手动重试」 |
| T25 | `(1/1)` 且 catalog 仅 1 条 | enrich | 用该 URL 显示手动重试 |
| T26 | 失败行 label=docs，catalog 有 parent/alias | enrich | 带 parentRepoUrl/cloneAlias |
| T27 | 两仓同 basename | enrich | 不猜测 URL，不显示按钮 |
| T28 | 失败文案已内嵌 `https://…git` | 解析 | 直接 repoUrl，显示手动重试 |
| T15 | 日志两仓 + live 只更新其中一仓 | resolve | 合并后仍两行，live 覆盖同仓百分比 |
| T16 | 两仓 rows | 挂载 CloneProgress | 总条可见；子仓 details 默认关闭 |
| T17 | 失败且有 repoUrl | 点击「手动重试」 | emit `repo-reclone` `{ repoUrl }` |
| T18 | ExecutionDetails 有 commentId，失败行有 git URL | 点击「手动重试」 | emit 含 `commentId`/`comment_id` |
| T19 | onRepoReclone payload 含 commentId | POST repo-reclone | path `/comment_id/{id}/` 且 body.comment_id 同值 |
| T29 | 任务级 `containerEndpointRegistered=false` + payload 含 commentId | 点击重试 | 仍 POST；状态不是「容器未启动」 |
| T20 | 无 comment_id | 点击重试 | 不发请求，状态「缺少评论ID」 |
| T21 | live 已有分仓失败后再到全局摘要 SSE | applyContainerGitCloneProgress | 不写入 `__global__` 重复行 |
| T22 | 全局摘要行 + 分仓失败行 | omitRedundantGlobalCloneRow | 只保留分仓行 |
| T23 | 1/1 仓均失败 | resolveBootstrapCloneFailurePolicy | 文案含「均失败」，不含「其余已就绪」 |
| T30 | 日志 `9%` → `完成 ram-work` → 再 `9%` | `parseCommentCloneProgressFromLogs` | progress=100，message 含「完成」 |
| T31 | 日志已完成 + live 9% | `resolveCommentCloneProgress` | 仍 100%，不被 live 盖掉 |
| T32 | SSE 9% → 100%「完成」→ 再 9%（含 recv 9） | `applyContainerGitCloneProgress` | comment/task map 仍 100%，recv/unpack 仍 100 |
| T33 | stderr 同时 Receiving 9% 与 Checkout 100% | `parseGitCloneProgressPhases` | overall=100 |
| T34 | 已 100/100 后迟到 `recv_progress: 9` | `mergeCloneProgressSubPhases` | recv/unpack 仍 100 |
| T35 | 启动日志/live 9% + 引导日志 `【项目克隆】克隆完成` | `applyBootstrapCloneDoneToRows` / catalog | progress=100 |
| T36 | 分仓 100% → SSE `仓库克隆已完成` → 再 `(1/1) … 3%` | `applyContainerGitCloneProgress` | 评论条仍 100%，message 含「完成」 |
| T37 | 父仓 100% + nested 0% 且 auto_clone=false | `omitSkippedNestedCloneProgressRows` | 只留父仓，overall=100 |
| T38 | 评论 live 含父仓 100% + nested 0%，项目 `auto_clone_nested_repos=false` | `useCommentSectionCloneProgressRows` | 只留父仓，overall=100 |

## 数据与环境

- 不依赖真实容器 / SSE；Vitest + jsdom。
- 日志样例与 onlineServiceJS `【项目克隆】(i/n) name … N%` 格式对齐。

## 通过标准

上表用例全部绿灯；无克隆行的既有 ExecutionDetails 测例不回归。
