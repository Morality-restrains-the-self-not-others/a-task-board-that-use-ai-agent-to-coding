# 实施计划 — auto_run 交付 push/PR 修复

## 任务

- [x] `shouldSkipAutoRunDelivery` + 失败不写成功 done + retry 计数
- [x] `commitLayerChanges` → `commitLayerGitWorkdirs`
- [x] `autoRunDeliveryHooks` + jobsRuntime / server 补跑 + remirror
- [x] 工作分支解析对齐 bootstrap
- [x] oauth push 模块拆分（行数 / 多仓明细，既有）
- [x] 单测：`autoRunOrchestration` / `autoRunDeliveryHooks` / `layerGitOauthPush`
- [x] 设计/价值流/NFR/DDD/本计划落盘；更新 2026-07-13 design changelog 与 failure #66

## 爆炸半径

- `trae-agent/onlineServiceJS` 容器镜像行为；需重建镜像 + 滚动 VM 才作用于存量任务。
- Django / Go 契约：`pr_base_branch` 已存在，本修复不依赖新字段。

## 验收命令

```bash
cd trae-agent/onlineServiceJS && node --test \
  src/autoRunOrchestration.test.mjs \
  src/autoRunDeliveryHooks.test.mjs \
  src/jobsRuntime.autoRunDelivery.test.mjs \
  src/layerGitOauthPush.test.mjs
```
