# 前端：系统管理 GitLab 区域删除须确认仓库地址

> 状态: **implemented** | 日期: 2026-08-18

## 意图

在 `/system-admin/gitlab-resources` 区域卡片点击「删除」时，须弹出自定义确认框，明确展示将要停用的 **仓库地址**（`gitlab_web_url`，缺失则回退 `gitlab_api_base`），管理员确认后再调用停用 API。

## 验收

1. 点击「删除」出现确认弹层（非浏览器 `confirm`）
2. 弹层显著展示仓库地址（`data-testid=gitlab-region-delete-repo-address`）
3. 「取消」关闭弹层且不发 DELETE
4. 「确认停用」才请求 `DELETE /api/system-admin/gitlab-regions/{slug}/`
