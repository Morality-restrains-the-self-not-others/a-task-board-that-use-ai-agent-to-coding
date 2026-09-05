# 权限分析：ztree step_full COS 归档

- **Date:** 2026-08-23
- **Design:** `docs/superpowers/specs/2026-08-23-ztree-step-full-cos-archive-design.md`

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| POST `server-container-token/job-step-full-push` | 评论容器（access_token） | Workspace/Task/Comment | write | validateContainerAccessToken + path 与 scope 匹配 | ✅ | 拒绝跨任务 comment |
| GET `container-job-execution-log` | 租户成员 / accessCode | Workspace/Task | read | 既有 compute 鉴权 + workspace/task 必填 | ✅ | COS hydrate 不绕过 |
| GET/PATCH `/api/system-admin/step-full-cos/` | 平台员工 | System | read/write | `authz.IsPlatformStaff` | ✅ 新增 | 密钥不回显 |
| 表 `cloud_job_step_full_object` | 仅 taskCloudService | 库 task_cloud | C/U/R | 单服务所有权 | ✅ | 他服务禁止直连 |
| COS object | Cloud SDK | 对象 key 含 ws/task/comment | R/W | pathRule 白名单 | ✅ | 禁止 `..` |

不新增角色。无跨租户读：GET 必须带 workspace_id+task_id+comment_id。
