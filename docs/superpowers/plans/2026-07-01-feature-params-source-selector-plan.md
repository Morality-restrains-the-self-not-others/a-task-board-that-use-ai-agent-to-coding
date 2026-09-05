# 实施计划: 容器启动时功能参数来源选择

> 输入:
> - 设计: `docs/designs/env-var-preset-switching.md`
> - 价值流: `docs/superpowers/plans/2026-07-01-feature-params-source-selector-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-07-01-feature-params-source-selector-nfr-clarification.md`
> - DDD: `docs/superpowers/plans/2026-07-01-feature-params-source-selector-ddd.md`

---

## Increment 1: 后端预览端点 + 启动参数扩展

### Task 1.1: 新增预览端点 `feature-params-env-preview`

**文件**: `cloud/views/cloud_compute_views.py`
**测试**: `cloud/view_test/test_feature_params_env_preview.py` (TDD first)

- [ ] **1.1a** 写测试 `test_feature_params_env_preview.py` (TDD RED)
  - 测试 1: `test_preview_personal_config_returns_200` — 用户预览自己的个人配置返回 200 + env
  - 测试 2: `test_preview_other_user_config_returns_403` — 跨用户 IDOR 防护
  - 测试 3: `test_preview_nonexistent_config_returns_404` — 不存在的配置
  - 测试 4: `test_preview_company_source_returns_403` — `?source=company` 被拒绝
  - 测试 5: `test_preview_workspace_source_returns_403` — `?source=workspace` 被拒绝
- [ ] **1.1b** 实现 `get_feature_params_env_preview` action (~25 行)
  - 解析 `source` query param
  - `source != 'personal'` → 403
  - 读取 `personal_config_id`，查 `PersonalFeatureParamsConfig`
  - 不存在 → 404；`config.user_id != request.user.id` → 403 (warn 日志)
  - 实例化 `FeatureParamsResolver` + `FeatureParamsEnvSerializer`
  - 调用 resolver → serializer → 返回 JSON `{source, config_name, env}`
- [ ] **1.1c** 注册路由 `cloud/urls.py` (+2 行)
  - `url_path='feature-params-env-preview'`, `methods=['get']`
- [ ] **1.1d** 运行测试 → GREEN

### Task 1.2: 修改启动端点接受 source 参数

**文件**: `cloud/views/cloud_compute_views.py` + `cloud/services/mock_run_container.py`
**测试**: 扩展现有 `test_relay_to_trae_start.py` (或新增测试函数)

- [ ] **1.2a** 写测试
  - 测试 1: `test_start_with_personal_source_persists_todo` — 启动时传 source=personal + config_id，验证 Todo 写入
  - 测试 2: `test_start_with_personal_source_other_user_config_rejected` — 传他人 config → 403
  - 测试 3: `test_start_without_source_defaults_to_company` — 不传 source，验证向后兼容
- [ ] **1.2b** 修改 `relay_to_trae_start` 服务函数 (+5 行)
  - 从 request body 读取 `feature_params_source` (optional, default None)
  - `source == 'personal'` 时: 校验 config 归属 (复用 `_check_idor` 模式)
  - 持久化到 `Todo.feature_params_source` + `Todo.personal_feature_params_config_id`
  - 添加 info 日志: `task_id + source + config_id`
- [ ] **1.2c** 运行测试 → GREEN

---

## Increment 2: 前端来源选择器 + 预览集成

### Task 2.1: 扩展 ServerConfig.logic.vue

**文件**: `components/ServerConfig.logic.vue`
**测试**: Playwright E2E (Task 2.3)

- [ ] **2.1a** 新增状态变量 (~10 行)
  ```javascript
  const featureParamsSource = ref('company')
  const personalConfigs = ref([])
  const selectedPersonalConfigId = ref('')
  const resolvedEnvPreview = ref({})
  const isEnvPreviewLoading = ref(false)
  const envPreviewExpanded = ref(false)
  ```
- [ ] **2.1b** 新增方法 (~40 行)
  - `initFeatureParamsSource()` — 从 `props.task.feature_params_source` 读取当前值
  - `fetchPersonalConfigs()` — `GET /api/personal/feature-params-configs/` 填充二级下拉
  - `fetchEnvPreview()` — `GET .../feature-params-env-preview/?source=personal&personal_config_id=...`
  - `onSourceChange()` — source=personal 时拉取个人配置列表；清空预览
  - 修改 `startRelayToTrae()` — 传递 `feature_params_source` + `personal_feature_params_config_id`
- [ ] **2.1c** 在 `onMounted` / `watch(task)` 中调用 `initFeatureParamsSource()`

### Task 2.2: 修改 ServerConfigRelayDirectPanel.vue

**文件**: `components/ServerConfigRelayDirectPanel.vue`

- [ ] **2.2a** 新增来源选择器 UI (~30 行)
  - 三选一下拉: `公司默认` / `工作空间默认` / `个人配置`
  - 绑定 `featureParamsSource`，`@change="onSourceChange"`
- [ ] **2.2b** 新增个人配置二级选择器 (~10 行)
  - `v-if="featureParamsSource === 'personal'"`
  - 下拉选项来自 `personalConfigs`
  - 空状态: "暂无个人配置，请先在设置页创建"
- [ ] **2.2c** 新增预览按钮 + 折叠预览面板 (~20 行)
  - `v-if="featureParamsSource === 'personal'"` 显示 [预览] 按钮
  - 点击 → `fetchEnvPreview()` → 展开折叠面板
  - 面板展示 `v-for="(value, key) in resolvedEnvPreview"` 键值对
  - 敏感 key（含 token/secret/key）遮罩处理

### Task 2.3: E2E 测试

**文件**: `playwright/front_project/tests/TaskDetail.feature-params-source-switching.playwright.test.js`

- [ ] **2.3a** 测试: 默认显示公司级来源
- [ ] **2.3b** 测试: 切换到工作空间来源 → 无预览按钮
- [ ] **2.3c** 测试: 切换到个人配置 → 出现二级下拉 + 预览按钮
- [ ] **2.3d** 测试: 选择个人配置 → 点击预览 → 看到 env 变量
- [ ] **2.3e** 测试: 点击启动 → 验证 Todo.feature_params_source 持久化 (检查 API 响应)
- [ ] **2.3f** 测试: 不选来源 → 启动成功 (向后兼容)

---

## 依赖关系

```
Task 1.1 (预览端点) ──→ Task 1.2 (启动端点修改)
         │
         └──→ Task 2.1 (logic 扩展) ──→ Task 2.2 (panel UI) ──→ Task 2.3 (E2E)
```

---

## 自检

- [ ] 每个 task 有明确的文件路径
- [ ] 每个 task 有对应的测试文件 (TDD 先行)
- [ ] 后端: 5 个测试场景覆盖权限 + 正常流程
- [ ] 前端: 6 个 E2E 场景覆盖 UI + 兼容性
- [ ] Task 依赖关系清晰 (后端先于前端)
