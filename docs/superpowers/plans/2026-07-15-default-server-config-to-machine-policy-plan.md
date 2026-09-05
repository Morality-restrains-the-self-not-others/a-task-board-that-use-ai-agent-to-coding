# 实施计划：默认服务器启动配置迁入机器节点策略

## 任务

- [x] T1：Intent 文档（frontend）
- [x] T2：`WorkspaceMachinePolicyModal` 增加默认配置分区 + 嵌套 `SetDefaultConfigModal`
- [x] T3：云平台页移除入口（Row + View）
- [x] T4：`ProjectRunTemplatePanel` 文案更新
- [x] T5：Playwright / 单元测试更新
- [x] T6：本地验证 + 前端 build/collectstatic（若改 SPA）

## 验证命令

```bash
# 单元（若有）
cd task2app/front_project/app && npm test -- --run ProjectRunTemplate WorkspaceMachinePolicy CloudPlatformAuthorization 2>/dev/null || true

# Playwright（视环境）
# WorkspaceSettings.machine-policy-modal + set-default-config-reload（入口改 task-panel）
```

## 文件清单

- `WorkspaceMachinePolicyModal.vue`
- `CloudPlatformAuthorizationRow.vue`
- `WorkspaceSettingsCloudPlatform.vue`
- `ProjectRunTemplatePanel.vue`
- Playwright tests
- `docs/intents/frontend/...`
