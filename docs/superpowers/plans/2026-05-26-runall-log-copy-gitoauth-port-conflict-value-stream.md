# Value Stream: runAll 日志复制与 git-oauth 端口冲突恢复启动

> Derived from design: `docs/superpowers/specs/2026-05-25-runall-stability-first-design.md`  
> Approved scope supplement: 在 `http://localhost:9999/` 增加日志复制按钮，并修复 `git-oauth` 启动时 `Address already in use` 的可恢复性。

## Value Summary
本价值流为本地运维/开发用户提供“可快速复制日志定位问题 + 启动失败可自动恢复”的闭环，减少手工排障和重复重启成本。

## End-to-End Flow
[用户在 runAll 页面点击“日志”/“启动 git-oauth”] -> [runAll 拉取并展示日志 + 提供复制动作] -> [启动前检测并处理端口冲突] -> [git-oauth 成功监听并通过健康检查] -> [用户完成日志分享与服务恢复]

## Value Stages
- **Trigger:** 用户在 `runAll` 状态页执行“查看日志并复制”或“启动 git-oauth”。
- **Stage 1 (Core value):** 日志面板提供一键复制，支持把当前可见日志快速带出到外部沟通渠道。
- **Stage 2 (Essential support):** 启动链路在 preflight/launch 阶段识别 `git-oauth` 端口占用，并把冲突进程信息转成可诊断错误。
- **Stage 3 (Core value):** 在可控策略下自动清理外部占用并重试绑定端口，避免用户陷入“手工杀进程->重启”的循环。
- **Stage 4 (Enhancement):** UI 给出复制成功/失败反馈与冲突恢复提示，提升可用性与可理解性。
- **Delivery point:** 用户能成功复制日志并看到 `git-oauth` 从失败恢复为健康状态。

## Wait / Dependency Points
- 日志复制依赖日志面板已有内容加载完成（`/api/logs` 返回）。
- `git-oauth` 自动恢复依赖 preflight 对端口监听 PID 的识别与可终止判断。
- 启动成功判定依赖健康检查 `http://127.0.0.1:8002/api/health/` 达标。

## Value Increments

### Increment 1: 日志复制最小闭环 (Thin Slice)
**Value to user:** 用户可在日志面板直接复制当前日志文本用于排障沟通。  
**Scope:** 增加“复制日志”按钮，调用浏览器剪贴板 API，失败时回退到可见错误提示。  
**Depends on:** nothing

### Increment 2: git-oauth 端口冲突可诊断化
**Value to user:** 启动失败时明确看到“端口占用”及冲突 PID，不再只有 Python traceback。  
**Scope:** 在启动前记录冲突来源、失败分类与修复提示，状态页可见。  
**Depends on:** Increment 1

### Increment 3: git-oauth 端口冲突自动恢复
**Value to user:** 点击“启动”后可自动清理外部占用并完成重试，服务更容易一次启动成功。  
**Scope:** 对 `git-oauth` 启动链路补齐“冲突识别 -> 终止冲突进程 -> 再次绑定端口 -> 健康校验”。  
**Depends on:** Increment 2

### Increment 4: 交互反馈增强
**Value to user:** 用户明确知道“复制成功/失败”“冲突已自动恢复/仍需手工处理”。  
**Scope:** 日志面板与服务行增加轻量反馈文案与状态提示，不改变核心 API 契约。  
**Depends on:** Increment 3

## Existing Value Stream Impact
- **Affected existing streams:**  
  - `gitlab-oauth-scope-failfast-governance`（间接受益：本地 `git-oauth` 可用性提高，减少认证链路调试噪音）  
  - `project-detail-repo-oauth-row-action`（间接受益：项目页 OAuth 调试链路更稳定）
- **New stream needed:** 是，建议新增 `runall-log-copy-gitoauth-port-conflict-recovery`（建议 domain：`云平台与资源`）。
- **Fields impact:** 新增运行态字段登记（见 YAML 草案），不涉及业务数据库 schema 变更。
- **Test impact:** 新增/补充 `runAll` 相关 UI 与 preflight 启动恢复测试；价值流阶段先以 `planned` 测试入口登记。
- **Status changes:** 新增流的步骤建议初始为 `planned`，实现后逐步切 `active`。
- **Cross-stream dependencies:** 无新的强业务依赖，仅对 OAuth 相关开发流提供环境稳定性支撑。
