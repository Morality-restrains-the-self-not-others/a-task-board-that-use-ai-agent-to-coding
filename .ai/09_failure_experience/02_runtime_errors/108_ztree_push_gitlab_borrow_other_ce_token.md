# [运行时] ztree GitLab 推送：借其它 CE 的 OAuth token → HTTP Basic Access denied

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-22
- 编号：108
- 维护者：Trae AI 团队

## 现象

- 修完 [107](./107_ztree_push_gitlab_oauth_missing_write_repository.md) 后，用户已对 `gitlab:tencent-sh-1` 重新授权，`/oauth/token/info` 含 `write_repository`。
- 本机用该 token `git ls-remote` / `git push --dry-run` 成功。
- 公网「提交并创建PR」仍 `remote: HTTP Basic: Access denied`（`data-traceId` `829ad513-b606-4281-b2b3-10796302fbad`）。

## 根因

1. 用户同时绑定 `gitlab:tencent-sh-1` 与 `gitlab:daydaymoney-gitlab`。
2. Cloud `loadGitOauthProviders` 未用 `confload.ResolveTemplate` 展开 `${scheme}://${subdomains.gitlabTencentSh1}`，主机匹配失败。
3. `resolveProviderKeyFromRepoURL` 把任意含 `gitlab` 的主机塌缩为 `gitlab:default`。
4. `gitlabProviderKeysToTry` 再按 YAML 文件名顺序尝试全部 GitLab provider；`http-gitlab-daydaymoney-com.yaml` 排在 tencent-sh-1 之前，先换到 **另一套 CE** 的 token。
5. 容器拿 daydaymoney token 去推 `gitlab-tencent-sh-1.daydaymoney.com` → HTTP Basic Access denied。

## 解决方案

1. 加载 provider YAML 时展开 `base.yaml` 寻址模板（与 taskCredentialService 一致）。
2. 主机命中区域实例后 **只** 换该 `provider_key`，禁止借其它 CE token（ADR-0014）。
3. 未匹配主机仍可回退尝试已配置的 GitLab provider（兼容旧 alias）。

## 验证

```bash
cd taskCloudService && go test ./src -count=1 -run 'TencentSh1|GitlabProviderKeysToTry'
# 公网同一任务再点「提交并创建PR」：GitLab 出现 feature 分支 + MR
```

## 关联

- `.ai/09_failure_experience/02_runtime_errors/107_ztree_push_gitlab_oauth_missing_write_repository.md`
- `conf/auth/git-oauth/ai.md`「多区域 GitLab」
- `taskCloudService/src/git_oauth_provider_config.go`
- `taskCloudService/src/git_push_oauth.go`
