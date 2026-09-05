# 测试意图：GitLab 镜像区域开发/发布模式

- **对应功能意图**: `system_admin_gitlab_region_access_mode.intent.md`
- **日期**: 2026-08-23

## 测试目标

验证模式字段、管理 UI、目录过滤与写路径门禁。

## 测试分层

- 前端 Vitest：卡片徽章、编辑/新建下拉。
- 后端 Go：默认 release、过滤、403、事件。

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| F1 | 区域卡 | 展示开发或发布 |
| F2 | 编辑保存 development | PUT 含 access_mode=development |
| B1 | 新列默认 | access_mode=release |
| B2 | 非测试 list | 不含 development |
| B3 | 测试 list | 含 development |
| B4 | 非测试购买 development | 403 |
| B5 | 测试购买 development | 不因模式拒绝 |
| B6 | 改模式 | 发布 GitlabRegionAccessModeChanged |
| B7 | CanUseGitlabRegion | release+任意 true；development+tester true；development+非测试 false |

## 通过标准

上述用例全绿。
