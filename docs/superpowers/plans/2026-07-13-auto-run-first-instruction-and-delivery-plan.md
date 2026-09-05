# 实施计划：auto_run 首指令与自动交付

- **日期**: 2026-07-13
- **设计**: `docs/superpowers/specs/2026-07-13-auto-run-first-instruction-and-delivery-design.md`
- **权限**: `docs/superpowers/specs/2026-07-13-auto-run-first-instruction-and-delivery-permission-analysis.md`
- **价值流**: `docs/superpowers/plans/2026-07-13-auto-run-first-instruction-and-delivery-value-stream.md`
- **架构**: v18 target — `docs/architecture/v18-application-integration-20260713-1427-claude.{puml,archimate,mermaid.md}`

## Task A — Go task-detail：`auto_run` + `repo_git_identities`

**垂直切片**：Credential 契约扩展 + 单测 + 文档 §4.4（T1、T2）

- [ ] A1 `TaskSnapshot` / `ContainerTaskDetail` 增 `AutoRun`、`RepoGitIdentities` 字段
- [ ] A2 `FetchTaskSnapshot` SELECT `COALESCE(auto_run, 0)`
- [ ] A3 `GitIdentitySnapshot` 增 `UserName`/`UserEmail`（`git_user_name`/`git_user_email`）
- [ ] A4 `FetchTaskDetail` 组装 `repo_git_identities`（与 `FetchRepoIdentities` 同源）
- [ ] A5 单测：T1 `task.auto_run=true`；T2 身份列表含 name/email；无绑定时空数组
- [ ] A6 更新 `task2app/Saas_project/skillList/machine_container.md` §4.4 示例 JSON

**验证命令**：

```bash
cd /tmp/ram-work/taskCredentialService && go test ./application/ -count=1 -run 'TaskDetail|FetchTask'
cd /tmp/ram-work/taskCredentialService && go test ./... -count=1
rg 'auto_run|repo_git_identities' /tmp/ram-work/task2app/Saas_project/skillList/machine_container.md
```

---

## Task B — Node：首指令编排（bootstrap → createJob）

**垂直切片**：`composeAutoRunCommand` + bootstrap 后触发 + 幂等标志（T3、T4、T5、T8 首指令侧）

- [ ] B1 新增 `autoRunOrchestration.mjs`（或等价模块）：`composeAutoRunCommand(title, description)`
- [ ] B2 `bootstrap.mjs` 暴露最后一次 task-detail（返回值或 getter）供 `server.mjs`
- [ ] B3 `server.mjs`：`BOOTSTRAP_COMPLETE` + `registerBootstrapCloneJob` **之后**判断 `auto_run` 并 `createJob`
- [ ] B4 写 `runtime/auto_run_first_job.json`；已存在则跳过（T8）
- [ ] B5 job record 标记 `auto_run_first=true`
- [ ] B6 单测：T3 命令组装；T4 调用一次；T5 不调用；title+description 皆空 WARN

**验证命令**：

```bash
cd /tmp/ram-work/trae-agent/onlineServiceJS && npm run test:unit -- src/autoRunOrchestration.test.mjs src/bootstrap.autoRun.test.mjs 2>/dev/null || node --test src/autoRunOrchestration.test.mjs src/bootstrap.autoRun.test.mjs
cd /tmp/ram-work/trae-agent/onlineServiceJS && npm run test:unit
```

---

## Task C — Node：job 完成后自动交付

**垂直切片**：close 钩子 → identities sync → commit → oauth-refresh-push（T6、T7、T8 交付侧）

- [ ] C1 `jobsRuntime` `proc.on('close')`：`auto_run_first && completed` 进入交付
- [ ] C2 抽 `syncRepoIdentities(layerId, repos)`（内部调用，不经 HTTP）
- [ ] C3 `git add -A` + `git commit -m <title>`；nothing to commit 可继续
- [ ] C4 `runLayerOauthRefreshPush({ layerId, targetBranch })`（工作分支回退逻辑复用）
- [ ] C5 写 `runtime/auto_run_delivery.done`；日志 COMPLETE / FAILED
- [ ] C6 单测：T6 mock sync/commit/push；T7 failed 不交付；T8 done 已存在跳过

**验证命令**：

```bash
cd /tmp/ram-work/trae-agent/onlineServiceJS && node --test src/autoRunDelivery.test.mjs src/jobsRuntime.autoRunDelivery.test.mjs
cd /tmp/ram-work/trae-agent/onlineServiceJS && npm run test:unit
rg 'AUTO_RUN_DELIVERY_(COMPLETE|FAILED)' /tmp/ram-work/trae-agent/onlineServiceJS/src/
```

---

## Task D — 文档与意图同步

**垂直切片**：价值流测试点、意图 checkbox、架构索引（S9）

- [ ] D1 功能意图 `006_*.intent.md` 验收项勾选（交付后）
- [ ] D2 测试意图 `006_*.test-intent.md` 与 T1–T8 绿勾对齐
- [ ] D3 `docs/flows/` 或 ProjectFeature 中 `create-task-auto-run` 流增步骤（若存在该流文档）
- [x] D4 确认 `docs/architecture/VERSION_HISTORY.md` v18 target 条目与设计文档互链

**验证命令**：

```bash
rg 'auto-run-first-instruction|auto-run-delivery|T[1-8]' /tmp/ram-work/task2app/docs/intents/engineering/cloud/006_auto_run_first_instruction_and_delivery.test-intent.md
rg 'v18.*auto_run' /tmp/ram-work/docs/architecture/VERSION_HISTORY.md
ls /tmp/ram-work/docs/architecture/v18-application-integration-20260713-1427-claude.*
```

---

## 全量回归（交付前）

```bash
cd /tmp/ram-work/taskCredentialService && go test ./... -count=1
cd /tmp/ram-work/trae-agent/onlineServiceJS && npm run test:unit
```

## 任务依赖

```text
Task A ──┬──> Task B ──> Task C
         └──> Task D（可与 B/C 并行，最终对齐）
```
