# Value Stream: 任务详情 OAuth 绑定方案调整

> Derived from approved design: `.cursor/plans/调整仓库授权方案_4e0a71a5.plan.md`

## Value Summary

当用户在任务详情进行“直接启动”前，系统按仓库地址逐行判断 OAuth 绑定状态，只在未绑定仓库行提供 `OAuth 绑定` 动作，并在预检失败时给出缺失仓库与可操作指引，减少“启动后才失败”的排障成本。

## End-to-End Flow

[Trigger: 用户进入任务详情并查看关联仓库] → [阶段1：按仓库地址检查 OAuth 绑定并在未绑定行显示按钮] → [阶段2：用户在目标仓库行完成 OAuth 绑定] → [阶段3：任务仓库逐项保存授权账号] → [阶段4：启动前预检 repo clone credentials] → [通过则启动 relay / 失败则可视化引导修复] → [用户获得“可启动”或“明确修复路径”]

## Value Stages（按价值分类）

- Core value
  - 将 OAuth 入口从全局按钮下沉到仓库行，避免误触发和无关跳转。
  - 基于 `repo_url` 逐仓库判断绑定状态，仅未绑定仓库显示 `OAuth 绑定`。
  - 预检失败时在启动位点直接展示缺失仓库与修复动作。
- Essential support
  - 复用 `missing_repo_credentials` 契约，保证前端可稳定消费结构化字段。
  - OAuth 连接状态查询支持携带 `repo_url` 以匹配仓库地址级判定。
  - 保持现有 API 行为不变，避免后端行为回归。
- Enhancement
  - 在 relay 启动面板增加显式引导卡片，减少用户在任务详情内来回定位成本。
- Future
  - 将失败原因细分为 `no_identity|oauth_unbound|token_fetch_failed|provider_unknown`。
  - 把“启动前预检”前置为页面被动校验能力（无需点击启动即可发现问题）。

## Wait / Dependency Points

- 阶段2依赖阶段1：未完成 OAuth 时，仓库级账号绑定不可达成。
- 阶段3依赖阶段2：仓库授权未保存时，`repo-clone-credentials` 覆盖率校验必然失败。
- relay 启动依赖阶段4：预检失败应阻断 `start`，防止进入无凭证 clone。

## Delivery Point

用户在仓库行即可完成精准绑定，点击“启动”时不再只看到笼统报错，而是获得“缺失仓库 + 去哪里做什么”的明确路径；完成绑定后启动成功率提升。

## Value Increments

### Increment 1: 仓库行级 OAuth 入口薄切片（Thin Slice）
**Value to user:** 在未绑定仓库行直接看到并点击 `OAuth 绑定`，避免全局入口误导。  
**Scope:**  
- 移除全局 OAuth 按钮。  
- 按 `repo_url` 逐仓库检测绑定状态。  
- 仅未绑定仓库行展示 `OAuth 绑定` 按钮。  
**Depends on:** 无。

### Increment 2: 启动前预检可视化引导（Core Value）
**Value to user:** 在点击启动的关键时刻看到缺失仓库清单与下一步操作。  
**Scope:**  
- 预检失败时消费 `missing_repo_credentials`。  
- 展示缺失摘要与醒目的修复提示，阻断 `start`。  
**Depends on:** Increment 1。

### Increment 3: 契约稳定性回归保护（Essential Support）
**Value to user:** 结构化错误契约长期稳定，行级 OAuth 引导不易失效。  
**Scope:**  
- 后端测试断言 `status/message/error_code/trace_id/missing_repo_credentials`。  
- 前端单测覆盖：未绑定仓库显示行级按钮、已绑定仓库不显示按钮、行级点击触发授权。  
- e2e 覆盖预检失败时引导展示与启动阻断。  
**Depends on:** Increment 2。

### Increment 4: 失败原因精细化（Future）
**Value to user:** 直接看到“为什么缺失”而非仅“缺哪个仓库”。  
**Scope:** 新增原因分类字段并做前端差异化引导。  
**Depends on:** Increment 3。

## Affected Existing Streams (input)

- `task-detail-repo-clone-credentials-contract`（active）
- `oauth-token-fetch-timeout-governance`（active）
- `task-detail-runtime-relay`（active）
- `task-detail-repo-clone-credentials-decoupling`（planned）

## Candidate Field Mapping

- `saas-backend.projects_taskrepoidentity.repo_url`
- `saas-backend.projects_taskgithubrepooauthbinding.github_user_id`
- `saas-backend.projects_taskgithubrepooauthbinding.repo_slug`
- `git-oauth.api_githubappusercredential.github_user_id`
