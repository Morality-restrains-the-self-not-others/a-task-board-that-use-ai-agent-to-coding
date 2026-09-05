# OAuth 回调失败 Toast — 设置页 skip 回归

## 路径（均应 skip Toast，仅内联错误）

- `/profile/git-site-oauth/`
- `/tenant/:tenant/profile/git-site-oauth/`
- `/user/:id/profile/git-site-oauth/`

## 验收

访问 `?gitlab=bad_state` 时页面内联显示错误，**不**出现全局 Toast。
