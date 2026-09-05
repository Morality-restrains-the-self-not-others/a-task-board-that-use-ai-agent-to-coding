# Value Stream: Task Detail 与 Repo Clone Credentials 解耦

> Derived from design: `docs/superpowers/specs/2026-05-24-task-detail-repo-clone-credentials-design.md`（并结合已批准的解耦调整：`task-detail` 只返回仓库列表，凭证改由独立接口获取）

## Value Summary
当用户在 task-detail 页面点击“启动”时，容器可先稳定获取任务与仓库列表，再按需拉取仓库克隆凭证；凭证缺失会在专用接口明确失败并给出缺失清单，提升可诊断性并降低耦合风险。

## End-to-End Flow
[Trigger: task-detail 页面点击启动] → [relay 启动 onlineServiceJS] → [拉取 task-detail 获取 project_repos] → [拉取 repo-clone-credentials 获取凭证] → [按仓库克隆] → [用户在界面收到成功或可操作错误]

## Value Stages
- Core value: 解耦 `task-detail` 与凭证下发，形成“两段式引导”。
- Essential support: 新增 `repo-clone-credentials` 接口与 409 错误契约迁移。
- Enhancement: 错误文案和缺失仓库摘要保持结构化一致，降低排障成本。
- Future: 启动前前端预检（可选，不阻塞当前交付）。

## Wait / Dependency Points
- `repo-clone-credentials` 依赖 `task-detail` 先返回 `project_repos`。
- clone 阶段依赖凭证接口成功返回映射；若缺失则 fail-fast（409）。
- 现有 token-exchange 主链路保持不变，是本改造的前置不变量。

## Delivery Point
用户点击“启动”后，系统不再依赖 `task-detail` 同时携带凭证；即使失败，也会收到明确的 `REPO_CLONE_CREDENTIALS_INCOMPLETE` 与缺失仓库列表。

## Value Increments

### Increment 1: 独立凭证接口薄切片（Thin Slice）
**Value to user:** 系统具备“先拿仓库列表，再拿凭证”的端到端能力，失败可定位。  
**Scope:**  
- 新增 `POST /server-container-token/repo-clone-credentials/`。  
- 复用现有 access_token 校验、路径匹配、过期校验。  
- 返回 `repo_clone_credentials`；缺失时返回 `409 + REPO_CLONE_CREDENTIALS_INCOMPLETE + missing_repo_credentials`。  
**Depends on:** 无（可独立上线，不影响旧接口）。

### Increment 2: onlineServiceJS 切换到两段式调用（Core Value）
**Value to user:** 启动流程从单接口耦合切换为可维护的两阶段流程，稳定性提升。  
**Scope:**  
- `runBootstrapAfterListen` 改为：先调 `task-detail`，再调 `repo-clone-credentials`。  
- clone 逻辑继续复用 `(urls, credRoot)`。  
- 错误转换继续识别 `REPO_CLONE_CREDENTIALS_INCOMPLETE`，输出可操作文案。  
**Depends on:** Increment 1。

### Increment 3: task-detail 契约瘦身（Essential Support）
**Value to user:** `task-detail` 响应职责更清晰，接口演进风险更低。  
**Scope:**  
- `task-detail` 保留 `task + project_repos`。  
- 移除 `repo_clone_credentials` 字段与凭证完整性 409 逻辑。  
- 保持 tenant/workspace/task 路径与鉴权模型不变。  
**Depends on:** Increment 2。

### Increment 4: 回归与可观测性加固（Enhancement）
**Value to user:** 变更后行为可验证、可追踪，减少回归概率。  
**Scope:**  
- 更新后端与 onlineServiceJS 单测、e2e mock。  
- 在 bootstrap 出站日志中明确记录两段式调用顺序。  
**Depends on:** Increment 2、3。

## Suggested Test Mapping
- `task2app/Saas_project/tests/test_container_runtime_tokens.py`
- `trae-agent/onlineServiceJS/src/bootstrap.cloneCredentials.test.mjs`
- `trae-agent/onlineServiceJS/e2e/bootstrap-clone-host-alias-credential.api.spec.mjs`

## Value Stream YAML Draft (for later write)
- stream name: `task-detail-repo-clone-credentials-decoupling`
- domain: `任务协作`
- description: `将 task-detail 与 repo_clone_credentials 解耦为两阶段引导调用`
- planned steps:
  - `repo-clone-credentials-endpoint-thin-slice`
  - `online-service-two-phase-bootstrap`
  - `task-detail-contract-slimming`
  - `decoupling-regression-guard`
