# 实施计划 — 导航栏代码仓库区域跳转

## 切片

- [x] **S1** Go：`listTenantGitlabNavTargets` + 单测（空 / 两区赠送）
- [x] **S2** HTTP：GET 无 region 把 `resources` 挂到 `gitlabResourceView`；更新 convention 测例
- [x] **S3** OpenAPI：`GitlabResourcesView.resources`
- [x] **S4** 先写 Navbar.ui 测例（红）再实现下拉/当前页/去 VIP 门禁
- [x] **S5** Navbar.logic 拉取 gitlab-resources 透传 `gitResources`
- [x] **S6** 行数：抽出 `NavbarGitServiceNav.vue` 若 Navbar.ui 将超 500
- [x] **S7** gofmt/vet、vitest、注册精准重启

无新事件契约。
