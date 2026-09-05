# 权限分析：项目文件树与变动列表 Tab

- **日期**: 2026-08-22
- **设计**: `docs/superpowers/specs/2026-08-22-file-tree-changes-tab-design.md`

## 结论

无新 endpoint、无新数据访问。Tab 仅为任务详情既有面板的展示切换。沿用任务详情页已有鉴权（登录 + 租户/工作区任务读权限 + 容器转发 comment_id）。

## 改动点

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| Tab 切换（本地 state） | 能打开该任务详情的用户 | Task | read（UI） | 页面级会话 | ✅ | 无服务端动作 |
| 展示 `TaskDetailProjectFileTree` | 同上 | Task / Layer | read | 既有 `container-layer-children` 转发 | ✅ | 不改 |
| 展示 `TaskDetailExecLayerChanges` | 同上 | Task / Layer | read | 既有执行日志 / diff 转发 | ✅ | 不改 |
| 变动 Tab 内提交/暂存 | 同上 | Task / Layer | write | 既有 git 身份门闩 + staged actions blocked | ✅ | 不改写路径 |

## 角色建模

不引入新角色或权限粒度。

## IDOR

Tab 不接受用户输入的 layerId；仍由当前选中 zTree 节点解析。无新 ID 参数。
