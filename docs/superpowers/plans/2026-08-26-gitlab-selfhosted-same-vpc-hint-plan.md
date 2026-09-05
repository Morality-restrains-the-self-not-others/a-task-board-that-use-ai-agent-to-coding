# 实施计划：自建 GitLab 同专有网络提示

- **日期**: 2026-08-26

## Task 1 — 纯函数（红→绿）

- [x] `taskFE/app/src/utils/defaultMachineNetworkHint.js` + `.test.js`
- 覆盖：空列表 / 有配置 / 优先带 vpc 的条目
- 验证：vitest 该文件

## Task 2 — 提示组件 + 弹窗

- [x] `GitlabSelfHostedSameVpcHint.vue` + 单测
- [x] 复用 CreateVpcModal / CreateVswitchModal
- [x] CreateVpcModal `created` 透传 `vpc_id`（兼容无参监听）
- 验证：vitest hint 组件

## Task 3 — 接入连接页

- [x] `WorkspaceSettingsGitlabConnection.vue` 自建区块内挂载
- [x] 既有连接页测例仍绿 + 显隐集成测例
- 验证：vitest WorkspaceSettingsGitlabConnection

## Task 4 — 文档与价值流图

- [x] intents 已写；更新 `docs/flows/value-stream-test-integration.wsd`

## Task 5 — 浏览器验证 + 登记重启

- [x] 有/无默认机器两种展示（vitest；公网旧包待精准重启）
- [x] `scripts/register-precise-restart.sh taskFE`
