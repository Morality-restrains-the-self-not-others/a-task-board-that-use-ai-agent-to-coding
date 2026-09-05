# Plan — GitLab 同步已购区域选择

## 切片

- [x] 设计 + 架构 v130 application-integration
- [x] `normalizePurchasedGitlabRegions` + 单测
- [x] `useGitlabProjectSync`：先拉 resources，再带 `gitlab_host` 拉仓；`selectRegion`
- [x] `GitlabSyncProjectsModal`：下拉 / 单 URL / 空态
- [x] `Projects.vue` 接线
- [x] Vitest + Playwright mock 更新
- [ ] Ship：提交 main、精准重启登记、VERSION_HISTORY current

## 事件

纯查询例外已在设计文档记录；无新 MQ 意图。
