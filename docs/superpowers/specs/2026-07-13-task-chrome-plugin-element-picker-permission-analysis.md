# 权限分析：悬浮面板指针选择页面元素

**日期**: 2026-07-13  
**关联设计**: `2026-07-13-task-chrome-plugin-element-picker-design.md`

## 结论

无新增 HTTP/RPC 端点；不改变租户/工作空间 ACL。创建任务仍走既有 `createTask` + Session Token。

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| 指针选 DOM 元素 | 已安装扩展的浏览器用户 | 当前页 DOM（只读） | read | content_scripts `<all_urls>` | ✅ 充分 | — |
| 写入浮窗任务描述 | 同上 | 本地 UI 状态 | write | 无服务端 | ✅ N/A | — |
| 提交创建任务 | 已登录用户 | Workspace Todo | write | Token + 既有 API | ✅ 充分 | 不变 |

## 敏感数据处理

- 不对 `input[type=password]` / 疑似密码框读取 `value`
- 可见文本与 outerHTML 截断，避免过大 payload

## 角色建模

无需新角色。
