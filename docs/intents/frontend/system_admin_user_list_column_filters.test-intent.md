# 测试意图：用户列表表头列过滤器

| ID | 场景 | 期望 |
|----|------|------|
| F1 | 挂载活跃用户 tab | 存在 `user-list-column-filters`，列数与表头一致 |
| F2 | 输入邮箱过滤 | 后续请求 URL 含 `email=` |
| F3 | 选择角色=超管 | URL 含 `role=superuser` |
| F4 | 选择登录方式=手机 | URL 含 `login_method=phone` |
| F5 | 填写注册开始日期 | URL 含 `date_joined_from=` |
| F6 | 选择状态=已禁用 | URL 含 `is_active=false` |
| F7 | 选择分账=是 | URL 含 `has_profit_sharing=true` |
| F8 | 点击重置 | 列参数从 URL 消失，仍带 `is_archived=` |
| F9 | 推荐码申请 tab | 无过滤行 |
