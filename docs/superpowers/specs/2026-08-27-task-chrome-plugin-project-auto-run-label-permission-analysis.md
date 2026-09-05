# 权限分析：Chrome 插件项目列表自动运行标注

- **Date:** 2026-08-27
- **Design:** `docs/superpowers/specs/2026-08-27-task-chrome-plugin-project-auto-run-label-design.md`

## 结论

无新端点、无新角色。只读展示调用方已有权看到的项目列表字段。

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET 工作空间项目列表（既有 `getProjects`） | 已登录扩展用户 | Workspace | read | 既有 token + workspace 归属 | ✅ 充分 | 不改鉴权 |
| 渲染 `default_auto_run` 徽章 | 同上 | Project | read（展示） | 仅渲染本次 GET 返回体 | ✅ 充分 | 禁止把未转义名称写入 HTML |
| 创建任务 `auto_run` | 同上 | Task | write | 既有 Git 身份 / OAuth / 后端 AUTO_RUN_PROJECT_NOT_ALLOWED | ✅ 充分 | 本增量不改写路径 |

## 建模

不引入新角色。无 IDOR 新增面：不按项目 id 另拉敏感字段。
