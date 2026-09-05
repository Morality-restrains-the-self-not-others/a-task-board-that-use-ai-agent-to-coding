# 实施计划：Git 不可用时自动运行不启服

**日期**: 2026-07-18  
**设计**: `docs/superpowers/specs/2026-07-18-autorun-skip-on-git-inaccessible-design.md`

## 任务

- [x] **T1** `probeGitAccessForAutoRun(userID, tenantID, projectIDs) (skipReason string)`  
  - 无 repo → 返回 ""  
  - 调用 `/api/internal/nested-git-repos/?repo_url=&user_id=`  
  - `error` 非空或 HTTP/运输失败 → 返回原因字符串  
- [x] **T2** create/update（含 force）在 `scheduleTaskAutoRunFn` 前调用探测；skip 时写日志 + 响应字段  
- [x] **T3** 扩展 `startAutoRunMockServices`：默认 nested OK；专用测例返回 auth error  
- [x] **T4** 单测：`TestCreateTaskAutoRunSkipsStartWhenNestedGitNeedsAuth`、`TestUpdateTaskForceAutoRunSkipsWhenGitInaccessible`  
- [x] **T5** 意图文档 `docs/intents/backend/autorun_skip_on_git_inaccessible.{intent,test-intent}.md`  
- [x] **T6**（可选）前端：创建成功且 skip 时 toast；`autoRunGateHints` 增加 skip 文案 helper  

## 验证

```bash
cd taskTaskService/src && go test -count=1 -run 'AutoRun' .
```
