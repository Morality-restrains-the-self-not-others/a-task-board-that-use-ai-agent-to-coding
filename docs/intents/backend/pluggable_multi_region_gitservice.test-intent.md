# 测试意图：可插拔多区域 gitService

## 覆盖

| 场景 | 测试 | 期望 |
|------|------|------|
| 缺 region | `taskBill/src/gitlab_region_routing_test.go` `TestRequireRegionSlug_EmptyRejected` | error 含 region |
| 按区域 Admin API | `TestEnsureTenantGitlabGroupForRegion_UsesRegionBaseAndToken` | 请求打到 region api_base 且带 PAT |
| 购买 body 缺 region | `handleGitlabResourcesPurchase` | HTTP 400 |
| 前端不默认区域 | `useGitlabResourcePurchase.test.js` | `region === ''`；purchase 无 region 不发请求 |
| 设置页空态选区域 | `WorkspaceSettingsGitlabConnection.test.js` | `gitlab-region-picker` 可见 |
| 设置页详情在下拉之下 | `WorkspaceSettingsGitlabConnection.test.js` | 开通后 picker 仍在；详情位于 select 之后；切换选项更新详情 |
| migrate 已记录 010 后追加 bootstrap client | `taskAuth/src/data_migrate_go_test.go` `TestRunGoDataMigrateReseedsOidcClientsWhenStepAlreadyLogged` | 测试用 `appended-after-010` INSERT 成功；`admin-locked-client` redirect 不被覆盖；checksum 刷新 |

## 命令

```bash
cd taskBill && go test ./src -count=1 -run 'Region|GitlabResource'
cd taskFE/app && npx vitest run src/composables/useGitlabResourcePurchase.test.js src/views/WorkspaceSettingsGitlabConnection.test.js
cd taskAuth && go test ./src -count=1 -run 'TestRunGoDataMigrateReseedsOidcClientsWhenStepAlreadyLogged|TestOidcBootstrapClientChecksumIgnoresSecret'
```
