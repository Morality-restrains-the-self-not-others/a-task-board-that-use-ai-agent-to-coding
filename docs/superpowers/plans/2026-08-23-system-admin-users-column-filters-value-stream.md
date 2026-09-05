# 价值流：系统管理用户表列过滤器

```
超管打开 /system-admin/users/
  → 表头下方看到每列过滤器
  → 输入/选择某一列条件
  → 前端防抖后 GET 带列参数 + is_archived + 可选 q
  → taskAuth SQL 过滤本库字段
  → （若填了公司/推荐人/分账）批量 hydration 后再筛
  → 分页列表只含命中用户
  → 点重置清空列条件并重新拉第一页
```

## 测试点

| ID | 步骤 | 用例 |
|----|------|------|
| VS-UF-1 | 渲染 | thead 第二行每列有过滤器，操作列为重置 |
| VS-UF-2 | 邮箱 contains | 请求带 `email=`，只返回命中用户 |
| VS-UF-3 | 角色下拉 | 请求带 `role=superuser` |
| VS-UF-4 | 注册日期 | 请求带 `date_joined_from` |
| VS-UF-5 | 登录方式 | 请求带 `login_method=phone` |
| VS-UF-6 | 状态 | 请求带 `is_active=false` |
| VS-UF-7 | 分账资格 | 请求带 `has_profit_sharing=true` |
| VS-UF-8 | 重置 | 列参数清空，重新请求 |
| VS-UF-9 | 非超管 | 403 |
| VS-UF-10 | 与 q AND | 全局搜索 + 列过滤同时生效 |
