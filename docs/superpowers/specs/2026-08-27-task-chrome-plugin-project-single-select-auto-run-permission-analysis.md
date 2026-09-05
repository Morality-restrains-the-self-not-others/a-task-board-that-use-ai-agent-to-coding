# 角色权限分析：Chrome 插件项目单选 + 自动运行随项目能力

- **Date:** 2026-08-27
- **Design:** `docs/superpowers/specs/2026-08-27-task-chrome-plugin-project-single-select-auto-run-design.md`

## 结论

无新角色、无新 endpoint、无新数据访问。项目列表仍走既有 `getProjects`（工作空间成员可读）；创建任务仍走既有 POST。客户端禁用自动运行是 UX 钳制，不替代服务端 `validateAutoRunPrerequisites`。

## 改动点权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| 浮窗/面板项目 radio 渲染 | 已登录扩展用户 | Workspace | read UI | 既有 token + getProjects | ✅ 充分 | — |
| 自动运行 checkbox enable/disable | 已登录扩展用户 | Workspace/Project | 本地 UI | 只读 `default_auto_run` | ✅ 充分 | 不允许时强制 uncheck，避免发 `auto_run:true` |
| 创建任务 payload（0 或 1 个 project） | 已登录扩展用户 | Workspace | write | 既有 createTask + 后端校验 | ✅ 充分 | 不放宽、不新开写路径 |

## 建模

不引入新角色/权限粒度。
