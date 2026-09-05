# 权限分析：Chrome 插件自动运行与已安装镜像挂钩

- **Date:** 2026-08-27
- **Design:** `docs/superpowers/specs/2026-08-27-task-chrome-plugin-auto-run-image-gate-design.md`

## 结论

无新 HTTP 端点、无新角色、无新数据访问。沿用既有：已登录用户读本租户已安装镜像、创建本工作空间任务。客户端门禁不扩大权限面。

## 改动点

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `/api/cloud/installed-images/tenant_id/{companyId}`（既有） | 已登录租户成员 | Tenant | read | 插件 `swApi` + 网关鉴权 | ✅ 充分 | 不改 |
| POST 创建任务（既有） | 已登录且有工作空间写权限 | Workspace | write | 网关 + taskTaskService 校验 | ✅ 充分 | 客户端提前镜像门禁，服务端 `AUTO_RUN_IMAGE_REQUIRED` 仍保留 |
| `resolveAutoRunControlState` / `validateCreateTaskForm` | 操作者本机 | — | UI | 无越权面 | ✅ | 纯 DOM 状态 |

## IDOR / 越权

镜像下拉仍只渲染当前工作空间所属 `companyId` 的已安装列表；不按用户输入拼接其它租户 ID。

## 新角色

无。

## python_api_approval

not_applicable
