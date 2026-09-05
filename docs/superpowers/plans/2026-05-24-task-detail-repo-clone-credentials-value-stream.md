# Value Stream: Task Detail Repo Clone Credentials Completeness

> Derived from design: `docs/superpowers/specs/2026-05-24-task-detail-repo-clone-credentials-design.md`

## Value Summary

容器启动阶段在 `task-detail` 即可得到“可克隆 / 不可克隆”的确定结果，缺失凭证时返回明确错误与缺失仓库列表，避免晚失败。

## End-to-End Flow

容器启动触发 `task-detail` 请求 → 后端汇总任务仓库 → 后端构建每仓凭证并做覆盖率校验 → 完整则返回可克隆上下文 / 不完整则返回 409 结构化错误 → 容器输出可操作日志并停止 bootstrap。

## Value Stages

- Trigger: `onlineServiceJS` 在 `runBootstrapAfterListen` 调用 `server-container-token/task-detail/`
- Stage 1 (Core value): 后端返回仓库列表与克隆凭证
- Stage 2 (Essential support): 后端执行凭证完整性校验并生成缺失清单
- Stage 3 (Core value): 容器侧识别结构化错误码并给出可操作指引
- Delivery point: 用户在首次失败点即看到“去任务详情绑定授权”的明确原因

## Wait / Dependency Points

- 依赖 `TaskRepoIdentity` / OAuth 凭据可解析
- 依赖 `project_repos` 组装完成后才能做凭证覆盖率校验

## Value Increments

### Increment 1: task-detail strict contract (Thin Slice)
**Value to user:** 缺失凭证时立刻失败，不再进入无凭证 clone。  
**Scope:** `task-detail` 增加凭证覆盖率校验；返回 `REPO_CLONE_CREDENTIALS_INCOMPLETE`。  
**Depends on:** 无（在现有接口内闭环）。

### Increment 2: actionable bootstrap error
**Value to user:** 容器日志直接提示“去任务详情绑定授权”，并附缺失仓库摘要。  
**Scope:** `onlineServiceJS` 在 bootstrap 捕获并翻译 `task-detail` 结构化错误。  
**Depends on:** Increment 1 错误契约稳定输出。

### Increment 3: regression safety
**Value to user:** 后续变更不回退为晚失败。  
**Scope:** Django + Node 单测覆盖无凭证、部分凭证、完整凭证三类路径。  
**Depends on:** Increment 1, 2。
