# 实施计划：测试角色 + GitLab 区域开发/发布模式

- **日期**: 2026-08-23
- **设计/权限/价值流/NFR/DDD**: 同主题 specs/plans

## 任务

### T1 领域纯函数（taskBill + 可被 FE 镜像的规则文档）

- [ ] `CanUseGitlabRegion` 单测：release 任意 true；development 仅 tester
- [ ] 实现函数

### T2 taskAuth DDL + 过滤 + PATCH + JSON + 事件

- [ ] `dataMigrate/taskAuth/039_user_is_tester.sql`
- [ ] 列表 role=tester / tenant 排除 tester
- [ ] PATCH/创建 `is_tester`；true 时 is_tenant=1
- [ ] 用户 JSON /me 含 is_tester
- [ ] `UserTesterFlagChanged` 映射 + publish
- [ ] 单测

### T3 forward-auth `X-User-Is-Tester`

- [ ] load 标志；写头；缓存字段
- [ ] 单测：tester 为 1，roles 不含 tester

### T4 taskFE 用户列表

- [ ] 筛选项「测试」
- [ ] 徽章优先测试
- [ ] 新增/编辑勾选
- [ ] Vitest

### T5 taskBill DDL + 区域 CRUD access_mode + 事件

- [ ] `dataMigrate/taskBill/065_gitlab_region_access_mode.sql`
- [ ] struct/scan/create/update
- [ ] `GitlabRegionAccessModeChanged`
- [ ] 单测

### T6 租户目录与购买门禁

- [ ] listGitlabRegions(isTester)
- [ ] purchase/provision 403
- [ ] 已购摘要过滤 development（非测试）
- [ ] 单测

### T7 taskFE 区域管理 UI

- [ ] 卡片徽章、编辑/新建下拉
- [ ] Vitest

### T8 价值流图测试点

- [ ] 更新 `docs/flows/value-stream-test-integration.wsd`

### T9 事件契约

- [ ] taskAuth eventTopicMap
- [ ] taskBill billingEventTopics
- [ ] 更新 intents 对照表（已写）

每切片：Red → Green → 提交（子仓优先）。
