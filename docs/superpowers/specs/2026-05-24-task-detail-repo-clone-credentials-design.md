# Task Detail Repo Clone Credentials Design

## Background

`onlineServiceJS` 在 bootstrap 阶段调用 `server-container-token/task-detail/` 后执行多仓克隆。此前 `task-detail` 即使未返回完整 `repo_clone_credentials` 也会返回 200，导致容器侧进入无凭证 clone，最终以 `git exit 128` 失败，排障成本高。

## Goal

将“仓库克隆凭证完整性”前置到 `task-detail` 接口契约：  
若存在任一仓库缺少凭证，接口直接失败并返回可操作错误，避免容器侧晚失败。

## Non-Goals

- 不修改 OAuth 发证流程。
- 不新增数据库字段。
- 不改变已有 token-exchange 主链路。

## Design

### API Contract

接口：`POST /server-container-token/task-detail/`

- 成功（200）：
  - `project_repos` 与 `repo_clone_credentials` 覆盖一致。
- 失败（409）：
  - `error_code = REPO_CLONE_CREDENTIALS_INCOMPLETE`
  - `detail`
  - `trace_id`
  - `missing_repo_credentials`（缺失仓库 URL 列表）

### Backend Behavior

1. 从 `project_repos[].git_repos[]` 收集期望仓库全集。
2. 构建 `repo_clone_credentials`。
3. 若 `expected - provided` 非空，返回 409。

### onlineServiceJS Behavior

- 识别上述错误码并输出明确文案：
  - “task-detail 未返回完整 repo_clone_credentials；请在任务详情为全部仓库绑定 Git 授权后重试”
- 附带缺失仓库摘要，提升可定位性。

## Testing

- Django 侧：
  - 无凭证场景改为断言 409。
  - 保留有凭证成功场景。
  - 新增部分仓库缺凭证场景，断言 `missing_repo_credentials`。
- Node 侧：
  - 新增结构化错误转换测试（识别错误码并生成可操作文案）。

## Risks and Mitigations

- 风险：历史任务配置不完整时失败率上升。  
  缓解：返回明确错误码与缺失清单，便于快速补齐绑定。

## Value Stream Impact (Input for Step 3)

- Affected streams:
  - `cloud-integration`
    - `cloud-github-binding-string-contract`
    - `cloud-credential-summary-string-contract`
- Field impact:
  - 无新增数据库字段，主要是 API 错误契约强化。
- Test impact:
  - `tests/test_container_runtime_tokens.py`
  - `trae-agent/onlineServiceJS/src/bootstrap.cloneCredentials.test.mjs`
