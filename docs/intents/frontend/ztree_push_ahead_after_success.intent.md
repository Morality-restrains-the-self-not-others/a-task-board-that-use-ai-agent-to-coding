# zTree 推送成功后 ahead 文案与推送按钮


## 业务意图 → 事件对照

> 存量回填（自动）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。事件名若为启发式占位，可在后续迭代精修。

**无对应事件**：纯前端展示/交互或设计治理，无服务端业务状态变更意图。

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| zTree 推送成功后 ahead 文案与推送按钮 | — | — | — | 纯前端展示/交互或设计治理，无服务端业务状态变更意图 |
## 变更记录
- 2026-07-12：首次记录。线上任务 `task_12949237300462721867`，traceId `35f69315-f30a-456f-975c-f461dfa0243b`：`container-layer-git-push` 返回 `ok: true` / `push_ok: true`，但 zTree 仍显示「1 个提交可推送」及推送按钮。
- 2026-07-12（复现加深）：任务 `task_12953905731855947865`（fork 自上一任务）。现象：点击推送后短暂变为「2 个提交已推送」，点击节点/刷新后又变回「2 个提交可推送」。容器内核实：主仓 `somanyad` 仍在 `master`（`@{u}=origin/master`），`origin/master..HEAD=2`；但 `HEAD` 已与 `origin/feature/..._runIt` 相同（推送与 `markOriginRemoteTrackingToHead` 已生效）。`ahead` 仍按 `@{u}..HEAD` 计算，故刷新覆盖乐观 UI。
- 2026-07-12（P0 落地）：`layerGitRemoteSnapshot` 支持 `compareBranch` + 层内 `git_push_compare_branch` 记忆；推送成功写入记忆并以工作分支远程算 ahead。单测 T5/T5b/T7/T8；relay overlay 扩展至 `layerFs.mjs` / `layerGitOauthPush.mjs`。本任务层图刷新后不再出现「可推送」。

## 根因（当前）

### 现象链路
1. 推送 API 成功 → 前端 `markLayerGitRemotePushedInSnapshot` 乐观写 `ahead=0` + `last_pushed_count=N` → 显示「N 个提交已推送」。
2. 随后 `refreshLayerGraphFromServer` / 点击节点触发层图刷新 → 容器 `layerGitRemoteSnapshot` 仍返回 `ahead=N`。
3. `applyLayerGraphFromPayload` **仅在** `next.ahead===0` 时保留 `last_pushed_count`；`ahead>0` 时直接采用服务端快照 → 文案回到「N 个提交可推送」。

### 容器侧真相（本任务实测）
| 仓库 | 当前分支 | `@{u}` | `@{u}..HEAD` | `origin/<工作分支>..HEAD` |
|---|---|---|---|---|
| `somanyad`（`layerPrimaryGitWorkdir`） | `master` | `origin/master` | **2** | **0**（已推送） |
| `somanyad-emailD` | 工作分支 | `origin/<工作分支>` | 0 | 0 |

- 推送使用 `HEAD:refs/heads/<target_branch>`，**不要求**本地已在工作分支。
- `markOriginRemoteTrackingToHead` 只更新 `refs/remotes/origin/<target>`，**不改变**当前分支的 `@{u}`。
- `layerGitRemoteSnapshot` 用 `git rev-list --count @{u}..HEAD`，在「本地仍在 master/基准分支、已推到工作分支」时会把「相对 merge base 的领先」误当成「还可推送」。

### 叠加因素
1. 主仓未切到工作分支（bootstrap `ensureRepoOnWorkBranch` 未生效或被跳过），放大了 `@{u}` 与推送目标不一致。
2. 前端乐观态无法对抗服务端持续返回的 `ahead>0`（合并策略故意不保留）。
3. 既有单测在断言前手动 `set-upstream-to=origin/<feature>`，掩盖了「只 mark remote-tracking、不改 @{u}」时 ahead 不归零的问题。

## 解决方案（按优先级）

### P0 — 修正「可推送」语义（容器，权威数据源）
`ahead` 应对齐 zTree「推送到工作分支」语义，而不是盲目信 `@{u}`。

推荐实现（`layerFs.mjs` → `layerGitRemoteSnapshot`）：
1. 入参可选 `compareBranch`（推送目标 / 任务工作分支名）。
2. 比较顺序：
   - 若存在 `refs/remotes/origin/<compareBranch>` → `origin/<compareBranch>..HEAD`
   - 否则若存在 `refs/remotes/origin/<current_branch>` → `origin/<current_branch>..HEAD`
   - 否则回退 `@{u}..HEAD`
   - 皆无 → `no_upstream: true`
3. 推送成功路径（`server.mjs` `/git/push`、`layerGitOauthPush.mjs`）：
   - 继续 `markOriginRemoteTrackingToHead(workdir, target)`
   - 调用 `layerGitRemoteSnapshot(layerId, { compareBranch: target })` 写入响应
4. `GET /api/layers` / jobs 快照：从任务详情解析工作分支传入 `compareBranch`（与前端 `resolveLayerGraphPushTargetBranch` 同源），避免刷新后再次用错基准。

### P1 — 推送后对齐本地分支跟踪（容器）
多仓 OAuth/API 推送成功后，对每个成功仓：
- 若本地已在 `target_branch`：`git branch --set-upstream-to=origin/<target>`
- 若本地不在工作分支：优先 `ensureRepoOnWorkBranch`（或至少记录 warning）；**不要**把 `master` 的 upstream 强行改成 feature（易误导后续 merge）。

### P2 — 前端防御（次要，不能单独修根因）
- 保持乐观更新与 `last_pushed_count`。
- 可选：刷新后若响应 `git_remote.ahead===0` 则保留文案（已有）；**不要**在服务端仍报 `ahead>0` 时用前端硬盖，以免掩盖真未推送。
- 单元测试补：模拟「push 后 refresh 仍 ahead>0」应失败于容器契约，而非前端吞掉。

### P3 — 测试补强
- 扩展 `layerFs.markRemoteTracking.test.mjs`：**不要**在断言前改 upstream；断言「仅 mark `origin/<target>` 后，若 snapshot 带 `compareBranch=target` 则 ahead=0；仅靠 `@{u}=origin/main` 则旧逻辑仍 >0」。
- Playwright：推送成功 → 手动刷新层图 → 仍为「已推送」且无推送按钮。
- 多仓：主仓停在 base、从仓在 work 时，层级 `ahead` 以工作分支远程为准。

## 验收
1. 推送成功后 zTree 显示「N 个提交已推送」，推送按钮消失。
2. 点击其它节点或手动刷新层图后，仍保持「已推送」（服务端 `ahead===0`），不会回退为「可推送」。
3. 仅当相对**工作分支远程**仍有未推送提交时显示推送按钮与「可推送」。
4. 容器单测覆盖 `compareBranch` / mark remote-tracking；Playwright mock + 可选 CDP 真机回归。

## 历史修复（仍有效，但不充分）
- 容器：推送成功后 `markOriginRemoteTrackingToHead`，响应附带 `git_remote`。
- 前端：`canPush` 仅当 `ahead>0`；`ahead===0` 且有 `last_pushed_count` 时展示「N 个提交已推送」；推送成功后乐观写快照并在 `ahead===0` 刷新时保留 `last_pushed_count`。
