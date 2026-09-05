# 实施计划: 修复预览环境变量按钮无响应

> 输入: `docs/designs/fix-env-preview-button-no-response.md`

## 任务清单

### Task 1: 修复 URL 路径构造 ✅ 核心修复

- [ ] **文件**: `front_project/app/src/components/ServerConfig.logic.vue`
- [ ] **位置**: `fetchEnvPreview()` 方法 (~L489-496)
- [ ] **变更**: 绕过 `relayToTraeApiUrl()`，直接用 `ctx` 拼接 URL
- [ ] **验证**: 点击「预览环境变量」→ 显示环境变量预览面板

### Task 2: 增加失败反馈

- [ ] **位置**: 同上方法，`if (resp.ok)` 之后
- [ ] **变更**: 增加 `else` 分支打印 `console.error` 含 status code 和 message
- [ ] **验证**: API 失败时控制台有错误日志

### Task 3: 手动验证

- [ ] 选择个人配置 → 点击预览环境变量 → 预览面板显示
- [ ] 预览内容包含预期的环境变量键值对
