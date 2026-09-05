# 测试意图：租户级自建 GitLab OAuth 连接

## 用例

1. **管理员 CRUD**：PUT 创建 → GET 脱敏含 redirect_uri → PUT 更新 → DELETE 204  
2. **唯一约束**：同 company 二次 PUT 为更新非第二行  
3. **权限**：成员 GET 200；成员 PUT 403  
4. **Resolve**：`service_provider=tenant-{id}` 返回该行 client_id/base_url  
5. **级联**：绑定一行 credential 后 DELETE，credential 消失  
6. **Catalog**：带 company_id 时 providers 含 `gitlab:tenant-{id}`  
7. **前端**：侧栏「GitLab 连接」可达；表单可提交（mock API）  
8. **Redirect URI 租户隔离**：GET/PUT 返回的 redirect_uri 含 `tenant-{company_id}`；不同 company 互不相同；遗留共享 `tenant-gitlab` URI 在 resolve 时改写为租户级  

## 对应测试文件（计划）

- `taskGitOauth/src/tenant_connection_test.go`
- `taskGitOauth/domain/tenant_connection_redirect_test.go`
- `taskGitOauth/infrastructure/tenant_redirect_uri_test.go`
- `taskFE/app/src/views/WorkspaceSettingsGitlabConnection.test.js`
- 可选 Playwright：`WorkspaceSettingsGitlabConnection.*.playwright.test.js`
