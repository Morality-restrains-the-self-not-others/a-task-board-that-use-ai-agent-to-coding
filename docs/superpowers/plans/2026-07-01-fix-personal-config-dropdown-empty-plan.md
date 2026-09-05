# 实施计划: 修复个人配置下拉为空

> 输入:
> - 设计文档: `docs/designs/fix-personal-config-dropdown-empty.md`
> - 价值流: `docs/superpowers/plans/2026-07-01-fix-personal-config-dropdown-empty-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-07-01-fix-personal-config-dropdown-empty-nfr-clarification.md` (skipped)
> - DDD: `docs/superpowers/plans/2026-07-01-fix-personal-config-dropdown-empty-ddd.md` (skipped)

## 任务清单

### Task 1: 修复 API 响应字段名解析错误 ✅ 核心修复

- [ ] **文件**: `task2app/front_project/app/src/components/ServerConfig.logic.vue`
- [ ] **位置**: L475
- [ ] **变更**:
  ```diff
  - personalConfigs.value = Array.isArray(data) ? data : (data.results || [])
  + personalConfigs.value = data.configs || []
  ```
- [ ] **验证**: 选择「个人配置」后下拉显示已创建的配置列表

### Task 2: 补充初始加载时自动获取个人配置

- [ ] **文件**: `task2app/front_project/app/src/components/ServerConfig.logic.vue`
- [ ] **位置**: L458-465 (`initFeatureParamsSource` 方法末尾)
- [ ] **变更**:
  ```diff
   const initFeatureParamsSource = () => {
     if (props.task?.feature_params_source) {
       featureParamsSource.value = props.task.feature_params_source
     }
     if (props.task?.personal_feature_params_config_id) {
       selectedPersonalConfigId.value = props.task.personal_feature_params_config_id
     }
   + if (featureParamsSource.value === 'personal') {
   +   fetchPersonalConfigs()
   + }
   }
  ```
- [ ] **验证**: 刷新已绑定 `source=personal` 的任务详情页，下拉自动填充配置列表

### Task 3: 手动验证

- [ ] 登录 `contact@daydaymoney.com`
- [ ] 确认用户在 `/user/{uid}/profile/feature-params/` 至少有一个个人配置
- [ ] 进入任务详情页 → 直接启动 → 功能参数来源 → 选择「个人配置」
- [ ] 验证二级下拉显示所有个人配置
- [ ] 选择一个配置 → 点击「预览环境变量」→ 验证返回正确 env
- [ ] 刷新页面 → 验证配置选择保持且下拉正常

## 影响范围

| 维度 | 详情 |
|------|------|
| 修改文件 | 1 个 (`ServerConfig.logic.vue`) |
| 修改行数 | 2 行 |
| 后端变更 | 无 |
| 测试变更 | 无（已有 E2E 覆盖此流程，但因 Bug 测试可能未覆盖到此边界） |
| 风险等级 | 极低 |

## 依赖

- 无。本修复独立于其他任何变更。
