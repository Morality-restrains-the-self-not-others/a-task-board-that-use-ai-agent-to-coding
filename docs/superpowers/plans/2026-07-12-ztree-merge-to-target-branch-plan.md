# 实施计划 — zTree 合并到目标分支

## Tasks

- [x] T1 onlineServiceJS：`layerGitMerge.mjs` + `POST …/git/merge` + skill.md + 单测
- [x] T2 taskContainerGateway：L0 `container-layer-git-merge` + 单测
- [x] T3 Django：forward + view + MIGRATED_OUTBOUND_ACTIONS
- [x] T4 前端：canMerge、按钮、事件链、resolveMergeTarget、handler + vitest
- [x] T5 意图文档已写；跑相关测试

## 验证结果（2026-07-12）

- `node --test src/layerGitMerge.test.mjs` — 6 passed
- `go test ./src -run L0RegistryLayerGitMerge` — ok
- vitest layerZtreeNodes / branchUtils / layerActions — 30 passed
