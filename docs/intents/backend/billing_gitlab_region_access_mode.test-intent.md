# 测试意图：GitLab 区域 access_mode 门禁（后端）

- **对应功能意图**: `billing_gitlab_region_access_mode.intent.md`
- **日期**: 2026-08-23

## 用例

| ID | 场景 | 期望 |
|----|------|------|
| B1 | 默认 | release |
| B2 | 非测试目录 | 无 development |
| B3 | 测试目录 | 有 development |
| B4 | 非测试购买 development | 403 |
| B5 | 更新模式 | 事件 GitlabRegionAccessModeChanged |

## 通过标准

Go 单测全绿。
