# 测试意图：项目详情内部仓库磁盘占用

## 用例

| ID | 场景 | 期望 |
|----|------|------|
| T1 | `git_repo_entries` 含 `is_internal=true` 且 `disk_size_bytes=1288490188` | 渲染磁盘徽章，文案含可读大小 |
| T2 | `is_internal=false` 且无 `disk_size_bytes` | 无 `git-repo-disk-size` 节点 |
| T3 | Go：gitlab-local URL | `IsInternalRepo` true |
| T4 | Go：github.com URL | `IsInternalRepo` false；enrich 不发起 HTTP |
| T5 | Go：mock statistics JSON | enrich 写入 `disk_size_bytes` |

## 实现映射

- 前端：`ProjectDetailGitReposSection.disk-size.test.js`
- 后端：`repo_disk_size_test.go` / `provider_resolver` 内部仓断言
