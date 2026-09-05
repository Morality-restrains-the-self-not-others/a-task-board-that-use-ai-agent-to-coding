# Value Stream: 项目详情页仓库行 OAuth 授权（分支预览场景）

> Derived from design: `/Users/task2app/.cursor/plans/project-detail-oauth-button_92646d07.plan.md`

## Value Summary
项目成员在项目详情页分支预览遇到 GitLab/GitHub 未授权时，可直接在仓库行点击 `OAuth 授权` 发起授权，不再被错误提示阻塞。

## End-to-End Flow
用户进入项目详情页并查看仓库列表  
→ 仓库行展示 `OAuth 授权` 按钮（可识别 provider）  
→ 点击按钮触发 `/api/accounts/{provider}/app/start/?next&return_key&repo_url`  
→ 后端基于 repo_url 匹配 provider 配置并返回 `authorize_url`  
→ 浏览器跳转至 gitOAuth 授权页完成授权  
→ 回跳项目详情页并再次点击 `分支列表预览`  
→ 成功读取分支列表（或返回明确错误）。

## Value Stages Classification
- **Core value**
  - 项目详情页仓库行提供 OAuth 入口，并可直达授权页。
- **Essential support**
  - start 接口按 `repo_url` 正确命中 provider 配置并生成正确 `authorize_url`。
  - 配置模型支持多 provider（含本地 `localhost:8012`）并提供明确错误契约。
- **Enhancement**
  - 授权成功后自动刷新分支预览（本轮不做）。
- **Future**
  - 按仓库展示 OAuth 已绑定状态徽标与细粒度账号选择（复用任务详情能力）。

## Trigger / Wait / Delivery
- **Trigger**: 用户在项目详情页点击仓库行 `OAuth 授权`。
- **Wait points**:
  - 后端 start 接口生成授权 URL。
  - 用户在授权站点完成 OAuth 回跳。
- **Delivery point**:
  - 用户无需离开项目详情页导航路径即可完成授权，并可继续分支预览。

## Value Increments

### Increment 1: 项目详情页 OAuth 薄切片 (Thin Slice)
**Value to user:** 在项目详情页可直接点击 OAuth 授权，不再只能看到“未检测到可用授权”错误。  
**Scope:**  
- `ProjectDetail` 仓库行按钮可见性（GitHub/GitLab）。
- 点击按钮请求 start API，携带 `next/return_key/repo_url`。
- 后端返回 `authorize_url` 后浏览器跳转。  
**Depends on:** 无（端到端最小可用）。

### Increment 2: Repo URL 驱动的 provider 配置命中
**Value to user:** 本地 GitLab 仓库（`http://localhost:8012/...`）会跳到对应配置的授权服务，不再误跳到默认远程域名。  
**Scope:**  
- start 接口按 `repo_url` 命中对应 `service_provider`。
- provider 配置支持按 `allowedHost` 匹配多实例。
- 缺少可用服务基址时返回明确 503 错误。  
**Depends on:** Increment 1。

### Increment 3: 配置模型统一与回归护栏
**Value to user:** 多环境配置稳定，升级后行为可预测。  
**Scope:**  
- `gitOauth` 配置支持对象结构（key=allowedHost）并归一化。
- 测试覆盖：按钮行为、start URL 参数、本地 provider 基址命中、无回退行为。  
**Depends on:** Increment 2。

## Mapping To Existing Streams
- `task-detail-oauth-repo-url-row-action`（复用仓库行 OAuth 动作设计模式）
- `gitoauth-binding-state-persistence`（扩展鉴权链路到项目详情页入口）

## Proposed New Stream Entry
- **name**: `project-detail-repo-oauth-row-action`
- **domain**: `项目与工作空间`
- **description**: 项目详情页仓库行 OAuth 授权入口与 repo_url 驱动的 provider 路由

## Candidate YAML Steps (for config write)
1. `project-detail-repo-oauth-button-thin-slice`（active）
2. `project-detail-repo-oauth-provider-routing`（active）
3. `project-detail-repo-oauth-regression-guard`（planned）
