# 价值流：租户级自建 GitLab OAuth 连接

**日期：** 2026-07-15  
**设计：** `2026-07-15-tenant-gitlab-oauth-connection-design.md`

## Related Value Streams

- 既有：用户 Git 网站授权 / 多 provider YAML（`git-site-oauth`、gitoauth 相关）  
- 本流为**扩展**：在绑定前增加「租户配置自建 GitLab App」前置步骤；不替换平台 YAML providers。

## Value Stream

**名称：** `tenant-gitlab-oauth-connection`  
**域：** 组织与成员 / Git OAuth

| # | Step | 状态 | 测试 | Fields |
|---|------|------|------|--------|
| 1 | 管理员打开 GitLab 连接设置页 | active | Playwright 菜单可达 | — |
| 2 | 管理员 PUT 保存 base_url+client | active | Go TestTenantGitlabConnectionPut | `git-oauth.tenant_gitlab_oauth_connections.company_id` 等 |
| 3 | GET 返回 redirect_uri（脱敏） | active | Go TestTenantGitlabConnectionGet | `redirect_uri` |
| 4 | providers 合并租户条目 | active | Django/API 或 Go+前端测 | providers[].provider_key |
| 5 | 成员 OAuth 绑定 tenant provider | planned | E2E（依赖可达自建 GitLab） | `api_gitoauthappusercredential.provider` |
| 6 | 管理员 DELETE 级联解绑 | active | Go TestTenantGitlabConnectionDelete | credential 行数 |

## Increments（交付切片）

1. **Inc1（MVP）：** 表 + Go CRUD + 单测 + 设置页 UI + redirect_uri 展示  
2. **Inc2：** Resolve 接入 start/callback + providers 合并  
3. **Inc3：** DELETE 级联 + Kafka 事件 + Playwright 冒烟  

## YAML 摘录（写入 conf/value-stream 或计划附录）

```yaml
- name: tenant-gitlab-oauth-connection
  description: 租户配置自建 GitLab OAuth 并供成员绑定
  steps:
    - name: admin-upsert-connection
      test_file: taskGitOauth/src/tenant_connection_test.go
      fields:
        - name: git-oauth.tenant_gitlab_oauth_connections.company_id
        - name: git-oauth.tenant_gitlab_oauth_connections.base_url
        - name: git-oauth.tenant_gitlab_oauth_connections.client_id
    - name: member-see-provider
      test_file: task2app/front_project/.../UserGitSiteOAuthSettings...
      fields:
        - name: git-oauth.tenant_gitlab_oauth_connections.active
```
