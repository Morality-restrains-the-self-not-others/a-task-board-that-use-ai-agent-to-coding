# Value Stream: GitHub OAuth 测例迁移至 gitOauth

> 设计：`docs/superpowers/specs/2026-05-29-github-oauth-tests-migration-to-gitoauth-design.md`

## Value Summary

开发者与价值流工具能在 **gitOauth** 服务内验证 GitHub OAuth 内部 API 与凭据字段契约，task2app 不再用 mock 冒充 git-oauth 真源。

## Related Value Streams

- **gitoauth-binding-state-persistence**：修改 `bind-state-summary-contract-thin-slice` 的 `test_file` 指向 gitOauth
- **cloud-credential-summary-string-contract**（云平台域）：同上

## End-to-End Flow

价值流校验触发 → pytest 加载 `gitOauth/api/tests.py` → 命中 internal summary/user-ids HTTP → 断言 `GithubAppUserCredential` 字段与 JSON 契约

## Value Increments

### Increment 1: gitOauth 契约测例（Thin Slice）
**Value to user:** 单仓内可证明 summary/user-ids 与 bind_status 字段正确  
**Scope:** 扩展 `GithubOAuthCredentialSummaryForUserViewTests` + user-ids provider_key 过滤；删除 task2app 两个 fetch_* 文件  
**Depends on:** 无

### Increment 2: 价值流对齐
**Value to user:** `value-stream.yaml` 中 git-oauth 字段与测例路径一致  
**Scope:** 更新两处 `test_file` 为 `../../gitOauth/api/tests.py`  
**Depends on:** Increment 1
