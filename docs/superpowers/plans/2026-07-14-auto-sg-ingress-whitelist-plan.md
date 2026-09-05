# 实施计划 — 自动 SG 入网白名单

## Task 1: 规则构造与撤销（TDD）

- [ ] 改写 `defaultAutoSGIngressRules` → 按 CIDR 列表生成；断言无 `0.0.0.0/0`
- [ ] 实现 `revokeIngress` / `revokeFullOpenIngress`
- [ ] 改 `ensureFullOpenIngress` → `ensureWhitelistIngress`
- [ ] 更新 `defaultSGName` 与单元测试

## Task 2: taskCloudService 透传 client IP

- [ ] 新增 `resolveClientIP`
- [ ] `handleStartVmAutoNative` 注入 body
- [ ] `buildStartVmEventData` 写入 `client_public_ip` + `auto_sg_whitelist`
- [ ] 单测

## Task 3: startauto / started 编排

- [ ] `AutoNetworkInput.ClientPublicIP`
- [ ] startauto 透传字段到 nextData
- [ ] started 成功后 Phase B（DescribeInstances + authorize）
- [ ] 单测（mock）

## Task 4: 前端与文档

- [ ] `ServerConfigHardwarePanel.vue` 提示文案
- [ ] `DOMAIN_EVENTS.md` + 意图文档
- [ ] OpenAPI 可选字段说明

## Task 5: 验证

- [ ] `go test` taskEvents aliyun + handlers
- [ ] `go test` taskCloudService 相关
