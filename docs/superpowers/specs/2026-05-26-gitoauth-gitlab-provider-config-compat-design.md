# 设计文档：gitOauth GitLab Provider 配置兼容修复

## 背景

项目详情页点击 `OAuth 授权` 后，链路在 `gitOauth /api/accounts/gitlab/oauth/start/` 返回 `gitlab=bad_state`，未进入 GitLab `oauth/authorize`。

已通过端到端核验确认：

- task2app `start` 接口成功返回 `authorize_url`
- `authorize_url` 二跳直接回落前端 `gitlab=bad_state`
- `gitOauth` 运行时 `GITOAUTH_PROVIDER_CONFIGS.get("gitlab")` 为空

## 问题定义

`gitOauth` 对 `task2app/conf/port_config.json` 中 `gitOauth` 段的解析只覆盖了部分结构，导致 GitLab provider 配置在运行时未被加载，后续无法拼接 `client_id/redirect_uri/allowedHost`。

## 目标

1. `repo_url=http://localhost:8012/...` 能稳定路由到 `gitlab:local-gitlab`
2. 二跳重定向进入 `http://localhost:8012/oauth/authorize?...`
3. OAuth 参数与本地 GitLab 应用一致（`client_id=8d984...`、`redirect_uri=http://localhost:8001/api/accounts/gitlab-local/oauth/callback/`）
4. 不破坏现有 GitHub / GitLab 兼容配置

## 方案

### 1) Provider 配置归一化增强

- 扩展 `gitOauth` 配置归一化逻辑，兼容以下两种输入：
  - 现有 `list` 结构
  - 当前仓库实际使用的 `dict` 映射结构（key 为 host）
- 统一产出字段：
  - `provider`
  - `service_provider`
  - `provider_key`
  - `allowedHost`
  - `client_id`
  - `client_secret`
  - `redirect_uri`
  - `scope`

### 2) 路由决策保持不变

- 继续以 `service_provider` 优先命中
- 未命中时回退 `allowedHost` 主机命中
- 均未命中时回退 provider 默认项

### 3) 可观测性增强

- 在授权 start 失败时记录结构化日志，区分：
  - token 解码失败
  - provider 配置不存在
  - OAuth 参数缺失

## 风险与回滚

- 风险：配置归一化逻辑改动可能影响 GitHub provider 读取。
- 缓解：补充 GitHub/GitLab 两条单测，覆盖 list/dict 双结构。
- 回滚：保留原逻辑分支，必要时快速回退到之前版本并重启 gitOauth。

## 验收标准

1. Playwright：项目详情点击 `OAuth 授权` 后，二跳进入 `localhost:8012/oauth/authorize`
2. URL 参数匹配本地 GitLab 应用（`client_id/redirect_uri`）
3. 不再出现 `gitlab=bad_state`（配置缺失类）
4. GitHub OAuth 启动链路不回归

