# 测试意图：系统管理 GitLab 区域删除须确认仓库地址

| 场景 | 测试 | 期望 |
|------|------|------|
| 地址解析优先 web | `SystemAdminGitlabRegionDeleteModal.test.js` resolveGitlabRegionRepoAddress | 返回 gitlab_web_url |
| 弹层展示地址 | 同上 + `SystemAdminGitlabResources.delete-confirm.test.js` | `gitlab-region-delete-repo-address` 含 URL |
| 确认才 DELETE | `SystemAdminGitlabResources.delete-confirm.test.js` | 确认后 method=DELETE |
| 取消不 DELETE | 同上 | 无 DELETE 调用 |
