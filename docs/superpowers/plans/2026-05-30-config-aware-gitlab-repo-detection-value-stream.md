# Value Stream: 配置感知的 GitLab 仓库识别

> Derived from design: `docs/superpowers/specs/2026-05-30-config-aware-gitlab-repo-detection-design.md`

## Value Summary

项目成员在详情页对 IP 自托管 GitLab（如腾讯云 `1.117.67.121:8012`）可正常预览分支并一键 OAuth，新增 `gitOauth` 站点无需改代码。

## Related Value Streams

- **project-detail-repo-oauth-row-action**：extension — 在现有 OAuth 行级入口上扩展配置驱动识别
- **git-site-oauth-multi-service-provider-catalog**：dependency — 复用 catalog API 作为前端识别源

## End-to-End Flow

用户进入项目详情页 → 系统识别 repo URL 对应 gitOauth 配置 → 点击「分支列表预览」走 GitLab API + token 路径（或明确鉴权提示）→ 仓库行显示 OAuth 按钮 → 完成授权后再次预览得到分支列表。

## Value Increments

### Increment 1: 后端统一 GitLab 识别（Thin Slice）

**Value to user:** IP GitLab 分支预览不再报 generic 错误，未授权时看到 GitLab 鉴权提示。  
**Scope:** `is_gitlab_repo_url` 委托 `infer_git_provider_from_repo_url`；单测覆盖 tencent/synology host。  
**Depends on:** 无

### Increment 2: 前端 catalog 驱动 OAuth 按钮

**Value to user:** 配置内 GitLab 站点显示 OAuth 授权按钮，可跳转授权。  
**Scope:** `repoOAuthAuthorizeUtils` + ProjectDetail 预加载 catalog；Vitest 覆盖。  
**Depends on:** Increment 1

### Increment 3: E2E 回归护栏

**Value to user:** 升级后 IP GitLab 行为可预测。  
**Scope:** Playwright 用例 + value-stream.yaml step。  
**Depends on:** Increment 2
